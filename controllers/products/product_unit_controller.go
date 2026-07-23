package controllers

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// barcodeInUse ตรวจสอบว่าบาร์โค้ดนี้ถูกใช้ไปแล้วหรือยังในร้านเดียวกัน ทั้งบาร์โค้ดหลัก
// ของสินค้า (products.barcode) และบาร์โค้ดของหน่วยขายเพิ่มเติม (product_units.barcode)
// excludeProductID/excludeUnitID ใช้ตอนแก้ไข เพื่อไม่ให้ชนกับ "ตัวเอง"
func barcodeInUse(shopID int, barcode string, excludeProductID int, excludeUnitID int) (bool, string) {
	var productCount int64
	pq := database.DB.Model(&models.Product{}).Where("shop_id = ? AND barcode = ?", shopID, barcode)
	if excludeProductID != 0 {
		pq = pq.Where("product_id != ?", excludeProductID)
	}
	pq.Count(&productCount)
	if productCount > 0 {
		return true, "บาร์โค้ดนี้ถูกใช้กับสินค้าอื่นอยู่แล้ว"
	}

	var unitCount int64
	uq := database.DB.Table("product_units").
		Joins("JOIN products ON products.product_id = product_units.product_id").
		Where("products.shop_id = ? AND product_units.barcode = ?", shopID, barcode)
	if excludeUnitID != 0 {
		uq = uq.Where("product_units.product_unit_id != ?", excludeUnitID)
	}
	uq.Count(&unitCount)
	if unitCount > 0 {
		return true, "บาร์โค้ดนี้ถูกใช้กับหน่วยขายอื่นอยู่แล้ว"
	}

	return false, ""
}

// CreateProductUnit godoc
// @Summary      เพิ่มหน่วยขายเพิ่มเติมให้สินค้า (เช่น ลัง/แพ็ค)
// @Tags         ProductUnit
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                true  "Product ID"
// @Param        unit  body      models.ProductUnit true  "ข้อมูลหน่วยขาย"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/products/{id}/units [post]
func CreateProductUnit(c *gin.Context) {
	productID := c.Param("id")

	var product models.Product
	if err := database.DB.First(&product, productID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบสินค้ารหัสนี้"})
		return
	}

	var input models.ProductUnit
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pid, _ := strconv.Atoi(productID)
	input.ProductUnitID = 0
	input.ProductID = pid
	input.Status = true

	if input.UnitName == product.Unit {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ชื่อหน่วยขายซ้ำกับหน่วยฐานของสินค้า (" + product.Unit + ")"})
		return
	}

	var dupCount int64
	database.DB.Model(&models.ProductUnit{}).
		Where("product_id = ? AND unit_name = ?", pid, input.UnitName).
		Count(&dupCount)
	if dupCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "สินค้านี้มีหน่วยขายชื่อ \"" + input.UnitName + "\" อยู่แล้ว"})
		return
	}

	if input.Barcode != nil && *input.Barcode != "" {
		if product.Barcode != nil && *product.Barcode == *input.Barcode {
			c.JSON(http.StatusBadRequest, gin.H{"error": "บาร์โค้ดซ้ำกับบาร์โค้ดหลักของสินค้า"})
			return
		}
		if used, msg := barcodeInUse(product.ShopID, *input.Barcode, 0, 0); used {
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}
	}

	if err := database.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถเพิ่มหน่วยขายได้: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "เพิ่มหน่วยขายสำเร็จ", "data": input})
}

// UpdateProductUnit godoc
// @Summary      แก้ไขหน่วยขาย (Partial Update)
// @Tags         ProductUnit
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        unitId  path      int                    true  "Product Unit ID"
// @Param        unit    body      map[string]interface{} true  "ข้อมูลที่จะแก้ (ส่งเฉพาะตัวที่แก้)"
// @Success      200     {object}  map[string]interface{}
// @Router       /api/products/units/{unitId} [put]
func UpdateProductUnit(c *gin.Context) {
	unitID := c.Param("unitId")

	var unit models.ProductUnit
	if err := database.DB.First(&unit, unitID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบหน่วยขายนี้"})
		return
	}

	var product models.Product
	if err := database.DB.First(&product, unit.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบสินค้าของหน่วยขายนี้"})
		return
	}

	var inputMap map[string]interface{}
	if err := c.ShouldBindJSON(&inputMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลไม่ถูกต้อง: " + err.Error()})
		return
	}

	updateData := make(map[string]interface{})
	allowedFields := []string{"unit_name", "conversion_qty", "barcode", "sell_price", "cost_price", "status"}
	for _, field := range allowedFields {
		if val, exists := inputMap[field]; exists {
			updateData[field] = val
		}
	}

	if val, ok := updateData["unit_name"]; ok {
		name, _ := val.(string)
		if name == product.Unit {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ชื่อหน่วยขายซ้ำกับหน่วยฐานของสินค้า (" + product.Unit + ")"})
			return
		}
		var dupCount int64
		database.DB.Model(&models.ProductUnit{}).
			Where("product_id = ? AND unit_name = ? AND product_unit_id != ?", unit.ProductID, name, unit.ProductUnitID).
			Count(&dupCount)
		if dupCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "สินค้านี้มีหน่วยขายชื่อ \"" + name + "\" อยู่แล้ว"})
			return
		}
	}

	if val, ok := updateData["conversion_qty"]; ok {
		conv, _ := val.(float64) // JSON number ผ่าน interface{} จะเป็น float64
		if conv <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "1 หน่วยนี้ต้องแปลงได้มากกว่า 1 หน่วยฐาน"})
			return
		}
	}

	if val, ok := updateData["barcode"]; ok {
		bc, _ := val.(string)
		if bc != "" {
			if product.Barcode != nil && *product.Barcode == bc {
				c.JSON(http.StatusBadRequest, gin.H{"error": "บาร์โค้ดซ้ำกับบาร์โค้ดหลักของสินค้า"})
				return
			}
			if used, msg := barcodeInUse(product.ShopID, bc, 0, unit.ProductUnitID); used {
				c.JSON(http.StatusBadRequest, gin.H{"error": msg})
				return
			}
		}
	}

	if len(updateData) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "ไม่มีข้อมูลเปลี่ยนแปลง", "data": unit})
		return
	}

	if err := database.DB.Model(&unit).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "แก้ไขหน่วยขายไม่สำเร็จ: " + err.Error()})
		return
	}

	var updated models.ProductUnit
	database.DB.First(&updated, unit.ProductUnitID)

	c.JSON(http.StatusOK, gin.H{"message": "แก้ไขหน่วยขายสำเร็จ", "data": updated})
}

// DeleteProductUnit godoc
// @Summary      ลบหน่วยขาย (Smart Delete — เหมือน DeleteProduct)
// @Tags         ProductUnit
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        unitId  path      int  true  "Product Unit ID"
// @Success      200     {object}  map[string]string
// @Router       /api/products/units/{unitId} [delete]
func DeleteProductUnit(c *gin.Context) {
	unitID := c.Param("unitId")

	var unit models.ProductUnit
	if err := database.DB.First(&unit, unitID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบหน่วยขายนี้"})
		return
	}

	var count int64
	database.DB.Table("sale_items").Where("product_unit_id = ?", unit.ProductUnitID).Count(&count)

	if count > 0 {
		if err := database.DB.Model(&unit).Update("status", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถซ่อนหน่วยขายได้: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "หน่วยขายนี้เคยถูกขายแล้ว ระบบได้ทำการซ่อนหน่วยขายแทนการลบ",
			"status":  "hidden",
		})
		return
	}

	if err := database.DB.Delete(&unit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถลบหน่วยขายได้: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ลบหน่วยขายออกจากระบบสำเร็จ", "status": "deleted"})
}
