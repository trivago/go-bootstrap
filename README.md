# go-bootstrap

A golang module to reduce common boilerplate code.

This module is shared between many golang tools in trivago and is very
opinionated on the modules used in these tools.
More precisely it expect tools to:

- Use [zerolog](https://github.com/rs/zerolog) for logging
- Be compatible to Google Cloud logs by providing [commonly used fields](https://cloud.google.com/logging/docs/structured-logging#structured_logging_special_fields)
- Use [Gin](https://github.com/gin-gonic/gin) for serving HTTP
- Use [viper](https://github.com/spf13/viper) for configuration

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

This extends the minimal example to let the workload serve HTTP.

```golang
package main

import (
  "github.com/trivago/go-bootstrap/config"
  "github.com/trivago/go-bootstrap/httpserver"
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

### HTTPs server

This example requires valid TLS certificates to be present as files.
The [hack] directory contains some self-signed examples and a [generator script](hack/gen-cert.sh)
for testing purposes.

```golang
package main

import (
  "github.com/trivago/go-bootstrap/config"
  "github.com/trivago/go-bootstrap/httpserver"
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
