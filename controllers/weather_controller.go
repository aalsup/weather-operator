/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	goerrs "errors"
	"fmt"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/record"
	"net/http"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	weatherv1beta1 "alsup/api/v1beta1"
)

const WeatherUrl = "https://api.openweathermap.org/data/2.5/weather"
const UnitFormat = "imperial"
const WeatherAPITimeout = 10 * time.Second
const DefaultRefreshPeriod = "5m"

// WeatherReconciler reconciles a Weather object
type WeatherReconciler struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

type OpenWeatherMapResponse struct {
	Coord struct {
		Lon float64 `json:"lon"`
		Lat float64 `json:"lat"`
	} `json:"coord"`
	Weather []struct {
		Id          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Base string `json:"base"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		TempMin   float64 `json:"temp_min"`
		TempMax   float64 `json:"temp_max"`
		Pressure  int64   `json:"pressure"`
		Humidity  int64   `json:"humidity"`
	} `json:"main"`
	Visibility uint32 `json:"visibility"`
	Wind       struct {
		Speed float64 `json:"speed"`
		Deg   uint16  `json:"deg"`
		Gust  float64 `json:"gust"`
	} `json:"wind"`
	Clouds struct {
		All uint16 `json:"all"`
	} `json:"clouds"`
	DateTime int64 `json:"dt"`
	Sys      struct {
		Type    uint16  `json:"type"`
		Id      uint32  `json:"id"`
		Message float64 `json:"message"`
		Country string  `json:"country"`
		Sumrise uint64  `json:"sunrise"`
		Sunset  uint64  `json:"sunset"`
	} `json:"sys"`
	Timezone int    `json:"timezone"`
	Id       uint32 `json:"id"`
	Name     string `json:"name"`
	Cod      uint16 `json:"cod"`
}

//+kubebuilder:rbac:groups=weather.alsup,resources=weathers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=weather.alsup,resources=weathers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=weather.alsup,resources=weathers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;update;patch

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *WeatherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling weather")

	// get the weather spec
	weather := &weatherv1beta1.Weather{}
	err := r.Client.Get(ctx, req.NamespacedName, weather)
	if err != nil {
		if errors.IsNotFound(err) {
			// instance was likely deleted, between Reconcile and here
			logger.Info("weather instance not found. probably deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to get weather instance")
		return ctrl.Result{}, err
	}
	logger.Info(fmt.Sprintf("got weather spec for lat: %s, lon: %s", weather.Spec.Lat, weather.Spec.Lon))

	// get the referenced secret spec (need to get the OpenWeatherAPI token)
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{Namespace: weather.Namespace, Name: weather.Spec.SecretRef.Name}
	err = r.Client.Get(ctx, secretKey, secret)
	if err != nil {
		errMsg := fmt.Sprintf("Cannot find secret '%s'", weather.Spec.SecretRef.Name)
		logger.Error(err, errMsg)
		r.Recorder.Event(weather, "Failure", "Secret", errMsg)
		return ctrl.Result{}, err
	}

	// build the OpenWeatherMap API URL
	secretBytes, ok := secret.Data["token"]
	if !ok {
		errMsg := fmt.Sprintf("Secret '%s' does not have a 'token' attribute", secretKey)
		logger.Error(nil, errMsg)
		r.Recorder.Event(weather, "Failure", "Secret", errMsg)
		return ctrl.Result{}, err
	}
	apiToken := string(secretBytes)
	url := fmt.Sprintf("%s?lat=%s&lon=%s&units=%s&appid=%s", WeatherUrl, weather.Spec.Lat, weather.Spec.Lon, UnitFormat, apiToken) //logger.Info(fmt.Sprintf("URL: %s", url))
	httpClient := http.Client{Timeout: WeatherAPITimeout}
	resp, err := httpClient.Get(url)
	if err != nil {
		errMsg := "Unable to query weather API"
		logger.Error(err, errMsg)
		r.Recorder.Event(weather, "Failure", "WeatherAPI", errMsg)
		return ctrl.Result{}, err
	}
	if resp.StatusCode != 200 {
		errMsg := fmt.Sprintf("WeatherAPI returned status-code: %d", resp.StatusCode)
		err = goerrs.New(errMsg)
		logger.Error(err, errMsg)
		r.Recorder.Event(weather, "Failure", "WeatherAPI", errMsg)
		return ctrl.Result{}, err
	}
	defer resp.Body.Close()

	// read the OpenWeatherMap response data
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errMsg := "Unable to read JSON weather response"
		logger.Error(err, errMsg)
		r.Recorder.Event(weather, "Failure", "WeatherAPI", errMsg)
		return ctrl.Result{}, err
	}

	// parse the OpenWeatherMap response
	var jResponse OpenWeatherMapResponse
	err = json.Unmarshal(data, &jResponse)
	if err != nil {
		errMsg := "Unable to parse JSON response into OpenWeatherMapResponse"
		logger.Error(err, errMsg)
		r.Recorder.Event(weather, "Failure", "WeatherAPI", errMsg)
		return ctrl.Result{}, err
	}

	// update the weather status
	var dataChanged []string
	sTemp := fmt.Sprintf("%.2f", jResponse.Main.Temp)
	if weather.Status.Temp != sTemp {
		attrib := "Temp"
		prevTemp, err := strconv.ParseFloat(weather.Status.Temp, 64)
		if err == nil {
			if prevTemp < jResponse.Main.Temp {
				attrib += "+"
			} else {
				attrib += "-"
			}
		}
		dataChanged = append(dataChanged, attrib)
		weather.Status.Temp = sTemp
	}
	if weather.Status.Pressure != jResponse.Main.Pressure {
		attrib := "Pressure"
		if weather.Status.Pressure < jResponse.Main.Pressure {
			attrib += "+"
		} else {
			attrib += "-"
		}
		dataChanged = append(dataChanged, attrib)
		weather.Status.Pressure = jResponse.Main.Pressure
	}
	if weather.Status.Humidity != jResponse.Main.Humidity {
		attrib := "Humidity"
		if weather.Status.Humidity < jResponse.Main.Humidity {
			attrib += "+"
		} else {
			attrib += "-"
		}
		dataChanged = append(dataChanged, attrib)
		weather.Status.Humidity = jResponse.Main.Humidity
	}
	sWindSpeed := fmt.Sprintf("%.2f", jResponse.Wind.Speed)
	if weather.Status.WindSpeed != sWindSpeed {
		attrib := "WindSpeed"
		prevVal, err := strconv.ParseFloat(weather.Status.WindSpeed, 64)
		if err == nil {
			if prevVal < jResponse.Wind.Speed {
				attrib += "+"
			} else {
				attrib += "-"
			}
		}
		dataChanged = append(dataChanged, attrib)
		weather.Status.WindSpeed = sWindSpeed
	}
	sWindGust := fmt.Sprintf("%.2f", jResponse.Wind.Gust)
	if weather.Status.WindGust != sWindGust {
		attrib := "WindGust"
		prevVal, err := strconv.ParseFloat(weather.Status.WindGust, 64)
		if err == nil {
			if prevVal < jResponse.Wind.Gust {
				attrib += "+"
			} else {
				attrib += "-"
			}
		}
		dataChanged = append(dataChanged, attrib)
		weather.Status.WindGust = sWindGust
	}
	weather.Status.RefreshTime = time.Unix(jResponse.DateTime, 0).String()
	weather.Status.CountryCode = jResponse.Sys.Country
	weather.Status.LocationName = jResponse.Name
	logger.Info(fmt.Sprintf("got weather response for: %s, %s", weather.Status.LocationName, weather.Status.CountryCode))

	// update the kubernetes status
	err = r.Client.Status().Update(ctx, weather)
	if err != nil {
		logger.Error(err, "Unable to post update to weather")
		return ctrl.Result{}, err
	}

	// record an event if data has changed
	if len(dataChanged) > 0 {
		msg := fmt.Sprintf("Weather changed. [%s]", strings.Join(dataChanged, ", "))
		r.Recorder.Event(weather, corev1.EventTypeNormal, "Updated", msg)
	}

	// schedule the next reconcile
	refreshPeriod := DefaultRefreshPeriod
	if len(weather.Spec.RefreshPeriod) > 0 {
		refreshPeriod = weather.Spec.RefreshPeriod
	}
	nextRun, _ := time.ParseDuration(refreshPeriod)
	logger.Info("Reconcile done", "Temp", jResponse.Main.Temp, "NextRun", nextRun.String())
	return ctrl.Result{RequeueAfter: nextRun}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WeatherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("weather")

	return ctrl.NewControllerManagedBy(mgr).
		For(&weatherv1beta1.Weather{}).
		Complete(r)
}
