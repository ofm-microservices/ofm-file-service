package main

import (
	appfx "file-service/internal/fx"

	"go.uber.org/fx"
)

var newApp = fx.New
var runApp = (*fx.App).Run

func main() {
	runApp(newApp(
		appfx.ConfigModule,
		appfx.LoggerModule,
		appfx.AppModule,
		appfx.StorageModule,
		appfx.RepoModule,
		appfx.ServiceModule,
		appfx.PresentationModule,
	))
}
