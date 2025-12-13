package routes

import (
	controllers "EazyStoreAPI/api/products"

	"github.com/gin-gonic/gin"
)

func ProductRoutes(r *gin.Engine) {
	r.GET("/products", controllers.GetProducts)
	r.PUT("/products/:id", controllers.UpdateProduct)
}
