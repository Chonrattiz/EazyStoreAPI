package routes

import (
	// api "EazyStoreAPI/api"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	ProductRoutes(r)

	return r
}
