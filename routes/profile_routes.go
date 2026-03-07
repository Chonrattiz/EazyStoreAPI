package routes

import (
	UserController "EazyStoreAPI/api/user" // ไฟล์ที่คุณเก็บ UpdateProfile ไว้

	"github.com/gin-gonic/gin"
)

// ProfileRoutes รับ RouterGroup "/api" (ที่ผ่านการกรอง CheckAuth แล้ว) เข้ามาจัดการต่อ
func ProfileRoutes(rg *gin.RouterGroup) {
	// สร้าง Sub-group เพิ่มเป็น /api/profile
	profileGroup := rg.Group("/profile")
	{
	
		profileGroup.PUT("/update", UserController.UpdateProfile) 
		profileGroup.GET("/", UserController.GetProfile)
	}
}