package routes

import (
	resetController "EazyStoreAPI/api/ResetPassword"
	authController "EazyStoreAPI/api/auth"
	dashboadController "EazyStoreAPI/api/dashboad"
	debtorController "EazyStoreAPI/api/debtor"
	paymentController "EazyStoreAPI/api/payment"
	productController "EazyStoreAPI/api/products"
	saleController "EazyStoreAPI/api/sales"
	shopController "EazyStoreAPI/api/shops"

	"EazyStoreAPI/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// โซน Public (ไม่ต้องใช้ Token)
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authController.Register)
		auth.POST("/login", authController.Login)
		auth.POST("/verify-registration", authController.VerifyRegistration)
		auth.POST("/change-email-verify", authController.ChangeEmailBeforeVerify)

		//  เพิ่มเส้นทางสำหรับกู้รหัสผ่านตรงนี้ครับ
		auth.POST("/request-reset", resetController.RequestResetOTP)
		auth.POST("/verify-otp", resetController.VerifyOTP)
		auth.POST("/reset-password", resetController.UpdatePassword)
	}

	//  โซน Protected (ต้องมี Token เท่านั้นถึงจะเข้าได้)

	protected := r.Group("/api")
	protected.Use(middleware.CheckAuth())
	{

		protected.POST("/createShop", shopController.CreateShop)
		protected.GET("/getShop", shopController.GetShopByUser)
		protected.DELETE("/deleteShop/:shop_id", shopController.DeleteShop)
		protected.PUT("/updateShop/:shop_id", shopController.UpdateShop)

		protected.POST("/createProduct", productController.CreateProduct)
		protected.GET("/categories", productController.GetCategories)
		protected.GET("/products", productController.GetProductsByShop)
		protected.GET("/product/search", productController.GetProductBySearch)
		protected.PUT("product/stock", productController.UpdateStock)
		protected.PUT("/products/:id", productController.UpdateProduct)
		protected.GET("/getNullBarcode", productController.GetNullBarcode)

		protected.POST("/createDebtor", debtorController.CreateDebtor)
		protected.GET("/debtor/search", debtorController.GetDebtorBySearch)
		protected.GET("/debtor", debtorController.GetDebtorByAll)

		protected.POST("/createSale", saleController.CreateSale)
		protected.POST("/createCreditSale", saleController.CreateCreditSale)

		protected.POST("/paymentDebt", paymentController.PaymentDebt)


		protected.GET("/sales/summary", dashboadController.GetSalesSummary)

	}

	return r
}
