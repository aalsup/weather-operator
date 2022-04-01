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
	"fmt"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	weatherv1beta1 "alsup/api/v1beta1"
)

// WeatherReconciler reconciles a Weather object
type WeatherReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=weather.alsup,resources=weathers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=weather.alsup,resources=weathers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=weather.alsup,resources=weathers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Weather object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *WeatherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling weather")

	weather := &weatherv1beta1.Weather{}
	err := r.Get(ctx, req.NamespacedName, weather)
	if err != nil {
		logger.Error(err, "failed to get weather instance")
		return ctrl.Result{}, err
	}

	const WeatherUrl = "https://api.openweathermap.org/data/2.5/weather"
	const UnitFormat = "imperial"
	url := fmt.Sprintf("%s?lat=%s&lon=%s&units=%s&appid=%s", WeatherUrl, weather.Spec.Lat, weather.Spec.Lon, UnitFormat, weather.Spec.ApiKey)
	//logger.Info(fmt.Sprintf("URL: %s", url))
	resp, err := http.Get(url)
	if err != nil {
		logger.Error(err, "failed to get weather info")
		return ctrl.Result{}, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err, "failed to read JSON weather response")
		return ctrl.Result{}, err
	}
	//logger.Info(fmt.Sprintf("Got weather response: %s", data))
	var mapData map[string]interface{}
	err = json.Unmarshal(data, &mapData)
	if err != nil {
		logger.Error(err, "failed to parse JSON weather response")
		return ctrl.Result{}, err
	}
	// extract top-level data
	epochTime := mapData["dt"].(int64)
	locName := mapData["name"].(string)
	// extract 'main' data
	mainData := mapData["main"].(map[string]interface{})
	var temp float64 = 0
	if _, ok := mainData["temp"]; ok {
		temp = mainData["temp"].(float64)
		weather.Status.Temp = fmt.Sprintf("%.2f", temp)
	} else {
		weather.Status.Temp = ""
	}
	var pressure int64 = 0
	if _, ok := mainData["pressure"]; ok {
		pressure = mainData["pressure"].(int64)
		weather.Status.Pressure = pressure
	} else {
		weather.Status.Pressure = 0
	}
	var humidity int64 = 0
	if _, ok := mainData["humidity"]; ok {
		humidity = mainData["humidity"].(int64)
		weather.Status.Humidity = humidity
	} else {
		weather.Status.Humidity = 0
	}
	// extract 'wind' data
	windData := mapData["wind"].(map[string]interface{})
	var windSpeed float64 = 0
	if _, ok := windData["speed"]; ok {
		windSpeed = windData["speed"].(float64)
		weather.Status.WindSpeed = fmt.Sprintf("%.2f", windSpeed)
	} else {
		weather.Status.WindSpeed = ""
	}
	var windGust float64 = 0
	if _, ok := windData["gust"]; ok {
		windGust = windData["gust"].(float64)
		weather.Status.WindGust = fmt.Sprintf("%.2f", windGust)
	} else {
		weather.Status.WindGust = ""
	}
	// extract 'sys' data
	sysData := mapData["sys"].(map[string]interface{})
	country := sysData["country"].(string)
	// update the weather status
	t := time.Unix(epochTime, 0)
	weather.Status.RefreshTime = t.String()
	weather.Status.CountryCode = country
	weather.Status.LocationName = locName
	err = r.Status().Update(ctx, weather)
	if err != nil {
		logger.Error(err, "failed to update weather status")
		return ctrl.Result{}, err
	}
	nextRun, _ := time.ParseDuration("5m")
	return ctrl.Result{RequeueAfter: nextRun}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WeatherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&weatherv1beta1.Weather{}).
		Complete(r)
}
