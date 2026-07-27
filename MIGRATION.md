# Upgrade guide for httpserver

This guide explains the breaking change in the `httpserver` package.
Use this guide when you move from the Gin-only API to the new API.

---

## What changed

The previous API created a Gin engine inside the package and accepted only Gin.
The new API supports three constructor pairs:

- `NewGin` / `NewGinWithConfig` for Gin applications (smallest change)
- `New` / `NewWithConfig` for any `net/http.Handler`
- `NewFastHTTP` / `NewFastHTTPWithConfig` for native fasthttp handlers

The package still provides these shared features:

- `/healthz` and `/readyz` probes
- access logs
- panic recovery
- TLS certificate reload
- signal handling through `Listen`

---

## API map

| Previous API | New Gin path |
|---|---|
| `New(port, health, ready, initRoutes)` | `NewGin(port, health, ready, initRoutes)` |
| `NewWithConfig(Config{..., InitRoutes: f})` | `NewGinWithConfig(GinConfig{..., InitRoutes: f})` |
| `Config` with Gin fields | `GinConfig` |
| `Health` / `Ready` as `gin.HandlerFunc` | unchanged on `GinConfig` |
| `AlwaysOk` as Gin handler | unchanged |
| `Listen(*http.Server, ...)` | `Listen(Server, ...)` |
| return type `*http.Server` | return type `*HTTPServer` |

For `New` and `NewFastHTTP`, probes use `Check`:

```go
type Check func(ctx context.Context) error
```

Use `CheckOK` when a Check must always succeed.
If `Check` is `nil`, the probe returns HTTP 200.
If `Check` returns an error, the probe returns HTTP 503.

---

## Upgrade procedure for Gin applications

Do these steps in order.

1. Update the `github.com/trivago/go-bootstrap` module version in `go.mod`.
2. Run `go mod tidy`.
3. Replace `New(` with `NewGin(` when the call uses Gin handlers.
4. Replace `NewWithConfig` with `NewGinWithConfig`.
5. Rename `httpserver.Config` to `httpserver.GinConfig` in those calls.
6. Keep `InitRoutes`, `Health`, `Ready`, and `AlwaysOk` unchanged.
7. Keep `httpserver.Listen(srv, signalHandler)`.
8. Build the application.
9. Run the tests for the HTTP endpoints.

---

## Example: probes only

This pattern matches `gcp-github-runners` command `github-downscaler`.

### Before

```go
srv, err := httpserver.New(
	viper.GetInt("port"),
	httpserver.AlwaysOk,
	httpserver.AlwaysOk,
	nil,
)
if err != nil {
	log.Fatal().Err(err).Msg("Failed to create HTTP server")
}
httpserver.Listen(srv, nil)
```

### After

```go
srv, err := httpserver.NewGin(
	viper.GetInt("port"),
	httpserver.AlwaysOk,
	httpserver.AlwaysOk,
	nil,
)
if err != nil {
	log.Fatal().Err(err).Msg("Failed to create HTTP server")
}
httpserver.Listen(srv, nil)
```

---

## Example: Gin routes

This pattern matches `gcp-github-runners` commands `github-workflow-watcher`
and `github-workflow-metrics`.

### Before

```go
srv, err := httpserver.NewWithConfig(httpserver.Config{
	Port:                viper.GetInt("port"),
	Health:              httpserver.AlwaysOk,
	Ready:               httpserver.AlwaysOk,
	DisableAccessLogFor: []string{"/healthz", "/readyz", "/metrics"},
	InitRoutes: func(router *gin.Engine) {
		router.POST("/workflow_job", handleWorkflowJobEvent)
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	},
})
if err != nil {
	log.Fatal().Err(err).Msg("Failed to create HTTP server")
}
httpserver.Listen(srv, nil)
```

### After

```go
srv, err := httpserver.NewGinWithConfig(httpserver.GinConfig{
	Port:                viper.GetInt("port"),
	Health:              httpserver.AlwaysOk,
	Ready:               httpserver.AlwaysOk,
	DisableAccessLogFor: []string{"/healthz", "/readyz", "/metrics"},
	InitRoutes: func(router *gin.Engine) {
		router.POST("/workflow_job", handleWorkflowJobEvent)
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	},
})
if err != nil {
	log.Fatal().Err(err).Msg("Failed to create HTTP server")
}
httpserver.Listen(srv, nil)
```

---

## Example: custom Gin health handler

This pattern matches `gcp-github-runners` command `groot`.
Keep the existing `HealthProbe` Gin handler.
The probe status codes stay the same, including HTTP 410.

### Before

```go
srv, err := httpserver.NewWithConfig(httpserver.Config{
	Port:                viper.GetInt("port"),
	Health:              HealthProbe,
	Ready:               httpserver.AlwaysOk,
	DisableAccessLogFor: []string{"/healthz", "/readyz", "/metrics"},
	InitRoutes: func(router *gin.Engine) {
		router.GET("/runz", RunStatus)
		router.POST("/kill", Kill)
		router.GET("/decommission", DecommissionStatus)
		router.POST("/decommission", Decommission)
		router.GET("/kill", Kill)
		router.GET("/metrics", MetricsDecoy)
	},
})
```

### After

```go
srv, err := httpserver.NewGinWithConfig(httpserver.GinConfig{
	Port:                viper.GetInt("port"),
	Health:              HealthProbe,
	Ready:               httpserver.AlwaysOk,
	DisableAccessLogFor: []string{"/healthz", "/readyz", "/metrics"},
	InitRoutes: func(router *gin.Engine) {
		router.GET("/runz", RunStatus)
		router.POST("/kill", Kill)
		router.GET("/decommission", DecommissionStatus)
		router.POST("/decommission", Decommission)
		router.GET("/kill", Kill)
		router.GET("/metrics", MetricsDecoy)
	},
})
```

---

## Example: Gin handlers from other modules

Register those handlers inside `InitRoutes`:

```go
srv, err := httpserver.NewGinWithConfig(httpserver.GinConfig{
	Port:   viper.GetInt("port"),
	Health: httpserver.AlwaysOk,
	Ready:  httpserver.AlwaysOk,
	InitRoutes: func(router *gin.Engine) {
		hook := kube.AdmissionRequestHook{ /* ... */ }
		router.POST("/validate", hook.Handle)
	},
})
if err != nil {
	log.Fatal().Err(err).Msg("Failed to create HTTP server")
}
httpserver.Listen(srv, nil)
```

---

## Optional: net/http and fasthttp

Use `New` / `NewWithConfig` when the application builds a `net/http.Handler`.
Use `CheckOK` for always-successful Check probes.

```go
srv, err := httpserver.New(
	viper.GetInt("port"),
	httpserver.CheckOK,
	httpserver.CheckOK,
	http.NewServeMux(),
)

srv, err = httpserver.NewWithConfig(httpserver.Config{
	Port:        viper.GetInt("port"),
	PathTLSCert: viper.GetString("tls.cert"),
	PathTLSKey:  viper.GetString("tls.key"),
}, http.NewServeMux())
```

Use `NewFastHTTP` / `NewFastHTTPWithConfig` when the application uses fasthttp
handlers.

```go
srv, err := httpserver.NewFastHTTP(
	viper.GetInt("port"),
	httpserver.CheckOK,
	httpserver.CheckOK,
	handler,
)
```

---

## Access to the native server

`New`, `NewWithConfig`, `NewGin`, and `NewGinWithConfig` return `*HTTPServer`.
That type holds `*http.Server` in the `Server` field.

`NewFastHTTP` and `NewFastHTTPWithConfig` return `*FastHTTPServer`.
That type holds `*fasthttp.Server` in the `Server` field.

Change fields on the native server before you call `Listen` when you need them.

---

## Verification procedure

1. Start the application.
2. Call `GET /healthz` and confirm HTTP 200 for a healthy process.
3. Call `GET /readyz` and confirm the expected status.
4. Call each application route and confirm the previous behavior.
5. Send SIGTERM and confirm a clean shutdown in the logs.
