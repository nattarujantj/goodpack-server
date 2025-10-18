package models

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PriceInfo represents price information for VAT and Non-VAT
type PriceInfo struct {
	Latest     float64 `bson:"latest" json:"latest"`         // ราคาล่าสุด
	Min        float64 `bson:"min" json:"min"`               // ราคาต่ำสุด
	Max        float64 `bson:"max" json:"max"`               // ราคาสูงสุด
	Average    float64 `bson:"average" json:"average"`       // ราคาเฉลี่ย
	AverageYTD float64 `bson:"averageYTD" json:"averageYTD"` // ราคาเฉลี่ย YTD
	AverageMTD float64 `bson:"averageMTD" json:"averageMTD"` // ราคาเฉลี่ย MTD

	// สำหรับคำนวณ YTD/MTD
	YTDCount int     `bson:"ytdCount" json:"ytdCount"` // จำนวนครั้งที่ซื้อ/ขายในปีนี้
	YTDTotal float64 `bson:"ytdTotal" json:"ytdTotal"` // รวมราคาในปีนี้
	YTDYear  int     `bson:"ytdYear" json:"ytdYear"`   // ปีที่เก็บข้อมูล YTD

	MTDCount int     `bson:"mtdCount" json:"mtdCount"` // จำนวนครั้งที่ซื้อ/ขายในเดือนนี้
	MTDTotal float64 `bson:"mtdTotal" json:"mtdTotal"` // รวมราคาในเดือนนี้
	MTDMonth int     `bson:"mtdMonth" json:"mtdMonth"` // เดือนที่เก็บข้อมูล MTD
	MTDYear  int     `bson:"mtdYear" json:"mtdYear"`   // ปีที่เก็บข้อมูล MTD
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
	Purchased int `bson:"purchased" json:"purchased"` // ซื้อ
	Sold      int `bson:"sold" json:"sold"`           // ขาย
	Remaining int `bson:"remaining" json:"remaining"` // คงเหลือ
}

// Stock represents all stock information
type Stock struct {
	VAT         StockInfo `bson:"vat" json:"vat"`                 // สต็อก VAT
	NonVAT      StockInfo `bson:"nonVAT" json:"nonVAT"`           // สต็อก Non-VAT
	ActualStock int       `bson:"actualStock" json:"actualStock"` // สินค้าคงเหลือจริง
}

// Product represents a product in the inventory
type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SKUID       string             `bson:"skuId" json:"skuId"`             // XY-0000 หรือ XYZ-0000
	Code        string             `bson:"code" json:"code"`               // XY-aaaa/AB
	Name        string             `bson:"name" json:"name"`               // ชื่อสินค้า
	Description string             `bson:"description" json:"description"` // รายละเอียด
	Color       string             `bson:"color" json:"color"`             // สี
	Size        string             `bson:"size" json:"size"`               // ขนาด
	Category    string             `bson:"category" json:"category"`       // ประเภทสินค้า (สำหรับสร้าง SKU_ID)
	QRData      string             `bson:"qrData" json:"qrData"`           // ข้อมูล QR
	ImageURL    *string            `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"`
	Price       Price              `bson:"price" json:"price"` // ข้อมูลราคา
	Stock       Stock              `bson:"stock" json:"stock"` // ข้อมูลสต็อก
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// ProductRequest represents the request body for creating/updating a product
type ProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Color       string  `json:"color"`
	Size        string  `json:"size"`
	Category    string  `json:"category"`
	ImageURL    *string `json:"imageUrl,omitempty"`
	Price       Price   `json:"price"`
	Stock       Stock   `json:"stock"`
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
	now := time.Now()
	return &Product{
		Name:        pr.Name,
		Description: pr.Description,
		Color:       pr.Color,
		Size:        pr.Size,
		Category:    pr.Category,
		ImageURL:    pr.ImageURL,
		Price:       pr.Price,
		Stock:       pr.Stock,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// UpdateFromRequest updates Product from ProductRequest
func (p *Product) UpdateFromRequest(pr *ProductRequest) {
	p.Name = pr.Name
	p.Description = pr.Description
	p.Color = pr.Color
	p.Size = pr.Size
	p.Category = pr.Category
	// Update ImageURL (allow null to delete image)
	p.ImageURL = pr.ImageURL
	p.Price = pr.Price
	p.Stock = pr.Stock
	p.UpdatedAt = time.Now()
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
func (p *Product) UpdatePrice(newPrice float64, isVAT bool, isPurchase bool) {
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

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

	// 4. อัปเดต average (เก่า+ใหม่)/2
	if priceInfo.Average == 0 {
		priceInfo.Average = newPrice
	} else {
		priceInfo.Average = (priceInfo.Average + newPrice) / 2
	}
	// ปัดเศษเป็น 2 ตำแหน่ง
	priceInfo.Average = float64(int(priceInfo.Average*100+0.5)) / 100

	// 5. อัปเดต YTD
	if priceInfo.YTDYear != currentYear {
		// ปีใหม่ - รีเซ็ตข้อมูล YTD
		priceInfo.YTDYear = currentYear
		priceInfo.YTDCount = 1
		priceInfo.YTDTotal = newPrice
		priceInfo.AverageYTD = newPrice
	} else {
		// ปีเดิม - อัปเดตข้อมูล YTD
		priceInfo.YTDCount++
		priceInfo.YTDTotal += newPrice
		priceInfo.AverageYTD = priceInfo.YTDTotal / float64(priceInfo.YTDCount)
		// ปัดเศษเป็น 2 ตำแหน่ง
		priceInfo.AverageYTD = float64(int(priceInfo.AverageYTD*100+0.5)) / 100
	}

	// 6. อัปเดต MTD
	if priceInfo.MTDYear != currentYear || priceInfo.MTDMonth != currentMonth {
		// เดือนใหม่ - รีเซ็ตข้อมูล MTD
		priceInfo.MTDYear = currentYear
		priceInfo.MTDMonth = currentMonth
		priceInfo.MTDCount = 1
		priceInfo.MTDTotal = newPrice
		priceInfo.AverageMTD = newPrice
	} else {
		// เดือนเดิม - อัปเดตข้อมูล MTD
		priceInfo.MTDCount++
		priceInfo.MTDTotal += newPrice
		priceInfo.AverageMTD = priceInfo.MTDTotal / float64(priceInfo.MTDCount)
		// ปัดเศษเป็น 2 ตำแหน่ง
		priceInfo.AverageMTD = float64(int(priceInfo.AverageMTD*100+0.5)) / 100
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
