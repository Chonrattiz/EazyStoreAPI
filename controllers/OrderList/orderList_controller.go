package controllers

import (
	"EazyStoreAPI/assets"
	"EazyStoreAPI/database"
	"EazyStoreAPI/middleware"
	"EazyStoreAPI/models"
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
)

func ExportOrderPDF(c *gin.Context) {
	var req models.ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลที่ส่งมาไม่ถูกต้อง กรุณาลองใหม่อีกครั้ง"})
		return
	}

	// PDF ที่ออกมามีชื่อร้าน ที่อยู่ และเบอร์เจ้าของร้าน ต้องตรวจสิทธิ์ก่อน
	if !middleware.RequireShopAccess(c, req.ShopID) {
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

	//  โหลดฟอนต์จาก embed.FS ตรงๆ เป็น bytes (ไม่เขียนลง temp dir อีกต่อไป) ---
	// เดิมเขียนไฟล์ฟอนต์ลง os.TempDir() แล้ว cache ไว้โดยเช็คแค่ว่าไฟล์มีอยู่หรือยัง (os.Stat)
	// โดยไม่เคยตรวจสอบความถูกต้องของไฟล์ที่ cache ไว้ ถ้าการเขียนครั้งแรกบน container
	// เกิดเขียนไม่สมบูรณ์ (เช่น container ถูก restart หรือ request แรกๆ ชนกัน) ไฟล์ฟอนต์ที่เสีย
	// จะถูกใช้ซ้ำตลอดไปทุก request หลังจากนั้น ทำให้ gofpdf พาร์สฟอนต์ไม่ได้และ PDF ล้มเหลวทุกครั้ง
	// การใช้ AddUTF8FontFromBytes อ่านตรงจาก embed.FS ในหน่วยความจำ ตัดปัญหานี้ทั้งหมด
	regularFontBytes, err := assets.FontsFS.ReadFile("fonts/THSARABUNNEW.TTF")
	if err != nil {
		fmt.Println("❌ Error loading regular font:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถสร้างไฟล์ PDF ได้ (โหลดฟอนต์ไม่สำเร็จ)"})
		return
	}

	boldFontBytes, err := assets.FontsFS.ReadFile("fonts/THSARABUNNEW BOLD.TTF")
	if err != nil {
		fmt.Println("❌ Error loading bold font:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถสร้างไฟล์ PDF ได้ (โหลดฟอนต์ตัวหนาไม่สำเร็จ)"})
		return
	}

	pdf.AddUTF8FontFromBytes("THSarabun", "", regularFontBytes)
	pdf.AddUTF8FontFromBytes("THSarabun", "B", boldFontBytes)

	pdf.AddPage()
	pdf.SetMargins(15, 15, 15)

	// ---  Header ---
	pdf.SetFont("THSarabun", "B", 22)
	pdf.Cell(120, 10, ("ร้าน ")+" "+result.Name)

	pdf.SetFont("THSarabun", "B", 16)
	pdf.CellFormat(0, 10, ("รายงานการสั่งซื้อ"), "", 1, "R", false, 0, "")

	pdf.SetFont("THSarabun", "", 14) 
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(120, 6, (result.Address))

	// เซิร์ฟเวอร์ (container) ส่วนใหญ่ตั้ง timezone เป็น UTC ทำให้ time.Now() ตรงๆ
	// ได้เวลาที่เพี้ยนไป 7 ชั่วโมงจากเวลาไทย ต้องระบุ Asia/Bangkok ให้ชัดเจน
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		loc = time.FixedZone("ICT", 7*60*60) // เผื่อ container ไม่มี tzdata ให้ fallback เป็น UTC+7 ตรงๆ
	}
	nowInLoc := time.Now().In(loc)
	currentTime := nowInLoc.Format("02/01/2006 | 15:04")
	pdf.CellFormat(0, 6, ("วันที่: " + currentTime), "", 1, "R", false, 0, "")

	pdf.Cell(120, 6, ("เบอร์โทรศัพท์: " + result.Phone))
	pdf.Ln(10)

	// เส้นคั่น
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.5)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(10)

	// ---  ส่วนตาราง (Table Header) ---
	pdf.SetFillColor(33, 37, 41)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("THSarabun", "B", 14)

	w := []float64{12, 85, 20, 20, 43}
	headers := []string{"ลำดับ", "ชื่อสินค้า", "จำนวน", "หน่วยนับ", "หมายเหตุ"}

	// ลูปหัวตาราง
	for i, str := range headers {
		pdf.CellFormat(w[i], 10, (str), "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// ---  ส่วนตาราง Body ---
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

	// ---  ส่วนการส่ง Output ---
	var buf bytes.Buffer
	err = pdf.Output(&buf)
	if err != nil {
		fmt.Println("❌ PDF Error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถสร้างไฟล์ PDF ได้ กรุณาลองใหม่อีกครั้ง"})
		return
	}

	c.Header("Content-Type", "application/pdf") //บอก browser ว่านี่ PDF
	dateStr := nowInLoc.Format("02-01-2006")
	fallbackFilename := fmt.Sprintf("%s-report.pdf", dateStr)
	utf8Filename := fmt.Sprintf("%s-%s.pdf", dateStr, result.Name)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", fallbackFilename, url.PathEscape(utf8Filename)))
	c.Header("Content-Length", fmt.Sprintf("%d", buf.Len()))//บอกขนาด bytes ที่จะส่ง

	c.Data(http.StatusOK, "application/pdf", buf.Bytes())
}
