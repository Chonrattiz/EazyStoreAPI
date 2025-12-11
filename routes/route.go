package routes

import (
	handlers "EazyStoreAPI/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// จัดกลุ่ม Route (Group) เช่น /api/v1
	api := r.Group("/api")
	{
		api.GET("/users", handlers.GetUsers)      // GET  http://localhost:8080/api/users
		api.POST("/users", handlers.CreateUser)   // POST http://localhost:8080/api/users
	}

	return r
}