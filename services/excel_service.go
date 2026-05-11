package services

import (
	"bytes"
	"fmt"
	"time"

	"goodpack-server/models"

	"github.com/xuri/excelize/v2"
)

type ExcelService struct{}

func NewExcelService() *ExcelService {
	return &ExcelService{}
}

// GeneratePurchaseSaleExcel creates an Excel file with 5 sheets (or 6 if snapshot is provided):
// ขาย VAT, ขาย Non-VAT, ซื้อ VAT, ซื้อ Non-VAT, ค่าใช้จ่าย, สินค้าคงคลัง (optional)
func (s *ExcelService) GeneratePurchaseSaleExcel(purchases []models.Purchase, sales []models.Sale, expenses []models.Expense, month int, year int, snapshot *models.InventorySnapshot) (*bytes.Buffer, error) {
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

	// Filter data by VAT status
	var salesVAT, salesNonVAT []models.Sale
	var purchasesVAT, purchasesNonVAT []models.Purchase

	for _, sale := range sales {
		if sale.IsVAT {
			salesVAT = append(salesVAT, sale)
		} else {
			salesNonVAT = append(salesNonVAT, sale)
		}
	}

	for _, purchase := range purchases {
		if purchase.IsVAT {
			purchasesVAT = append(purchasesVAT, purchase)
		} else {
			purchasesNonVAT = append(purchasesNonVAT, purchase)
		}
	}

	// Create sheets (Sheet1 will be renamed to first sheet)
	f.SetSheetName("Sheet1", "ขาย VAT")
	s.createSaleSheet(f, "ขาย VAT", salesVAT, monthName, buddhistYear, true)

	f.NewSheet("ขาย Non-VAT")
	s.createSaleSheet(f, "ขาย Non-VAT", salesNonVAT, monthName, buddhistYear, false)

	f.NewSheet("ซื้อ VAT")
	s.createPurchaseSheet(f, "ซื้อ VAT", purchasesVAT, monthName, buddhistYear, true)

	f.NewSheet("ซื้อ Non-VAT")
	s.createPurchaseSheet(f, "ซื้อ Non-VAT", purchasesNonVAT, monthName, buddhistYear, false)

	f.NewSheet("ค่าใช้จ่าย")
	s.createExpenseSheet(f, "ค่าใช้จ่าย", expenses, monthName, buddhistYear)

	if snapshot != nil {
		f.NewSheet("สินค้าคงคลัง")
		s.createInventorySheet(f, "สินค้าคงคลัง", snapshot, monthName, buddhistYear)
	}

	// Save to buffer
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write Excel to buffer: %v", err)
	}

	return buf, nil
}

// getCustomerDisplayName returns "ชื่อบริษัท (เลขผู้เสียภาษี)" or just "ชื่อบริษัท" if no tax ID
func getCustomerDisplayName(customerName string, taxID *string) string {
	if taxID != nil && *taxID != "" {
		return fmt.Sprintf("%s (%s)", customerName, *taxID)
	}
	return customerName
}

func (s *ExcelService) createPurchaseSheet(f *excelize.File, sheet string, purchases []models.Purchase, monthName string, year int, isVAT bool) {
	// Set column widths
	// A: ลำดับ, B: รหัส, C: วันที่, D: ชื่อลูกค้า, E: รายการสินค้า, F: ราคารวมก่อน VAT, G: VAT, H: ราคารวม, I: ค่าส่ง, J: รวมทั้งหมด
	f.SetColWidth(sheet, "A", "A", 5)  // ลำดับ
	f.SetColWidth(sheet, "B", "B", 15) // รหัส
	f.SetColWidth(sheet, "C", "C", 12) // วันที่
	f.SetColWidth(sheet, "D", "D", 40) // ชื่อลูกค้า (บริษัท + เลขผู้เสียภาษี)
	f.SetColWidth(sheet, "E", "E", 40) // รายการสินค้า
	f.SetColWidth(sheet, "F", "F", 15) // ราคารวมก่อน VAT
	f.SetColWidth(sheet, "G", "G", 12) // VAT
	f.SetColWidth(sheet, "H", "H", 15) // ราคารวม
	f.SetColWidth(sheet, "I", "I", 12) // ค่าส่ง
	f.SetColWidth(sheet, "J", "J", 15) // รวมทั้งหมด

	// Styles
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	headerColor := "4472C4" // Blue for purchases
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{headerColor}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

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

	numberStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:    4, // #,##0.00
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Title
	vatLabel := "VAT"
	if !isVAT {
		vatLabel = "Non-VAT"
	}
	f.MergeCell(sheet, "A1", "J1")
	f.SetCellValue(sheet, "A1", fmt.Sprintf("รายการซื้อ (%s) - %s %d", vatLabel, monthName, year))
	f.SetCellStyle(sheet, "A1", "J1", titleStyle)

	// Headers
	headers := []string{"ลำดับ", "รหัส", "วันที่", "ชื่อลูกค้า", "รายการสินค้า", "ราคารวมก่อน VAT", "VAT", "ราคารวม", "ค่าส่ง", "รวมทั้งหมด"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c3", 'A'+i)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A3", "J3", headerStyle)

	// Data rows
	var sumBeforeVAT, sumVAT, sumAfterVAT, sumShipping, sumTotal float64

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

		// ชื่อซัพพลายเออร์ (บริษัท + เลขผู้เสียภาษี)
		supplierDisplay := getCustomerDisplayName(p.SupplierName, p.TaxID)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), supplierDisplay)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), dataStyle)

		// รายการสินค้า
		var items string
		for j, item := range p.Items {
			if j > 0 {
				items += ", "
			}
			items += fmt.Sprintf("%s (x%d)", item.ProductName, item.Quantity)
		}
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), items)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), dataStyle)

		// Calculate values
		beforeVAT := p.TotalAmount
		vatAmount := p.TotalVAT
		if !isVAT {
			vatAmount = 0
		}
		afterVAT := beforeVAT + vatAmount
		shipping := p.ShippingCost
		total := afterVAT + shipping

		// ราคารวมก่อน VAT
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), beforeVAT)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), numberStyle)

		// VAT
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), vatAmount)
		f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), numberStyle)

		// ราคารวม
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), afterVAT)
		f.SetCellStyle(sheet, fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), numberStyle)

		// ค่าส่ง
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), shipping)
		f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("I%d", row), numberStyle)

		// รวมทั้งหมด
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), total)
		f.SetCellStyle(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), numberStyle)

		// Accumulate totals
		sumBeforeVAT += beforeVAT
		sumVAT += vatAmount
		sumAfterVAT += afterVAT
		sumShipping += shipping
		sumTotal += total
	}

	// Summary row
	summaryRow := len(purchases) + 5
	f.SetCellValue(sheet, fmt.Sprintf("E%d", summaryRow), "รวมทั้งหมด:")
	f.SetCellStyle(sheet, fmt.Sprintf("E%d", summaryRow), fmt.Sprintf("E%d", summaryRow), headerStyle)

	f.SetCellValue(sheet, fmt.Sprintf("F%d", summaryRow), sumBeforeVAT)
	f.SetCellValue(sheet, fmt.Sprintf("G%d", summaryRow), sumVAT)
	f.SetCellValue(sheet, fmt.Sprintf("H%d", summaryRow), sumAfterVAT)
	f.SetCellValue(sheet, fmt.Sprintf("I%d", summaryRow), sumShipping)
	f.SetCellValue(sheet, fmt.Sprintf("J%d", summaryRow), sumTotal)
	f.SetCellStyle(sheet, fmt.Sprintf("F%d", summaryRow), fmt.Sprintf("J%d", summaryRow), numberStyle)
}

func (s *ExcelService) createSaleSheet(f *excelize.File, sheet string, sales []models.Sale, monthName string, year int, isVAT bool) {
	// Set column widths
	// A: ลำดับ, B: รหัส, C: วันที่, D: ชื่อลูกค้า, E: รายการสินค้า, F: ราคารวมก่อน VAT, G: VAT, H: ราคารวม, I: ค่าส่ง, J: รวมทั้งหมด
	f.SetColWidth(sheet, "A", "A", 5)  // ลำดับ
	f.SetColWidth(sheet, "B", "B", 15) // รหัส
	f.SetColWidth(sheet, "C", "C", 12) // วันที่
	f.SetColWidth(sheet, "D", "D", 40) // ชื่อลูกค้า (บริษัท + เลขผู้เสียภาษี)
	f.SetColWidth(sheet, "E", "E", 40) // รายการสินค้า
	f.SetColWidth(sheet, "F", "F", 15) // ราคารวมก่อน VAT
	f.SetColWidth(sheet, "G", "G", 12) // VAT
	f.SetColWidth(sheet, "H", "H", 15) // ราคารวม
	f.SetColWidth(sheet, "I", "I", 12) // ค่าส่ง
	f.SetColWidth(sheet, "J", "J", 15) // รวมทั้งหมด

	// Styles
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	headerColor := "70AD47" // Green for sales
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{headerColor}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

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

	numberStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:    4, // #,##0.00
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Title
	vatLabel := "VAT"
	if !isVAT {
		vatLabel = "Non-VAT"
	}
	f.MergeCell(sheet, "A1", "J1")
	f.SetCellValue(sheet, "A1", fmt.Sprintf("รายการขาย (%s) - %s %d", vatLabel, monthName, year))
	f.SetCellStyle(sheet, "A1", "J1", titleStyle)

	// Headers
	headers := []string{"ลำดับ", "รหัส", "วันที่", "ชื่อลูกค้า", "รายการสินค้า", "ราคารวมก่อน VAT", "VAT", "ราคารวม", "ค่าส่ง", "รวมทั้งหมด"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c3", 'A'+i)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A3", "J3", headerStyle)

	// Data rows
	var sumBeforeVAT, sumVAT, sumAfterVAT, sumShipping, sumTotal float64

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

		// ชื่อลูกค้า (บริษัท + เลขผู้เสียภาษี)
		customerDisplay := getCustomerDisplayName(sale.CustomerName, sale.TaxID)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), customerDisplay)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), dataStyle)

		// รายการสินค้า
		var items string
		for j, item := range sale.Items {
			if j > 0 {
				items += ", "
			}
			items += fmt.Sprintf("%s (x%d)", item.ProductName, item.Quantity)
		}
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), items)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), dataStyle)

		// Calculate values based on vatType
		var itemsTotal float64
		for _, item := range sale.Items {
			itemsTotal += item.TotalPrice
		}

		var beforeVAT, vatAmount, afterVAT float64
		shipping := sale.ShippingCost

		if isVAT {
			if sale.VatType == "inclusive" {
				// VAT ใน: ราคาที่กรอกรวม VAT แล้ว
				beforeVAT = itemsTotal / 1.07
				vatAmount = itemsTotal - beforeVAT
				afterVAT = itemsTotal
			} else {
				// VAT นอก: ราคา + VAT 7%
				beforeVAT = itemsTotal
				vatAmount = itemsTotal * 0.07
				afterVAT = itemsTotal + vatAmount
			}
		} else {
			beforeVAT = itemsTotal
			vatAmount = 0
			afterVAT = itemsTotal
		}
		total := afterVAT + shipping

		// ราคารวมก่อน VAT
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), beforeVAT)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), numberStyle)

		// VAT
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), vatAmount)
		f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), numberStyle)

		// ราคารวม
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), afterVAT)
		f.SetCellStyle(sheet, fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), numberStyle)

		// ค่าส่ง
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), shipping)
		f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("I%d", row), numberStyle)

		// รวมทั้งหมด
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), total)
		f.SetCellStyle(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), numberStyle)

		// Accumulate totals
		sumBeforeVAT += beforeVAT
		sumVAT += vatAmount
		sumAfterVAT += afterVAT
		sumShipping += shipping
		sumTotal += total
	}

	// Summary row
	summaryRow := len(sales) + 5
	f.SetCellValue(sheet, fmt.Sprintf("E%d", summaryRow), "รวมทั้งหมด:")
	f.SetCellStyle(sheet, fmt.Sprintf("E%d", summaryRow), fmt.Sprintf("E%d", summaryRow), headerStyle)

	f.SetCellValue(sheet, fmt.Sprintf("F%d", summaryRow), sumBeforeVAT)
	f.SetCellValue(sheet, fmt.Sprintf("G%d", summaryRow), sumVAT)
	f.SetCellValue(sheet, fmt.Sprintf("H%d", summaryRow), sumAfterVAT)
	f.SetCellValue(sheet, fmt.Sprintf("I%d", summaryRow), sumShipping)
	f.SetCellValue(sheet, fmt.Sprintf("J%d", summaryRow), sumTotal)
	f.SetCellStyle(sheet, fmt.Sprintf("F%d", summaryRow), fmt.Sprintf("J%d", summaryRow), numberStyle)
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

func (s *ExcelService) createExpenseSheet(f *excelize.File, sheet string, expenses []models.Expense, monthName string, year int) {
	f.SetColWidth(sheet, "A", "A", 5)
	f.SetColWidth(sheet, "B", "B", 25)
	f.SetColWidth(sheet, "C", "C", 40)
	f.SetColWidth(sheet, "D", "D", 12)
	f.SetColWidth(sheet, "E", "E", 15)
	f.SetColWidth(sheet, "F", "F", 30)

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"ED7D31"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

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

	numberStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:    4,
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	f.MergeCell(sheet, "A1", "F1")
	f.SetCellValue(sheet, "A1", fmt.Sprintf("ค่าใช้จ่าย - %s %d", monthName, year))
	f.SetCellStyle(sheet, "A1", "F1", titleStyle)

	headers := []string{"ลำดับ", "หมวดหมู่", "รายละเอียด", "วันที่", "จำนวนเงิน", "หมายเหตุ"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c3", 'A'+i)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A3", "F3", headerStyle)

	var sumAmount float64

	for i, exp := range expenses {
		row := i + 4

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), dataStyle)

		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), exp.Category)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), dataStyle)

		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), exp.Description)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), dataStyle)

		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), exp.ExpenseDate.Format("02/01/2006"))
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), dataStyle)

		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), exp.Amount)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), numberStyle)

		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), exp.Notes)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), dataStyle)

		sumAmount += exp.Amount
	}

	summaryRow := len(expenses) + 5
	f.SetCellValue(sheet, fmt.Sprintf("D%d", summaryRow), "รวมทั้งหมด:")
	f.SetCellStyle(sheet, fmt.Sprintf("D%d", summaryRow), fmt.Sprintf("D%d", summaryRow), headerStyle)

	f.SetCellValue(sheet, fmt.Sprintf("E%d", summaryRow), sumAmount)
	f.SetCellStyle(sheet, fmt.Sprintf("E%d", summaryRow), fmt.Sprintf("E%d", summaryRow), numberStyle)
}

// Helper function to format time as Thai date
func FormatThaiDate(t time.Time) string {
	return t.Format("02/01/2006")
}

func (s *ExcelService) createInventorySheet(f *excelize.File, sheet string, snapshot *models.InventorySnapshot, monthName string, buddhistYear int) {
	f.SetColWidth(sheet, "A", "A", 5)  // ลำดับ
	f.SetColWidth(sheet, "B", "B", 12) // รหัส SKU
	f.SetColWidth(sheet, "C", "C", 15) // รหัสสินค้า
	f.SetColWidth(sheet, "D", "D", 35) // ชื่อสินค้า
	f.SetColWidth(sheet, "E", "E", 18) // หมวดหมู่
	f.SetColWidth(sheet, "F", "F", 12) // สี
	f.SetColWidth(sheet, "G", "G", 12) // ขนาด
	f.SetColWidth(sheet, "H", "H", 15) // คงเหลือ VAT
	f.SetColWidth(sheet, "I", "I", 18) // คงเหลือ Non-VAT
	f.SetColWidth(sheet, "J", "J", 15) // คงเหลือจริง

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7030A0"}}, // Purple for inventory
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

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

	numberStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:    1, // integer
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	summaryStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7030A0"}},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		NumFmt:    1,
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Title row
	f.MergeCell(sheet, "A1", "J1")
	f.SetCellValue(sheet, "A1", fmt.Sprintf("สินค้าคงคลัง ณ สิ้นเดือน %s %d (บันทึก: %s)", monthName, buddhistYear, snapshot.SnapshotDate.Format("02/01/2006 15:04:05")))
	f.SetCellStyle(sheet, "A1", "J1", titleStyle)

	// Header row
	headers := []string{"ลำดับ", "รหัส SKU", "รหัสสินค้า", "ชื่อสินค้า", "หมวดหมู่", "สี", "ขนาด", "คงเหลือ VAT", "คงเหลือ Non-VAT", "คงเหลือจริง"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c3", 'A'+i)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A3", "J3", headerStyle)

	// Data rows
	for i, p := range snapshot.Products {
		row := i + 4
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), dataStyle)

		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), p.SKUID)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), dataStyle)

		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), p.Code)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), dataStyle)

		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), p.Name)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), dataStyle)

		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), p.Category)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), dataStyle)

		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), p.Color)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), dataStyle)

		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), p.Size)
		f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), dataStyle)

		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), p.VATRemaining)
		f.SetCellStyle(sheet, fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), numberStyle)

		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), p.NonVATRemaining)
		f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("I%d", row), numberStyle)

		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), p.ActualStock)
		f.SetCellStyle(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), numberStyle)
	}

	// Summary row
	summaryRow := len(snapshot.Products) + 5
	f.MergeCell(sheet, fmt.Sprintf("A%d", summaryRow), fmt.Sprintf("G%d", summaryRow))
	f.SetCellValue(sheet, fmt.Sprintf("A%d", summaryRow), fmt.Sprintf("รวมทั้งหมด (%d รายการ)", snapshot.TotalProducts))
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", summaryRow), fmt.Sprintf("G%d", summaryRow), headerStyle)

	f.SetCellValue(sheet, fmt.Sprintf("H%d", summaryRow), snapshot.TotalVATStock)
	f.SetCellStyle(sheet, fmt.Sprintf("H%d", summaryRow), fmt.Sprintf("H%d", summaryRow), summaryStyle)

	f.SetCellValue(sheet, fmt.Sprintf("I%d", summaryRow), snapshot.TotalNonVATStock)
	f.SetCellStyle(sheet, fmt.Sprintf("I%d", summaryRow), fmt.Sprintf("I%d", summaryRow), summaryStyle)

	f.SetCellValue(sheet, fmt.Sprintf("J%d", summaryRow), snapshot.TotalActualStock)
	f.SetCellStyle(sheet, fmt.Sprintf("J%d", summaryRow), fmt.Sprintf("J%d", summaryRow), summaryStyle)
}
