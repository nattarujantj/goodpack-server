package models

import (
	"fmt"
	"time"

	"goodpack-server/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PriceInfo represents price information for VAT and Non-VAT
type PriceInfo struct {
	Latest     float64 `bson:"latest" json:"latest"`         // ราคาล่าสุด
	Min        float64 `bson:"min" json:"min"`               // ราคาต่ำสุด
	Max        float64 `bson:"max" json:"max"`               // ราคาสูงสุด
	Average    float64 `bson:"average" json:"average"`       // ราคาเฉลี่ย (Weighted Average)
	AverageYTD float64 `bson:"averageYTD" json:"averageYTD"` // ราคาเฉลี่ย YTD (Weighted Average)
	AverageMTD float64 `bson:"averageMTD" json:"averageMTD"` // ราคาเฉลี่ย MTD (Weighted Average)

	// สำหรับคำนวณ Weighted Average
	TotalQuantity int     `bson:"totalQuantity" json:"totalQuantity"` // จำนวนสินค้าทั้งหมด
	TotalAmount   float64 `bson:"totalAmount" json:"totalAmount"`     // มูลค่ารวม (ราคา × จำนวน)

	// สำหรับคำนวณ Weighted Average YTD
	YTDYear        int     `bson:"ytdYear" json:"ytdYear"`               // ปีที่เก็บข้อมูล YTD
	YTDQuantity    int     `bson:"ytdQuantity" json:"ytdQuantity"`       // จำนวนสินค้าในปีนี้
	YTDTotalAmount float64 `bson:"ytdTotalAmount" json:"ytdTotalAmount"` // มูลค่ารวมในปีนี้ (ราคา × จำนวน)

	// สำหรับคำนวณ Weighted Average MTD
	MTDMonth       int     `bson:"mtdMonth" json:"mtdMonth"`             // เดือนที่เก็บข้อมูล MTD
	MTDYear        int     `bson:"mtdYear" json:"mtdYear"`               // ปีที่เก็บข้อมูล MTD
	MTDQuantity    int     `bson:"mtdQuantity" json:"mtdQuantity"`       // จำนวนสินค้าในเดือนนี้
	MTDTotalAmount float64 `bson:"mtdTotalAmount" json:"mtdTotalAmount"` // มูลค่ารวมในเดือนนี้ (ราคา × จำนวน)
}

// TierPrice represents tier pricing for sales
type TierPrice struct {
	MinQuantity    int       `bson:"minQuantity" json:"minQuantity"`                     // จำนวนขั้นต่ำ
	MaxQuantity    *int      `bson:"maxQuantity,omitempty" json:"maxQuantity,omitempty"` // จำนวนสูงสุด (nil = ไม่จำกัด)
	Price          PriceInfo `bson:"price" json:"price"`                                 // ข้อมูลราคา
	WholesalePrice float64   `bson:"wholesalePrice" json:"wholesalePrice"`               // ราคาขายส่ง (บาท)
}

// Price represents all pricing information
type Price struct {
	PurchaseVAT    PriceInfo   `bson:"purchaseVAT" json:"purchaseVAT"`       // ราคาซื้อ VAT
	PurchaseNonVAT PriceInfo   `bson:"purchaseNonVAT" json:"purchaseNonVAT"` // ราคาซื้อ Non-VAT
	SaleVAT        PriceInfo   `bson:"saleVAT" json:"saleVAT"`               // ราคาขาย VAT
	SaleNonVAT     PriceInfo   `bson:"saleNonVAT" json:"saleNonVAT"`         // ราคาขาย Non-VAT
	SalesTiers     []TierPrice `bson:"salesTiers" json:"salesTiers"`         // ราคาขายแบบ tier
}

// StockInfo represents stock information for VAT and Non-VAT
type StockInfo struct {
	InitialStock int `bson:"initialStock" json:"initialStock"` // สินค้ายกยอดมา
	Purchased    int `bson:"purchased" json:"purchased"`       // ซื้อ
	Sold         int `bson:"sold" json:"sold"`                 // ขาย
	Remaining    int `bson:"remaining" json:"remaining"`       // คงเหลือ
}

// Stock represents all stock information
type Stock struct {
	VAT                StockInfo `bson:"vat" json:"vat"`                               // สต็อก VAT
	NonVAT             StockInfo `bson:"nonVAT" json:"nonVAT"`                         // สต็อก Non-VAT
	ActualStockInitial int       `bson:"actualStockInitial" json:"actualStockInitial"` // ยกยอด ActualStock
	ActualStock        int       `bson:"actualStock" json:"actualStock"`               // สินค้าคงเหลือจริง
}

// Product represents a product in the inventory
type Product struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SKUID           string             `bson:"skuId" json:"skuId"`             // XY-0000 หรือ XYZ-0000
	Code            string             `bson:"code" json:"code"`               // XY-aaaa/AB
	Name            string             `bson:"name" json:"name"`               // ชื่อสินค้า
	Description     string             `bson:"description" json:"description"` // รายละเอียด
	Color           string             `bson:"color" json:"color"`             // สี
	Size            string             `bson:"size" json:"size"`               // ขนาด
	Category        string             `bson:"category" json:"category"`       // ประเภทสินค้า (สำหรับสร้าง SKU_ID)
	QRData          string             `bson:"qrData" json:"qrData"`           // ข้อมูล QR
	ImageURL        *string            `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"`
	QuantityPerPack int                `bson:"quantityPerPack" json:"quantityPerPack"` // จำนวน/ลัง(แพ็ค)
	Price           Price              `bson:"price" json:"price"`                     // ข้อมูลราคา
	Stock           Stock              `bson:"stock" json:"stock"`                     // ข้อมูลสต็อก
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// ProductRequest represents the request body for creating/updating a product
type ProductRequest struct {
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Color           string  `json:"color"`
	Size            string  `json:"size"`
	Category        string  `json:"category"`
	ImageURL        *string `json:"imageUrl,omitempty"`
	QuantityPerPack int     `json:"quantityPerPack"`
	Price           Price   `json:"price"`
	Stock           Stock   `json:"stock"`
}

// StockUpdateRequest represents the request body for updating stock
type StockUpdateRequest struct {
	Stock Stock `json:"stock"`
}

// PriceUpdateRequest represents the request body for updating price
type PriceUpdateRequest struct {
	Price Price `json:"price"`
}

// ToProduct converts ProductRequest to Product
func (pr *ProductRequest) ToProduct() *Product {
	now := utils.NowInThailand()
	return &Product{
		Name:            pr.Name,
		Description:     pr.Description,
		Color:           pr.Color,
		Size:            utils.NormalizeSize(pr.Size), // Normalize size (e.g., "50ml." → "50 mL")
		Category:        pr.Category,
		ImageURL:        pr.ImageURL,
		QuantityPerPack: pr.QuantityPerPack,
		Price:           pr.Price,
		Stock:           pr.Stock,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// UpdateFromRequest updates Product from ProductRequest
// Note: ไม่อัพเดท Price และ Stock เพราะค่าเหล่านี้ควรถูกอัพเดทจากระบบ purchase/sale โดยเฉพาะ
func (p *Product) UpdateFromRequest(pr *ProductRequest) {
	p.Name = pr.Name
	p.Description = pr.Description
	p.Color = pr.Color
	p.Size = utils.NormalizeSize(pr.Size) // Normalize size (e.g., "50ml." → "50 mL")
	p.Category = pr.Category
	// ไม่อัพเดท ImageURL ตรงนี้ เพื่อป้องกันการรีเซ็ตรูปเป็น null โดยไม่ได้ตั้งใจ
	// ใช้ endpoint เฉพาะ POST /products/{id}/image สำหรับอัพโหลดรูป
	// และ DELETE /products/{id}/image สำหรับลบรูป
	p.QuantityPerPack = pr.QuantityPerPack
	// ไม่อัพเดท Price และ Stock ตรงนี้ เพื่อป้องกันการรีเซ็ตค่าเป็น 0
	// ใช้ UpdatePrice, UpdateStock หรือ endpoint เฉพาะแทน

	// อัพเดท SalesTiers เสมอ (empty array = ลบ tier ทั้งหมด)
	p.Price.SalesTiers = pr.Price.SalesTiers

	p.UpdatedAt = utils.NowInThailand()
}

// GetTotalStock returns the actual stock (ActualStock represents the real total)
func (p *Product) GetTotalStock() int {
	return p.Stock.ActualStock
}

// GetDisplayPrice returns the latest purchase price for display
func (p *Product) GetDisplayPrice() float64 {
	if p.Price.PurchaseVAT.Latest > 0 {
		return p.Price.PurchaseVAT.Latest
	}
	return p.Price.PurchaseNonVAT.Latest
}

// UpdatePrice updates price information based on new transaction
// quantity: จำนวนสินค้าในรายการนี้ (ใช้สำหรับคำนวณ Weighted Average)
// ถ้า quantity = 0 จะใช้ค่า default = 1 (สำหรับกรณี manual update)
func (p *Product) UpdatePrice(newPrice float64, isVAT bool, isPurchase bool, quantity int) {
	now := utils.NowInThailand()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	// ถ้า quantity = 0 ให้ใช้ค่า default = 1 (สำหรับกรณี manual update)
	if quantity <= 0 {
		quantity = 1
	}

	var priceInfo *PriceInfo

	// เลือก PriceInfo ที่จะอัปเดต
	if isPurchase {
		if isVAT {
			priceInfo = &p.Price.PurchaseVAT
		} else {
			priceInfo = &p.Price.PurchaseNonVAT
		}
	} else {
		if isVAT {
			priceInfo = &p.Price.SaleVAT
		} else {
			priceInfo = &p.Price.SaleNonVAT
		}
	}

	// 1. อัปเดต latest
	priceInfo.Latest = newPrice

	// 2. อัปเดต min (ระวังเคสแรก)
	if priceInfo.Min == 0 || newPrice < priceInfo.Min {
		priceInfo.Min = newPrice
	}

	// 3. อัปเดต max
	if newPrice > priceInfo.Max {
		priceInfo.Max = newPrice
	}

	// 4. อัปเดต Weighted Average (TotalAmount / TotalQuantity)
	amount := newPrice * float64(quantity)
	priceInfo.TotalAmount += amount
	priceInfo.TotalQuantity += quantity

	if priceInfo.TotalQuantity > 0 {
		priceInfo.Average = priceInfo.TotalAmount / float64(priceInfo.TotalQuantity)
		// ปัดเศษเป็น 2 ตำแหน่ง
		priceInfo.Average = float64(int(priceInfo.Average*100+0.5)) / 100
	} else {
		priceInfo.Average = newPrice
	}

	// 5. อัปเดต YTD (Weighted Average)
	if priceInfo.YTDYear != currentYear {
		// ปีใหม่ - รีเซ็ตข้อมูล YTD
		priceInfo.YTDYear = currentYear
		priceInfo.YTDQuantity = quantity
		priceInfo.YTDTotalAmount = amount
		priceInfo.AverageYTD = newPrice
	} else {
		// ปีเดิม - อัปเดตข้อมูล YTD
		priceInfo.YTDQuantity += quantity
		priceInfo.YTDTotalAmount += amount

		if priceInfo.YTDQuantity > 0 {
			priceInfo.AverageYTD = priceInfo.YTDTotalAmount / float64(priceInfo.YTDQuantity)
			// ปัดเศษเป็น 2 ตำแหน่ง
			priceInfo.AverageYTD = float64(int(priceInfo.AverageYTD*100+0.5)) / 100
		} else {
			priceInfo.AverageYTD = newPrice
		}
	}

	// 6. อัปเดต MTD (Weighted Average)
	if priceInfo.MTDYear != currentYear || priceInfo.MTDMonth != currentMonth {
		// เดือนใหม่ - รีเซ็ตข้อมูล MTD
		priceInfo.MTDYear = currentYear
		priceInfo.MTDMonth = currentMonth
		priceInfo.MTDQuantity = quantity
		priceInfo.MTDTotalAmount = amount
		priceInfo.AverageMTD = newPrice
	} else {
		// เดือนเดิม - อัปเดตข้อมูล MTD
		priceInfo.MTDQuantity += quantity
		priceInfo.MTDTotalAmount += amount

		if priceInfo.MTDQuantity > 0 {
			priceInfo.AverageMTD = priceInfo.MTDTotalAmount / float64(priceInfo.MTDQuantity)
			// ปัดเศษเป็น 2 ตำแหน่ง
			priceInfo.AverageMTD = float64(int(priceInfo.AverageMTD*100+0.5)) / 100
		} else {
			priceInfo.AverageMTD = newPrice
		}
	}
}

// RollbackPriceUpdate rollback price information when deleting/updating transaction
// quantity: จำนวนสินค้าที่ต้องการ rollback
func (p *Product) RollbackPriceUpdate(oldPrice float64, isVAT bool, isPurchase bool, quantity int) {
	now := utils.NowInThailand()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	// ถ้า quantity = 0 ไม่ต้อง rollback
	if quantity <= 0 {
		return
	}

	var priceInfo *PriceInfo

	// เลือก PriceInfo ที่จะ rollback
	if isPurchase {
		if isVAT {
			priceInfo = &p.Price.PurchaseVAT
		} else {
			priceInfo = &p.Price.PurchaseNonVAT
		}
	} else {
		if isVAT {
			priceInfo = &p.Price.SaleVAT
		} else {
			priceInfo = &p.Price.SaleNonVAT
		}
	}

	// Rollback Weighted Average
	amount := oldPrice * float64(quantity)
	if priceInfo.TotalAmount >= amount {
		priceInfo.TotalAmount -= amount
	}
	if priceInfo.TotalQuantity >= quantity {
		priceInfo.TotalQuantity -= quantity
	}

	if priceInfo.TotalQuantity > 0 {
		priceInfo.Average = priceInfo.TotalAmount / float64(priceInfo.TotalQuantity)
		// ปัดเศษเป็น 2 ตำแหน่ง
		priceInfo.Average = float64(int(priceInfo.Average*100+0.5)) / 100
	} else {
		priceInfo.Average = 0
	}

	// Rollback YTD (ถ้าอยู่ในปีเดียวกัน)
	if priceInfo.YTDYear == currentYear {
		if priceInfo.YTDTotalAmount >= amount {
			priceInfo.YTDTotalAmount -= amount
		}
		if priceInfo.YTDQuantity >= quantity {
			priceInfo.YTDQuantity -= quantity
		}

		if priceInfo.YTDQuantity > 0 {
			priceInfo.AverageYTD = priceInfo.YTDTotalAmount / float64(priceInfo.YTDQuantity)
			// ปัดเศษเป็น 2 ตำแหน่ง
			priceInfo.AverageYTD = float64(int(priceInfo.AverageYTD*100+0.5)) / 100
		} else {
			priceInfo.AverageYTD = 0
		}
	}

	// Rollback MTD (ถ้าอยู่ในเดือนเดียวกัน)
	if priceInfo.MTDYear == currentYear && priceInfo.MTDMonth == currentMonth {
		if priceInfo.MTDTotalAmount >= amount {
			priceInfo.MTDTotalAmount -= amount
		}
		if priceInfo.MTDQuantity >= quantity {
			priceInfo.MTDQuantity -= quantity
		}

		if priceInfo.MTDQuantity > 0 {
			priceInfo.AverageMTD = priceInfo.MTDTotalAmount / float64(priceInfo.MTDQuantity)
			// ปัดเศษเป็น 2 ตำแหน่ง
			priceInfo.AverageMTD = float64(int(priceInfo.AverageMTD*100+0.5)) / 100
		} else {
			priceInfo.AverageMTD = 0
		}
	}
}

// IsLowStock checks if the product is low on stock
func (p *Product) IsLowStock() bool {
	totalStock := p.GetTotalStock()
	return totalStock <= 10
}

// GetFormattedPrice returns formatted price string
func (p *Product) GetFormattedPrice() string {
	price := p.GetDisplayPrice()
	return fmt.Sprintf("฿%.2f", price)
}
