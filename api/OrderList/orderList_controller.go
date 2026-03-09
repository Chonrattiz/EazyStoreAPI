package controllers

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
)

func ExportOrderPDF(c *gin.Context) {
	var req models.ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. สร้าง Struct เฉพาะกิจสำหรับรับข้อมูลที่ Join กัน
	var result struct {
		Name    string // จาก Shop.Name
		Address string // จาก Shop.Address
		Phone   string // จาก User.Phone (เบอร์เจ้าของร้าน)
	}

	// 2. Query ดึงข้อมูลร้านค้า Join กับตาราง Users
	// เชื่อมด้วย shops.user_id = users.user_id
	err := database.DB.Table("shops").
		Select("shops.name, shops.address, users.phone").
		Joins("left join users on users.user_id = shops.user_id").
		Where("shops.shop_id = ?", req.ShopID).
		First(&result).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบข้อมูลร้านค้าหรือเจ้าของร้าน"})
		return
	}

	// 3. เริ่มสร้าง PDF
	pdf := gofpdf.New("P", "mm", "A4", "")

	// --- 🟢 ส่วนการตั้งค่าภาษาไทย ---
	pdf.AddUTF8Font("THSarabun", "", "assets/fonts/THSarabunNew.ttf")
	pdf.AddUTF8Font("THSarabun", "B", "assets/fonts/THSarabunNew_Bold.ttf")

	pdf.AddPage()
	pdf.SetMargins(15, 15, 15)

	// --- 🔵 ส่วน Header (ใช้ข้อมูลจากตัวแปร result ที่ Join มาแล้ว) ---
	pdf.SetFont("THSarabun", "B", 22)
	pdf.Cell(120, 10, result.Name) // ชื่อร้านจากตาราง shops

	pdf.SetFont("THSarabun", "B", 16)
	pdf.CellFormat(0, 10, "รายงานการสั่งซื้อ", "", 1, "R", false, 0, "")

	pdf.SetFont("THSarabun", "", 12)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(120, 6, result.Address) // ที่อยู่จากตาราง shops

	currentTime := time.Now().Format("02 ม.ค. 2006 | 15:04")
	pdf.CellFormat(0, 6, "วันที่: "+currentTime, "", 1, "R", false, 0, "")

	pdf.Cell(120, 6, "โทรศัพท์: "+result.Phone) // เบอร์โทรเจ้าของจากตาราง users
	pdf.Ln(10)

	// วาดเส้นคั่นหัวกระดาษ
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.5)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(10)

	// --- 🟠 ส่วนตาราง (Table Header) ---
	pdf.SetFillColor(33, 37, 41)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("THSarabun", "B", 13)

	w := []float64{12, 85, 20, 20, 43}
	headers := []string{"ลำดับ", "ชื่อสินค้า", "จำนวน", "หน่วยนับ", "หมายเหตุ"}

	for i, str := range headers {
		pdf.CellFormat(w[i], 10, str, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// --- ⚪ ส่วนตาราง (Table Body) ---
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("THSarabun", "", 12)

	for i, item := range req.Items {
		if i%2 != 0 {
			pdf.SetFillColor(248, 249, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		pdf.CellFormat(w[0], 10, fmt.Sprintf("%d", i+1), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w[1], 10, " "+item.Name, "1", 0, "L", true, 0, "")
		pdf.CellFormat(w[2], 10, fmt.Sprintf("%d", item.Quantity), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w[3], 10, item.Unit, "1", 0, "C", true, 0, "")
		pdf.CellFormat(w[4], 10, item.Note, "1", 1, "L", true, 0, "")
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=order_report.pdf")
	pdf.Output(c.Writer)
}
