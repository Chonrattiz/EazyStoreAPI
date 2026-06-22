package controllers

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
	"bytes"
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

	// 1. ดึงข้อมูลร้านค้า Join กับตาราง Users
	var result struct {
		Name    string
		Address string
		Phone   string
	}

	err := database.DB.Table("shops").
		Select("shops.name, shops.address, users.phone").
		Joins("left join users on users.user_id = shops.user_id").
		Where("shops.shop_id = ?", req.ShopID).
		First(&result).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบข้อมูลร้านค้าหรือเจ้าของร้าน"})
		return
	}

	// 2. เริ่มสร้าง PDF
	pdf := gofpdf.New("P", "mm", "A4", "")

	// --- 🟢 แก้ไขจุดที่ 1: ตรวจสอบชื่อไฟล์ให้ตรงกับที่ปรากฏใน Folder (ตัวพิมพ์ใหญ่ทั้งหมด) ---
	//
	pdf.AddUTF8Font("THSarabun", "", "assets/fonts/THSARABUNNEW.TTF")
	pdf.AddUTF8Font("THSarabun", "B", "assets/fonts/THSARABUNNEW BOLD.TTF")

	pdf.AddPage()
	pdf.SetMargins(15, 15, 15)

	// --- 🔵 Header ---
	pdf.SetFont("THSarabun", "B", 22)
	pdf.Cell(120, 10, ("ร้าน ")+" "+result.Name)

	pdf.SetFont("THSarabun", "B", 16)
	pdf.CellFormat(0, 10, ("รายงานการสั่งซื้อ"), "", 1, "R", false, 0, "")

	pdf.SetFont("THSarabun", "", 14) // เพิ่มขนาดฟอนต์นิดหน่อยให้อ่านง่าย
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(120, 6, (result.Address))

	currentTime := time.Now().Format("02/01/2006 | 15:04")
	pdf.CellFormat(0, 6, ("วันที่: " + currentTime), "", 1, "R", false, 0, "")

	pdf.Cell(120, 6, ("เบอร์โทรศัพท์: " + result.Phone))
	pdf.Ln(10)

	// เส้นคั่น
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.5)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(10)

	// --- 🟠 ส่วนตาราง (Table Header) ---
	pdf.SetFillColor(33, 37, 41)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("THSarabun", "B", 14)

	w := []float64{12, 85, 20, 20, 43}
	headers := []string{"ลำดับ", "ชื่อสินค้า", "จำนวน", "หน่วยนับ", "หมายเหตุ"}

	// --- 🟢 แก้ไขจุดที่ 3: ลูปหัวตารางแค่รอบเดียว (เดิมคุณมีเบิ้ล 2 ลูป) ---
	for i, str := range headers {
		pdf.CellFormat(w[i], 10, (str), "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// --- ⚪ ส่วนตาราง Body ---
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("THSarabun", "", 14)

	for i, item := range req.Items {
		if i%2 != 0 {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		pdf.CellFormat(w[0], 10, fmt.Sprintf("%d", i+1), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w[1], 10, " "+(item.Name), "1", 0, "L", true, 0, "")
		pdf.CellFormat(w[2], 10, fmt.Sprintf("%d", item.Quantity), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w[3], 10, (item.Unit), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w[4], 10, (item.Note), "1", 1, "L", true, 0, "")
	}

	// --- 🔴 ส่วนการส่ง Output ---
	var buf bytes.Buffer
	err = pdf.Output(&buf)
	if err != nil {
		fmt.Println("❌ PDF Error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=order_report.pdf")
	c.Header("Content-Length", fmt.Sprintf("%d", buf.Len()))

	c.Data(http.StatusOK, "application/pdf", buf.Bytes())
}
