# OpenWeatherAPI Kubernetes Operator

This operator allows the user to create `weather` resources, 
providing `lat/lon` coordinates. The weather operator will query
the OpenWeatherAPI and provide back the details into the `status`.

### Dependencies

- MacOS - install HomeBrew
- Ubuntu - install HomeBrew (https://brew.sh/)

```bash
brew install operator-sdk
brew install kubernetes-cli
brew install go@1.17
brew install kind
```

### Build and run

Ubuntu notes

```bash
export CGO_ENABLED=0 
```

MacOS and Ubuntu

Be sure go v1.17 is used for the build

```bash
export PATH=/usr/local/opt/go@1.17/bin:$PATH
```

```bash
make generate
make manifests
make install
make run
```

The kubernetes weather operator should now be running locally, 
attached to your k8s cluster.

### Deploy a weather object (instance)

- Go to https://openweathermap.org and create a free account.
  - Within your account, generate an API token.
- Create a secret that contains your WeatherAPI token
  - `kubectl create secret generic weather-api-secret --from-literal=token=<YOUR-SECRET-TOKEN>`
- Edit the file `./config/samples/weather_v1beta1_weather.yaml`
  - Change the `lat` and `lon` attributes to whatever you desire
- Upload your new weather instance
  - `kubectl create -f ./config/samples/weather_v1beta1_weather.yaml`

Now you can use `kubectl` to list/view/describe your weather instance(s).

```bash
kubectl get weather -n default
kubectl describe weather/sample -n default
``` 

