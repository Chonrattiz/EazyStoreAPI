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



// GetCategories ดึงรายการหมวดหมู่ทั้งหมดจากฐานข้อมูล
func GetCategories(c *gin.Context) {
    var categories []models.Category
    
    // ดึงข้อมูลทั้งหมดจากตาราง category
    if err := database.DB.Order("category_id ASC").Find(&categories).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถดึงข้อมูลหมวดหมู่ได้: " + err.Error()})
        return
    }

    c.JSON(http.StatusOK, categories)
}


// GetProductsByShop godoc
// @Summary      ดึงรายการสินค้าทั้งหมดของร้านค้า
// @Description  ดึงข้อมูลสินค้าทั้งหมดที่ผูกกับ shop_id โดยเรียงจากใหม่ไปเก่า
// @Tags         Product
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        shop_id   query     int  true  "รหัสร้านค้า (Shop ID)"
// @Success      200  {array}   models.Product
// @Failure      400  {object}  map[string]string "กรุณาระบุ shop_id"
// @Failure      500  {object}  map[string]string "ไม่สามารถดึงข้อมูลสินค้าได้"
// @Router       /api/products [get]
func GetProductsByShop(c *gin.Context) {
    // รับ shop_id จาก Query Parameter (เช่น /api/products?shop_id=1)
    shopID := c.Query("shop_id")
    
    if shopID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ shop_id"})
        return
    }

    var products []models.Product
    
    // ใช้ GORM ดึงข้อมูลทั้งหมดโดยกรองตาม shop_id
    // SELECT * FROM products WHERE shop_id = ?
    result := database.DB.Where("shop_id = ?", shopID).Order("product_id DESC").Find(&products)

    if result.Error != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถดึงข้อมูลสินค้าได้: " + result.Error.Error()})
        return
    }

    // ส่งข้อมูลกลับไปในรูปแบบ List
    c.JSON(http.StatusOK, products)
}
