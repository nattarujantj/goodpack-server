package services

import (
	"bytes"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
}

func NewEmailService() *EmailService {
	return &EmailService{
		smtpHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		smtpPort:     getEnv("SMTP_PORT", "587"),
		smtpUsername: getEnv("SMTP_USERNAME", ""),
		smtpPassword: getEnv("SMTP_PASSWORD", ""),
		fromEmail:    getEnv("SMTP_FROM_EMAIL", ""),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// SendEmailWithAttachment sends an email with an Excel attachment
func (s *EmailService) SendEmailWithAttachment(to []string, subject string, body string, attachment *bytes.Buffer, filename string) error {
	if s.smtpUsername == "" || s.smtpPassword == "" {
		return fmt.Errorf("SMTP credentials not configured. Please set SMTP_USERNAME and SMTP_PASSWORD environment variables")
	}

	// Create boundary for multipart
	boundary := "GoodPackExport2024"

	// Build message
	var message bytes.Buffer

	// Headers
	message.WriteString(fmt.Sprintf("From: %s\r\n", s.fromEmail))
	message.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	message.WriteString(fmt.Sprintf("Subject: =?UTF-8?B?%s?=\r\n", base64Encode(subject)))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	message.WriteString("\r\n")

	// Body part
	message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	message.WriteString("Content-Transfer-Encoding: base64\r\n")
	message.WriteString("\r\n")
	message.WriteString(base64Encode(body))
	message.WriteString("\r\n")

	// Attachment part
	message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	message.WriteString(fmt.Sprintf("Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet; name=\"%s\"\r\n", filename))
	message.WriteString("Content-Transfer-Encoding: base64\r\n")
	message.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", filename))
	message.WriteString("\r\n")

	// Encode attachment to base64
	attachmentData := base64EncodeBytes(attachment.Bytes())
	// Split into 76 character lines
	for i := 0; i < len(attachmentData); i += 76 {
		end := i + 76
		if end > len(attachmentData) {
			end = len(attachmentData)
		}
		message.WriteString(attachmentData[i:end])
		message.WriteString("\r\n")
	}

	// End boundary
	message.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	// Send email
	auth := smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)

	err := smtp.SendMail(
		s.smtpHost+":"+s.smtpPort,
		auth,
		s.fromEmail,
		to,
		message.Bytes(),
	)

	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

// base64Encode encodes a string to base64
func base64Encode(s string) string {
	return base64EncodeBytes([]byte(s))
}

// base64EncodeBytes encodes bytes to base64 string
func base64EncodeBytes(data []byte) string {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	result := make([]byte, ((len(data)+2)/3)*4)
	padding := (3 - len(data)%3) % 3

	n := 0
	for i := 0; i < len(data); i += 3 {
		var val uint32
		val = uint32(data[i]) << 16
		if i+1 < len(data) {
			val |= uint32(data[i+1]) << 8
		}
		if i+2 < len(data) {
			val |= uint32(data[i+2])
		}

		result[n] = base64Chars[(val>>18)&0x3F]
		result[n+1] = base64Chars[(val>>12)&0x3F]
		result[n+2] = base64Chars[(val>>6)&0x3F]
		result[n+3] = base64Chars[val&0x3F]
		n += 4
	}

	// Add padding
	for i := 0; i < padding; i++ {
		result[len(result)-1-i] = '='
	}

	return string(result)
}

// BuildEmailBody creates an HTML email body for the export notification
func (s *EmailService) BuildEmailBody(month int, year int, purchaseCount int, saleCount int, expenseCount int) string {
	thaiMonths := []string{
		"มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน",
		"พฤษภาคม", "มิถุนายน", "กรกฎาคม", "สิงหาคม",
		"กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
	}
	monthName := thaiMonths[month-1]
	buddhistYear := year + 543

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: 'Sarabun', Arial, sans-serif; background-color: #f5f5f5; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: white; border-radius: 8px; padding: 30px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { text-align: center; color: #1976d2; margin-bottom: 20px; }
        .header h1 { margin: 0; }
        .content { color: #333; line-height: 1.6; }
        .summary { background-color: #e3f2fd; padding: 15px; border-radius: 8px; margin: 20px 0; }
        .summary-item { display: flex; justify-content: space-between; margin: 8px 0; }
        .footer { text-align: center; color: #888; font-size: 12px; margin-top: 30px; padding-top: 20px; border-top: 1px solid #eee; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📦 GoodPack Export</h1>
            <p>รายงานประจำเดือน %s %d</p>
        </div>
        <div class="content">
            <p>เรียน</p>
            <p>แนบไฟล์ Excel รายงานซื้อ-ขายประจำเดือน <strong>%s %d</strong> มาพร้อมกับอีเมลนี้</p>
            
            <div class="summary">
                <h3 style="margin-top: 0;">📊 สรุปข้อมูล</h3>
                <div class="summary-item">
                    <span>🛒 รายการซื้อ:</span>
                    <strong>%d รายการ</strong>
                </div>
                <div class="summary-item">
                    <span>💰 รายการขาย:</span>
                    <strong>%d รายการ</strong>
                </div>
                <div class="summary-item">
                    <span>📋 ค่าใช้จ่าย:</span>
                    <strong>%d รายการ</strong>
                </div>
            </div>
            
            <p>ไฟล์ Excel ประกอบด้วย:</p>
            <ul>
                <li><strong>รายการซื้อ</strong> - รายละเอียดการซื้อทั้งหมด</li>
                <li><strong>รายการขาย</strong> - รายละเอียดการขายทั้งหมด</li>
                <li><strong>ค่าใช้จ่าย</strong> - รายละเอียดค่าใช้จ่ายทั่วไป</li>
            </ul>
        </div>
        <div class="footer">
            <p>อีเมลนี้ถูกส่งจากระบบ GoodPackagingSupply Inventory Management</p>
            <p>© %d Nattarujantj. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`, monthName, buddhistYear, monthName, buddhistYear, purchaseCount, saleCount, expenseCount, year)
}

