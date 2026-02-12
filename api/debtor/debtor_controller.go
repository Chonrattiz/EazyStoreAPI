package controllers

import (
	"EazyStoreAPI/database"
	"net/http"

	"EazyStoreAPI/models"

	"github.com/gin-gonic/gin"
)

// CreateDebtor godoc
// @Summary      เพิ่มลูกหนี้
// @Description  สร้างลูกหนี้ใหม่ลงในฐานข้อมูล
// @Tags         Debtor
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Debtor body models.Debtor true "ข้อมูลลูกหนี้"
// @Success      200  {object} models.Debtor
// @Failure      400  {object} map[string]string
// @Router       /api/createDebtor [post]
func CreateDebtor(c *gin.Context) {
	var input models.Debtor

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	debtor := models.Debtor{
		DebtorID:    input.DebtorID,
		Name:        input.Name,
		Phone:       input.Phone,
		Address:     input.Address,
		ImgDebtor:   input.ImgDebtor,
		CreditLimit: input.CreditLimit,
		CurrentDebt: input.CurrentDebt,
	}

	if err := database.DB.Create(&debtor).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถบันทึกลูกหนี้ได้: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "บันทึกลูกหนี้สำเร็จ",
		"data":    debtor,
	})
}
