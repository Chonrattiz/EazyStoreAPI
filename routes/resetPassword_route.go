package routes

import (
	controllers "EazyStoreAPI/api/ResetPassword"

	"github.com/gin-gonic/gin"
)

    func ResetRouter() *gin.Engine {
        r := gin.Default()


        auth := r.Group("/api")
        {
           
            auth.POST("/request-reset", controllers.RequestResetOTP)
        }

     

        return r
    }