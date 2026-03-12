package gin_api

import (
	"embed"

	"github.com/gin-gonic/gin"
)

// This will not run on it's own. You must
// Build & Embed Angular in same Dir as this import,
// Then "//go:embed static/*"
// Then follow steps below

var staticFS embed.FS

func Start() {
	var app App
	InputApp = &app

	app.Router = gin.Default()
	app.StaticFS = staticFS

	AddStaticRoutes()
	app.Router.Run("localhost:5000")
}
