package main

import (
	"embed"

	"github.com/gin-gonic/gin"
	"github.com/martoranam/gin_api"
)

//go:embed static/*
var staticFS embed.FS

func main() {
	app := gin_api.App{StaticFS: staticFS, Router: gin.Default()}

	gin_api.InputApp = &app
	gin_api.AddStaticRoutes()
	app.Router.SetTrustedProxies(nil)
	app.Router.Run("localhost:5000")
}
