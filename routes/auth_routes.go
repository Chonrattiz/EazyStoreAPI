package routes

import (
	resetController "EazyStoreAPI/controllers/ResetPassword"
	authController "EazyStoreAPI/controllers/auth"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(rg *gin.RouterGroup) {
	rg.POST("/register", authController.Register)
	rg.POST("/login", authController.Login)
	rg.POST("/verify-registration", authController.VerifyRegistration)
	rg.POST("/change-email-verify", authController.ChangeEmailBeforeVerify)

	// กู้รหัสผ่าน
	rg.POST("/request-reset", resetController.RequestResetOTP)
	rg.POST("/verify-otp", resetController.VerifyOTP)
	rg.POST("/reset-password", resetController.UpdatePassword)
}