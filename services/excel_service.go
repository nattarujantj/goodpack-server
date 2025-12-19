package services

import (
	"bytes"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
	"goodpack-server/models"
)

type ExcelService struct{}

func NewExcelService() *ExcelService {
	return &ExcelService{}
}

// GeneratePurchaseSaleExcel creates an Excel file with 2 sheets: Purchases and Sales
func (s *ExcelService) GeneratePurchaseSaleExcel(purchases []models.Purchase, sales []models.Sale, month int, year int) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Thai month names
	thaiMonths := []string{
		"มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน",
		"พฤษภาคม", "มิถุนายน", "กรกฎาคม", "สิงหาคม",
		"กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
	}
	monthName := thaiMonths[month-1]
	buddhistYear := year + 543

	// Create Purchases sheet
	purchaseSheet := "รายการซื้อ"
	f.SetSheetName("Sheet1", purchaseSheet)
	s.createPurchaseSheet(f, purchaseSheet, purchases, monthName, buddhistYear)

	// Create Sales sheet
	saleSheet := "รายการขาย"
	f.NewSheet(saleSheet)
	s.createSaleSheet(f, saleSheet, sales, monthName, buddhistYear)

	// Save to buffer
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write Excel to buffer: %v", err)
	}

	return buf, nil
}

func (s *ExcelService) createPurchaseSheet(f *excelize.File, sheet string, purchases []models.Purchase, monthName string, year int) {
	// Set column widths
	f.SetColWidth(sheet, "A", "A", 5)   // ลำดับ
	f.SetColWidth(sheet, "B", "B", 15)  // รหัส
	f.SetColWidth(sheet, "C", "C", 12)  // วันที่
	f.SetColWidth(sheet, "D", "D", 30)  // ชื่อลูกค้า
	f.SetColWidth(sheet, "E", "E", 10)  // VAT
	f.SetColWidth(sheet, "F", "F", 40)  // รายการสินค้า
	f.SetColWidth(sheet, "G", "G", 15)  // ยอดรวม
	f.SetColWidth(sheet, "H", "H", 15)  // VAT
	f.SetColWidth(sheet, "I", "I", 15)  // รวมทั้งหมด
	f.SetColWidth(sheet, "J", "J", 12)  // สถานะชำระ
	f.SetColWidth(sheet, "K", "K", 12)  // สถานะคลัง

	// Title style
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"4472C4"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Data style
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Number style
	numberStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 10},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:       4, // #,##0.00
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Title
	f.MergeCell(sheet, "A1", "K1")
	f.SetCellValue(sheet, "A1", fmt.Sprintf("รายการซื้อ - %s %d", monthName, year))
	f.SetCellStyle(sheet, "A1", "K1", titleStyle)

	// Headers
	headers := []string{"ลำดับ", "รหัส", "วันที่", "ชื่อลูกค้า", "VAT", "รายการสินค้า", "ยอดรวม", "VAT", "รวมทั้งหมด", "สถานะชำระ", "สถานะคลัง"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c3", 'A'+i)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A3", "K3", headerStyle)

	// Data rows
	for i, p := range purchases {
		row := i + 4

		// ลำดับ
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), dataStyle)

		// รหัส
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), p.PurchaseCode)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), dataStyle)

		// วันที่
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), p.PurchaseDate.Format("02/01/2006"))
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), dataStyle)

		// ชื่อลูกค้า
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), p.CustomerName)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), dataStyle)

		// VAT
		vatStatus := "Non-VAT"
		if p.IsVAT {
			vatStatus = "VAT"
		}
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), vatStatus)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), dataStyle)

		// รายการสินค้า
		var items string
		for j, item := range p.Items {
			if j > 0 {
				items += ", "
			}
			items += fmt.Sprintf("%s (x%d)", item.ProductName, item.Quantity)
		}
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), items)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), dataStyle)

		// ยอดรวม
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), p.TotalAmount)
		f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), numberStyle)

		// VAT
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), p.TotalVAT)
		f.SetCellStyle(sheet, fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), numberStyle)

		// รวมทั้งหมด
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), p.GrandTotal)
		f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("I%d", row), numberStyle)

		// สถานะชำระ
		paymentStatus := "ยังไม่ชำระ"
		if p.Payment.IsPaid {
			paymentStatus = "ชำระแล้ว"
		}
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), paymentStatus)
		f.SetCellStyle(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), dataStyle)

		// สถานะคลัง
		warehouseStatus := "ยังไม่อัพเดต"
		if p.Warehouse.IsUpdated {
			warehouseStatus = "อัพเดตแล้ว"
		}
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), warehouseStatus)
		f.SetCellStyle(sheet, fmt.Sprintf("K%d", row), fmt.Sprintf("K%d", row), dataStyle)
	}

	// Summary row
	summaryRow := len(purchases) + 5
	f.SetCellValue(sheet, fmt.Sprintf("F%d", summaryRow), "รวมทั้งหมด:")
	f.SetCellStyle(sheet, fmt.Sprintf("F%d", summaryRow), fmt.Sprintf("F%d", summaryRow), headerStyle)

	var totalAmount, totalVAT, grandTotal float64
	for _, p := range purchases {
		totalAmount += p.TotalAmount
		totalVAT += p.TotalVAT
		grandTotal += p.GrandTotal
	}
	f.SetCellValue(sheet, fmt.Sprintf("G%d", summaryRow), totalAmount)
	f.SetCellValue(sheet, fmt.Sprintf("H%d", summaryRow), totalVAT)
	f.SetCellValue(sheet, fmt.Sprintf("I%d", summaryRow), grandTotal)
	f.SetCellStyle(sheet, fmt.Sprintf("G%d", summaryRow), fmt.Sprintf("I%d", summaryRow), numberStyle)
}

func (s *ExcelService) createSaleSheet(f *excelize.File, sheet string, sales []models.Sale, monthName string, year int) {
	// Set column widths
	f.SetColWidth(sheet, "A", "A", 5)   // ลำดับ
	f.SetColWidth(sheet, "B", "B", 15)  // รหัส
	f.SetColWidth(sheet, "C", "C", 12)  // วันที่
	f.SetColWidth(sheet, "D", "D", 30)  // ชื่อลูกค้า
	f.SetColWidth(sheet, "E", "E", 10)  // VAT
	f.SetColWidth(sheet, "F", "F", 40)  // รายการสินค้า
	f.SetColWidth(sheet, "G", "G", 15)  // ยอดรวม
	f.SetColWidth(sheet, "H", "H", 15)  // ค่าส่ง
	f.SetColWidth(sheet, "I", "I", 12)  // สถานะชำระ
	f.SetColWidth(sheet, "J", "J", 12)  // สถานะคลัง

	// Title style
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"70AD47"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Data style
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Number style
	numberStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 10},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:       4, // #,##0.00
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Title
	f.MergeCell(sheet, "A1", "J1")
	f.SetCellValue(sheet, "A1", fmt.Sprintf("รายการขาย - %s %d", monthName, year))
	f.SetCellStyle(sheet, "A1", "J1", titleStyle)

	// Headers
	headers := []string{"ลำดับ", "รหัส", "วันที่", "ชื่อลูกค้า", "VAT", "รายการสินค้า", "ยอดรวม", "ค่าส่ง", "สถานะชำระ", "สถานะคลัง"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c3", 'A'+i)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A3", "J3", headerStyle)

	// Data rows
	for i, sale := range sales {
		row := i + 4

		// ลำดับ
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), dataStyle)

		// รหัส
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), sale.SaleCode)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), dataStyle)

		// วันที่
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), sale.SaleDate.Format("02/01/2006"))
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), dataStyle)

		// ชื่อลูกค้า
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), sale.CustomerName)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), dataStyle)

		// VAT
		vatStatus := "Non-VAT"
		if sale.IsVAT {
			vatStatus = "VAT"
		}
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), vatStatus)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), dataStyle)

		// รายการสินค้า
		var items string
		for j, item := range sale.Items {
			if j > 0 {
				items += ", "
			}
			items += fmt.Sprintf("%s (x%d)", item.ProductName, item.Quantity)
		}
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), items)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), dataStyle)

		// คำนวณยอดรวม
		var totalAmount float64
		for _, item := range sale.Items {
			totalAmount += item.TotalPrice
		}
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), totalAmount)
		f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), numberStyle)

		// ค่าส่ง
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), sale.ShippingCost)
		f.SetCellStyle(sheet, fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), numberStyle)

		// สถานะชำระ
		paymentStatus := "ยังไม่ชำระ"
		if sale.Payment.IsPaid {
			paymentStatus = "ชำระแล้ว"
		}
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), paymentStatus)
		f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("I%d", row), dataStyle)

		// สถานะคลัง
		warehouseStatus := "ยังไม่อัพเดต"
		if sale.Warehouse.IsUpdated {
			warehouseStatus = "อัพเดตแล้ว"
		}
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), warehouseStatus)
		f.SetCellStyle(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), dataStyle)
	}

	// Summary row
	summaryRow := len(sales) + 5
	f.SetCellValue(sheet, fmt.Sprintf("F%d", summaryRow), "รวมทั้งหมด:")
	f.SetCellStyle(sheet, fmt.Sprintf("F%d", summaryRow), fmt.Sprintf("F%d", summaryRow), headerStyle)

	var totalAmount, totalShipping float64
	for _, sale := range sales {
		for _, item := range sale.Items {
			totalAmount += item.TotalPrice
		}
		totalShipping += sale.ShippingCost
	}
	f.SetCellValue(sheet, fmt.Sprintf("G%d", summaryRow), totalAmount)
	f.SetCellValue(sheet, fmt.Sprintf("H%d", summaryRow), totalShipping)
	f.SetCellStyle(sheet, fmt.Sprintf("G%d", summaryRow), fmt.Sprintf("H%d", summaryRow), numberStyle)
}

// GetThaiMonthName returns Thai month name for a given month number (1-12)
func GetThaiMonthName(month int) string {
	thaiMonths := []string{
		"มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน",
		"พฤษภาคม", "มิถุนายน", "กรกฎาคม", "สิงหาคม",
		"กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
	}
	if month < 1 || month > 12 {
		return ""
	}
	return thaiMonths[month-1]
}

// Helper function to format time as Thai date
func FormatThaiDate(t time.Time) string {
	return t.Format("02/01/2006")
}

