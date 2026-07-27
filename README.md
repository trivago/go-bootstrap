# go-bootstrap

A golang module to reduce common boilerplate code.

This module is shared between many golang tools in trivago and is very
opinionated on the modules used in these tools.
More precisely it expect tools to:

- Use [zerolog](https://github.com/rs/zerolog) for logging
- Be compatible to Google Cloud logs by providing [commonly used fields](https://docs.cloud.google.com/logging/docs/structured-logging#structured_logging_special_fields)
- Use [net/http](https://pkg.go.dev/net/http), [Gin](https://github.com/gin-gonic/gin), or [fasthttp](https://github.com/valyala/fasthttp) for serving HTTP
- Use [viper](https://github.com/spf13/viper) for configuration

## Maintenance and PRs

This repository is in active development but is not our main focus.  
PRs are welcome, but will take some time to be reviewed.

## License

All files in the repository are subject to the [Apache 2.0 License](LICENSE)

## Builds and Releases

All commits to the main branch need to use [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/).  
Releases will be generated automatically from these commits using [Release Please](https://github.com/googleapis/release-please).

### Required tools

All [required tools](flake.nix) can be installed locally via [nix](https://nixos.org/)
and are loaded on demand via [direnv](https://direnv.net/).  
On MacOS you can install nix via the installer from [determinate systems](https://determinate.systems/).

```shell
curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install
```

We provided a [justfile](https://github.com/casey/just) to generate the required `.envrc` file.
Run `just init-nix` to get started, or run the [script](hack/init-nix.sh) directly.

### Running unit-tests

After you have set up your environment, run unittests via `just test` or

```shell
go test ./...
```

## Examples

### Minimal usage

This allows reading configuration flags via `viper`, sets up `zerolog` in a google cloud logging friendly way and makes
the workload CGroup aware.

```golang
package main

import (
  "github.com/trivago/go-bootstrap/config"
)

func main() {
  config.Read("CFG","config.yaml")
}
```

### HTTP server

This extends the minimal example to let the workload serve HTTP with the
standard library. Any framework that implements `http.Handler` can be passed
the same way. See [MIGRATION.md](MIGRATION.md) for upgrade notes.

```golang
package main

import (
  "log"
  "net/http"

  "github.com/trivago/go-bootstrap/config"
  "github.com/trivago/go-bootstrap/httpserver"
  "github.com/spf13/viper"
)

func main() {
  viper.SetDefault("port", 8080)
  config.Read("CFG","config.yaml")

  mux := http.NewServeMux()
  mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
  })

  srv, err := httpserver.New(
    viper.GetInt("port"),
    httpserver.CheckOK,
    httpserver.CheckOK,
    mux,
  )
  if err != nil {
    log.Fatal(err)
  }

  httpserver.Listen(srv, nil)
}
```

### Gin server

This creates a Gin engine inside the package and registers routes through an
init callback. Existing Gin handlers and probe functions stay valid.

```golang
package main

import (
  "log"

  "github.com/gin-gonic/gin"
  "github.com/trivago/go-bootstrap/config"
  "github.com/trivago/go-bootstrap/httpserver"
  "github.com/spf13/viper"
)

func main() {
  viper.SetDefault("port", 8080)
  config.Read("CFG","config.yaml")

  srv, err := httpserver.NewGin(
    viper.GetInt("port"),
    httpserver.AlwaysOk,
    httpserver.AlwaysOk,
    func(router *gin.Engine) {
      router.GET("/", func(c *gin.Context) {
        c.Status(200)
      })
    },
  )
  if err != nil {
    log.Fatal(err)
  }

  httpserver.Listen(srv, nil)
}
```

### FastHTTP server

This uses a native fasthttp server with the same probes, logging, recovery,
and TLS options.

```golang
package main

import (
  "log"

  "github.com/trivago/go-bootstrap/config"
  "github.com/trivago/go-bootstrap/httpserver"
  "github.com/spf13/viper"
  "github.com/valyala/fasthttp"
)

func main() {
  viper.SetDefault("port", 8080)
  config.Read("CFG","config.yaml")

  handler := func(ctx *fasthttp.RequestCtx) {
    ctx.SetStatusCode(fasthttp.StatusOK)
  }

  srv, err := httpserver.NewFastHTTP(
    viper.GetInt("port"),
    httpserver.CheckOK,
    httpserver.CheckOK,
    handler,
  )
  if err != nil {
    log.Fatal(err)
  }

  httpserver.Listen(srv, nil)
}
```

### HTTPs server

This example requires valid TLS certificates to be present as files.
The [hack] directory contains some self-signed examples and a [generator script](hack/gen-cert.sh)
for testing purposes.

```golang
package main

import (
  "log"
  "net/http"

  "github.com/trivago/go-bootstrap/config"
  "github.com/trivago/go-bootstrap/httpserver"
  "github.com/spf13/viper"
)

func main() {
  viper.SetDefault("port", 8443)
  viper.SetDefault("tls.cert", "/etc/certs/tls.crt")
  viper.SetDefault("tls.key", "/etc/certs/tls.key")

  config.Read("CFG","config.yaml")

  srv, err := httpserver.NewWithConfig(httpserver.Config{
    Port:        viper.GetInt("port"),
    PathTLSCert: viper.GetString("tls.cert"),
    PathTLSKey:  viper.GetString("tls.key"),
  }, http.NewServeMux())
  if err != nil {
    log.Fatal(err)
  }

  httpserver.Listen(srv, nil)
}
```
