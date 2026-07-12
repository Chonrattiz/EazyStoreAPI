package routes

import (
	"EazyStoreAPI/middleware"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

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
		ProfileRoutes(protectedGroup)
		OrderListRoutes(protectedGroup)
	}

	return r
}
