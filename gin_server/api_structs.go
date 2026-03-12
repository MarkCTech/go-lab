package gin_api

import (
	"embed"

	"github.com/gin-gonic/gin"
)

type App struct {
	Router   *gin.Engine
	StaticFS embed.FS
}
