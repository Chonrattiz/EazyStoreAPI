package routes

import (
	controllers "EazyStoreAPI/api/auth"
	"EazyStoreAPI/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
    r := gin.Default()

    // -----------------------------------------------------
    // 🟢 โซน Public (ไม่ต้องใช้ Token)
    // -----------------------------------------------------
    auth := r.Group("/api/auth")
    {
        auth.POST("/register", controllers.Register)
        auth.POST("/login", controllers.Login)
    }

    // -----------------------------------------------------
    // 🔒 โซน Protected (ต้องมี Token เท่านั้นถึงจะเข้าได้)
    // -----------------------------------------------------
    // สร้างกลุ่ม api ใหม่ แล้วสั่ง Use(middleware.CheckAuth())
    protected := r.Group("/api")
    protected.Use(middleware.CheckAuth()) 
    {
        // ใส่เส้น API ของระบบร้านค้า หรือสินค้า ไว้ในนี้
        // ตัวอย่าง:
        // protected.GET("/myshop", shopController.GetMyShop)
        // protected.POST("/product", productController.CreateProduct)
        
        // ทดสอบระบบ (Test Token)
        protected.GET("/profile", func(c *gin.Context) {
            // ลองดึงค่าที่ Middleware แปะไว้ให้ออกมาดู
            userId, _ := c.Get("user_id")
            username, _ := c.Get("username")
            
            c.JSON(200, gin.H{
                "message": "คุณเข้าสู่โซนปลอดภัยได้แล้ว!",
                "your_id": userId,
                "your_name": username,
            })
        })
    }

    return r
}