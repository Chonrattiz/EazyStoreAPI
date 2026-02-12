package controllers

import (
	"EazyStoreAPI/database"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"EazyStoreAPI/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

	var products []models.Product

	// ✨ ใช้ Preload("Category") เพื่อจอยเอาข้อมูลชื่อหมวดหมู่มาแสดง
	result := database.DB.Preload("Category").Where("shop_id = ?", shopID).Order("product_id DESC").Find(&products)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถดึงข้อมูลสินค้าได้: " + result.Error.Error()})
		return
	}

	// ส่งข้อมูลกลับไปในรูปแบบ List
	c.JSON(http.StatusOK, products)
}

// GetProductBySearch godoc
// @Summary      ค้นหาสินค้า (Search Product)
// @Description  ค้นหาสินค้าด้วย Keyword (รองรับทั้ง Barcode, Product Code และชื่อสินค้า)
// @Tags         Product
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        keyword    query    string   true   "คำค้นหา (ระบุ Barcode, รหัส หรือ ชื่อ)"
// @Success      200  {object}  models.Product
// @Failure      400  {object}  map[string]string "Bad Request"
// @Failure      404  {object}  map[string]string "Product not found"
// @Router       /api/product/search [get]
func GetProductBySearch(c *gin.Context) {
	// รับค่า keyword ตัวเดียวพอ สำหรับการค้นหาแบบครอบจักรวาล
	keyword := c.Query("keyword")

	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุคำค้นหา"})
		return
	}

	var product models.Product

	// ค้นหาใน product_code หรือ barcode หรือ name
	// ใช้ Preload("Category") เพื่อดึงชื่อหมวดหมู่มาด้วย (ตามที่คุณต้องการใน Frontend)
	result := database.DB.Preload("Category").
		Where("product_code = ? OR barcode = ? OR name LIKE ?", keyword, keyword, "%"+keyword+"%").
		First(&product)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบสินค้า"})
		return
	}

	c.JSON(http.StatusOK, product)
}

// UpdateStock godoc
// @Summary      อัปเดตสต็อกสินค้า (Update Product Stock)
// @Description  บันทึกยอดสต็อกสินค้าล่าสุด
// @Tags         Product
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body models.UpdateStockRequest true "ข้อมูลอัปเดตสต็อก"
// @Success      200  {object}  map[string]interface{} "message: success"
// @Failure      400  {object}  map[string]string "Invalid input"
// @Failure      500  {object}  map[string]string "Update failed"
// @Router       /api/product/stock [put]
func UpdateStock(c *gin.Context) {
	var input models.UpdateStockRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลไม่ถูกต้อง"})
		return
	}

	//  อัปเดตแบบ "บวกเพิ่ม" (Atomic Update)
	// ใช้ gorm.Expr("stock + ?", input.Stock) แทนการใส่ค่าตรงๆ
	result := database.DB.Model(&models.Product{}).
		Where("product_id = ?", input.ProductID).
		Update("stock", gorm.Expr("stock + ?", input.Stock))

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "อัปเดตไม่สำเร็จ: " + result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบสินค้ารหัสนี้"})
		return
	}

	// ดึงค่าล่าสุดมาโชว์
	var updatedProduct models.Product
	database.DB.Select("stock").First(&updatedProduct, input.ProductID)

	c.JSON(http.StatusOK, gin.H{
		"message":       "เพิ่มสต็อกสินค้าเรียบร้อย",
		"product_id":    input.ProductID,
		"added_amount":  input.Stock,          // จำนวนที่เติมเข้าไป
		"current_stock": updatedProduct.Stock, // ยอดคงเหลือล่าสุดใน DB
	})
}

// UpdateProduct godoc
// @Summary      แก้ไขข้อมูลสินค้า (Partial Update)
// @Description  อัปเดตเฉพาะฟิลด์ที่ส่งมา และบันทึกประวัติราคาอัตโนมัติ (Manual Log)
// @Tags         Product
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int             true  "Product ID"
// @Param        product  body      map[string]interface{}  true  "ข้อมูลสินค้า (ส่งเฉพาะตัวที่แก้)"
// @Success      200      {object}  models.Product
// @Router       /api/products/{id} [put]
func UpdateProduct(c *gin.Context) {
	// 1. รับ ID จาก URL
	productID := c.Param("id")

	// 2. ค้นหาสินค้าเดิมก่อน (จำเป็นต้องมีค่าเดิมเพื่อเทียบราคาเก่า)
	var product models.Product
	if err := database.DB.First(&product, productID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบสินค้ารหัสนี้"})
		return
	}

	// 3. รับข้อมูลเป็น Map (เพื่อดูว่าเขาส่งฟิลด์ไหนมาบ้าง)
	// การใช้ Map ช่วยให้รู้ว่า User ส่ง key ไหนมา ถ้าไม่ส่ง key ไหนมา map จะไม่มีค่านั้น
	var inputMap map[string]interface{}
	if err := c.ShouldBindJSON(&inputMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลไม่ถูกต้อง: " + err.Error()})
		return
	}

	// 4. กรองข้อมูล (White-list)
	updateData := make(map[string]interface{})
	allowedFields := []string{
		"name", "category_id", "barcode", "img_product",
		"sell_price", "cost_price", "unit", "status",
		// ❌ ไม่ใส่ "stock" ในนี้ เพื่อป้องกันการแก้ไขสต็อกผ่านหน้านี้
	}

	for _, field := range allowedFields {
		if val, exists := inputMap[field]; exists {
			updateData[field] = val
		}
	}

	if len(updateData) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "ไม่มีข้อมูลเปลี่ยนแปลง", "data": product})
		return
	}

	// ---------------------------------------------------------
	// ✨ ส่วนที่เพิ่ม: Manual Trigger (เขียน Logic เก็บ Log ด้วย Go)
	// เพราะ Database ไม่อนุญาตให้สร้าง Trigger เราเลยทำเองตรงนี้เลย
	// ---------------------------------------------------------

	// 1. เช็คราคาขาย (Sell Price)
	if val, ok := updateData["sell_price"]; ok {
		// แปลงค่าเป็น float64 เพื่อเปรียบเทียบ
		newPrice, _ := val.(float64)
		oldPrice := product.SellPrice

		if newPrice != oldPrice {
			// บันทึกลงตาราง sell_price_logs
			go database.DB.Exec(`
                INSERT INTO sell_price_logs (product_id, sell_price_old, sell_price_new) 
                VALUES (?, ?, ?)`,
				product.ProductID, oldPrice, newPrice,
			)
		}
	}

	// 2. เช็คราคาต้นทุน (Cost Price)
	if val, ok := updateData["cost_price"]; ok {
		newCost, _ := val.(float64)
		oldCost := product.CostPrice

		if newCost != oldCost {
			// บันทึกลงตาราง cost_price_logs
			go database.DB.Exec(`
                INSERT INTO cost_price_logs (product_id, cost_price_old, cost_price_new) 
                VALUES (?, ?, ?)`,
				product.ProductID, oldCost, newCost,
			)
		}
	}
	// ---------------------------------------------------------

	// 5. สั่งอัปเดตลงฐานข้อมูล
	// GORM จะอัปเดตเฉพาะคอลัมน์ที่มีใน map updateData เท่านั้น
	// คอลัมน์ไหนไม่อยู่ใน map จะคงค่าเดิมไว้ (Partial Update สมบูรณ์แบบ)
	if err := database.DB.Model(&product).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "แก้ไขข้อมูลไม่สำเร็จ: " + err.Error()})
		return
	}

	// 6. ดึงข้อมูลล่าสุดมาแสดงผล (พร้อม Join Category เพื่อความสวยงาม)
	// ต้องดึงใหม่เพราะค่าในตัวแปร product เก่ายังไม่อัปเดต
	var updatedProduct models.Product
	database.DB.Preload("Category").First(&updatedProduct, productID)

	c.JSON(http.StatusOK, gin.H{
		"message": "แก้ไขข้อมูลสำเร็จ",
		"data":    updatedProduct, // ส่งข้อมูลที่มีชื่อหมวดหมู่กลับไป
	})
}

// GetNullBarcode godoc
// @Summary      ดูสินค้าไม่มีบาร์โค้ด
// @Description  ดึงรายการสินค้าที่ไม่มีบาร์โค้ดแยกตามร้านค้า
// @Tags         Product
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        shop_id   query     int    true  "รหัสร้านค้า (Shop ID)"
// @Param        category_id  query    int     false "รหัสหมวดหมู่สินค้า (Category ID)"
// @Success      200  {array}   models.Product
// @Failure      404  {object}  map[string]string "Product not found"
// @Failure      500  {object}  map[string]string "Internal Error"
// @Router       /api/getNullBarcode [get]
func GetNullBarcode(c *gin.Context) {
	shopID := c.Query("shop_id")
	categoryID := c.Query("category_id")

	if shopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ shop_id"})
		return
	}

	var products []models.Product
	query := database.DB.Where("shop_id = ? AND barcode IS NULL", shopID)

	// ถ้ามีการส่ง category_id มา ให้เพิ่มเงื่อนไขการค้นหา
	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	err := query.Order("category_id ASC").Find(&products).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ดึงข้อมูลล้มเหลว: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}
