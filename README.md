# go-bootstrap

A golang module to reduce common boilerplate code.

## Minimal usage

This allows reading configuration flags via `viper`, sets up `zerolog` in a google cloud logging friendly way and makes
the workload CGroup aware.

```golang
package main

import (
  "trivago.com/bootstrap/config"
)

func main() {
  config.Read("CFG","config.yaml")
}
```

## HTTP server

This extends the minimal example to let the workload serve HTTP.

```golang
package main

import (
  "trivago.com/bootstrap/config"
  "trivago.com/bootstrap/httpserver"
	"github.com/spf13/viper"
)

func main() {
  viper.SetDefault("port", 8080)
  config.Read("CFG","config.yaml")

  port := viper.GetInt("port")

  srv := httpserver.New(port, httpserver.AlwaysOk, httpserver.AlwaysOk, nil)
  httpserver.Listen(srv, nil)
}
```

## HTTPs server

This example requires valid TLS certificates to be present as files.
If you use the [app chart](https://github.com/trivago/gcp-shared-artifacts/tree/master/kubernetes/helm/charts/app),
the required certificate files will be generated for you and placed at the paths used here.

```golang
package main

import (
  "trivago.com/bootstrap/config"
  "trivago.com/bootstrap/httpserver"
	"github.com/spf13/viper"
)

func main() {
  viper.SetDefault("port", 8443)
  viper.SetDefault("tls.cert", "/etc/certs/tls.crt")
  viper.SetDefault("tls.key", "/etc/certs/tls.key")
  
  config.Read("CFG","config.yaml")

  srv := httpserver.NewWithConfig(httpserver.Config{
    Port:        viper.GetInt("port"),
    PathTLSCert: viper.GetString("tls.cert"),
    PathTLSKey:  viper.GetString("tls.key"),
  })

  httpserver.Listen(srv, nil)
}
```
