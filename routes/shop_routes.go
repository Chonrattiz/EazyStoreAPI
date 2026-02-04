package routes

import (
	controllers "EazyStoreAPI/api/shops"
	

	"github.com/gin-gonic/gin"
)

func ShopRoutes(r *gin.Engine) {
	r.POST("/createShop", controllers.CreateShop)
	// r.GET("/products", controllers.GetProducts)
	// r.PUT("/products/:id", controllers.UpdateProduct)
}
