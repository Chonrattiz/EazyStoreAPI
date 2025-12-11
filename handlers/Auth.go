package handlers

import (
	"github.com/gin-gonic/gin"
)

// ฟังก์ชันสำหรับดึงข้อมูล User ทั้งหมด
func GetUsers(c *gin.Context) {
	// var users []database.User
	
    // // ดึงข้อมูลจาก DB ผ่านตัวแปร database.DB ที่เราสร้างไว้
	// result := database.DB.Find(&users)

	// if result.Error != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
	// 	return
	// }

	// c.JSON(http.StatusOK, users)
}

// ฟังก์ชันสำหรับสร้าง User ใหม่
func CreateUser(c *gin.Context) {
	// var user database.User

	// // รับค่า JSON จาก Body มาใส่ในตัวแปร user
	// if err := c.ShouldBindJSON(&user); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }

	// // บันทึกลง DB
	// result := database.DB.Create(&user)
	// if result.Error != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
	// 	return
	// }

	// c.JSON(http.StatusCreated, user)
}