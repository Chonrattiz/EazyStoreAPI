package routes

import (
	UserController "EazyStoreAPI/controllers/user" // ไฟล์ที่คุณเก็บ UpdateProfile ไว้

	"github.com/gin-gonic/gin"
)

// ProfileRoutes รับ RouterGroup "/api" (ที่ผ่านการกรอง CheckAuth แล้ว) เข้ามาจัดการต่อ
func ProfileRoutes(rg *gin.RouterGroup) {
	rg.GET("/profile", UserController.GetProfile)    // ดูข้อมูลโปรไฟล์
	rg.PUT("/profile", UserController.UpdateProfile) // แก้ไขข้อมูลโปรไฟล์
}