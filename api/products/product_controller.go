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
	// 1. ใช้ตัวแปร product รับค่าตรงๆ เลย (ลดการเขียน map ข้อมูลซ้ำซ้อน)
	var product models.Product

	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//   สร้างรหัสสินค้า product_code (Auto Generate Code)

	var lastProduct models.Product
	var newCode string

	// ค้นหาสินค้า "ตัวล่าสุด" ของร้านนี้
	result := database.DB.Where("shop_id = ?", product.ShopID).Order("product_id desc").First(&lastProduct)

	if result.RowsAffected == 0 {
		// กรณี A: ร้านนี้ยังไม่เคยมีสินค้าเลย -> เริ่มต้นที่ 001
		// รูปแบบ: ps + shop_id + _001 (เช่น ps1_001)
		newCode = fmt.Sprintf("ps%d_001", product.ShopID)
	} else {
		// กรณี B: มีสินค้าอยู่แล้ว (เช่น ps1_005) -> ต้องแกะเลข 5 ออกมาบวก 1

		// แยก String ด้วยเครื่องหมาย "_" (จะได้ ["ps1", "005"])
		parts := strings.Split(lastProduct.ProductCode, "_")

		if len(parts) == 2 {
			// เอาส่วนที่เป็นตัวเลข (parts[1]) มาแปลงเป็น int
			currentNum, err := strconv.Atoi(parts[1])
			if err == nil {
				// ถ้าแปลงสำเร็จ ให้บวก 1
				nextNum := currentNum + 1
				// ประกอบร่างใหม่ (%03d คือเติม 0 ข้างหน้าให้ครบ 3 หลัก เช่น 1 -> 001)
				newCode = fmt.Sprintf("ps%d_%03d", product.ShopID, nextNum)
			}
		}
	}

	// กันเหนียว: ถ้าเกิด error ในการแกะเลข หรือ format เดิมผิดเพี้ยน ให้ตั้งเป็น 001 ใหม่ไปเลย
	if newCode == "" {
		newCode = fmt.Sprintf("ps%d_%s", product.ShopID, "001")
	}

	// ยัดรหัสที่คำนวณเสร็จแล้ว ใส่เข้าไปในตัวแปร product
	product.ProductCode = newCode
	// =========================================================

	// 4. Insert into Database
	// ใช้ตัวแปร product (ที่มี ProductCode แล้ว) บันทึกได้เลย
	if err := database.DB.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถบันทึกสินค้าได้: " + err.Error()})
		return
	}

	// 5. Return Success
	c.JSON(http.StatusOK, gin.H{
		"message": "บันทึกสินค้าสำเร็จ",
		"data":    product,
	})
}
