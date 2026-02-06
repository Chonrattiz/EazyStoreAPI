package controllers

import (
	"EazyStoreAPI/database"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"EazyStoreAPI/models"

	"github.com/gin-gonic/gin"
)

// CreateProduct godoc
// @Summary      เพิ่มสินค้า
// @Description  สร้างรายการสินค้าใหม่ลงในฐานข้อมูล
// @Tags         Product
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        product body models.Product true "ข้อมูลสินค้า"
// @Success      200  {object} models.Product
// @Failure      400  {object} map[string]string
// @Router       /api/createProduct [post]
func CreateProduct(c *gin.Context) {

	var input models.Product

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//   สร้างรหัสสินค้า product_code
	var lastProduct models.Product
	var newCode string

	// ค้นหาสินค้า ล่าสุด ของร้านนี้
	result := database.DB.Where("shop_id = ?", input.ShopID).Order("product_id desc").First(&lastProduct)

	if result.RowsAffected == 0 {
		// fmt.Sprintf คือ ผสมข้อความ String Formatting
		newCode = fmt.Sprintf("ps%d_001", input.ShopID)
	} else {
		// แยก String ด้วยเครื่องหมาย "_" (จะได้ ["ps1", "005"])

		parts := strings.Split(lastProduct.ProductCode, "_")

		if len(parts) == 2 {
			// แปลงจาก string เป็น int และ ผลลัพธ์ จาก "005" กลายเป็นเลข 5
			// Atoi ย่อมาจาก ASCII to Integer
			currentNum, err := strconv.Atoi(parts[1])

			if err == nil {
				nextNum := currentNum + 1
				// แล้วมาผสมข้อความใหม่
				newCode = fmt.Sprintf("ps%d_%03d", input.ShopID, nextNum)
			}
		}
	}

	// กันพลาด ถ้าเกิด error ในการถอดเลข หรือ format เดิมผิดเพี้ยน ให้ตั้งเป็น 001 ใหม่ไปเลย
	if newCode == "" {
		newCode = fmt.Sprintf("ps%d_%s", input.ShopID, "001")
	}

	input.ProductCode = newCode

	// Insert into Database
	if err := database.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถบันทึกสินค้าได้: " + err.Error()})
		return
	}

	//  Return Success
	c.JSON(http.StatusOK, gin.H{
		"message": "บันทึกสินค้าสำเร็จ",
		"data":    input,
	})
}
