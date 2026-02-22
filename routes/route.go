package routes

import (
	"EazyStoreAPI/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	//  Public (ไม่ต้องใช้ Token)
	authGroup := r.Group("/api/auth")
	AuthRoutes(authGroup)

	// Protected (ต้องมี Token เท่านั้น)
	protectedGroup := r.Group("/api")
	protectedGroup.Use(middleware.CheckAuth())
	{
		ShopRoutes(protectedGroup)
		ProductRoutes(protectedGroup)
		DebtorRoutes(protectedGroup)
		SaleRoutes(protectedGroup)
		PaymentRoutes(protectedGroup)
		DashboardRoutes(protectedGroup)
	}

	return r
}
