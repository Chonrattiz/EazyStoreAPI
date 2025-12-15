package routes

import (
	controllers "EazyStoreAPI/api/auth"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
    r := gin.Default()

    // สร้างกลุ่ม API (เช่น localhost:8080/api/auth/register)
    auth := r.Group("/api/auth")
    {
        auth.POST("/register", controllers.Register) // เส้นสมัครสมาชิก
		 auth.POST("/login", controllers.Login)
    }

    return r
}