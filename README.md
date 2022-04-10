# OpenWeatherAPI Kubernetes Operator

This operator allows the user to create `weather` resources, 
providing `lat/lon` coordinates. The weather operator will query
the OpenWeatherAPI and provide back the details into the `status`.

### Dependencies

```bash
brew install operator-sdk
```

### Build and run

```bash
make generate
make manifests
make install
make run
```

The kubernetes weather operator should now be running locally, 
attached to your k8s cluster.

### Deploy a weather instance

- Go to https://openweathermap.org and create a free account.
  - Within your account, generate an API token.
- Convert the token into a Base64-encoded string
  - `echo -n <TOKEN> | echo -n "yada"`
- Edit the file `./config/samples/weather_api_secret.yaml`
  - Copy the result into the field `token`
- Upload your new secret to Kubernetes
  - `kubectl create -f ./config/samples/weather_api_secret.yaml`
- Edit the file `./config/samples/weather_v1beta1_weather.yaml`
  - Change the `lat` and `lon` attributes to whatever you desire
- Upload your new weather instance
  - `kubectl create -f ./config/samples/weather_v1beta1_weather.yaml`

Now you can use `kubectl` to list/view/describe your weather instance(s).

```bash
kubectl get weather -n default
kubectl describe weather/sample -n default
``` 

