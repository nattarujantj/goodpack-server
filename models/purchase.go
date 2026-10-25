package models

import (
	"time"

	"goodpack-server/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Purchase struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PurchaseCode      string             `bson:"purchaseCode" json:"purchaseCode"`
	InvoiceNumber     *string            `bson:"invoiceNumber,omitempty" json:"invoiceNumber,omitempty"`
	CreatedAt         time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt" json:"updatedAt"`
	PurchaseDate      time.Time          `bson:"purchaseDate" json:"purchaseDate"`
	SupplierID        string             `bson:"supplierId" json:"supplierId"`
	SupplierName      string             `bson:"supplierName" json:"supplierName"`
	ContactName       *string            `bson:"contactName,omitempty" json:"contactName,omitempty"`
	SupplierCode      *string            `bson:"supplierCode,omitempty" json:"supplierCode,omitempty"`
	TaxID             *string            `bson:"taxId,omitempty" json:"taxId,omitempty"`
	Address           *string            `bson:"address,omitempty" json:"address,omitempty"`
	Phone             *string            `bson:"phone,omitempty" json:"phone,omitempty"`
	Notes             *string            `bson:"notes,omitempty" json:"notes,omitempty"`
	Items             []PurchaseItem     `bson:"items" json:"items"`
	IsVAT             bool               `bson:"isVAT" json:"isVAT"`
	VATType           string             `bson:"vatType" json:"vatType"` // "exclusive" (VAT นอก), "inclusive" (VAT ใน) or "import" (VAT ตามใบขน)
	ImportCustoms     *ImportCustomsInfo `bson:"importCustoms,omitempty" json:"importCustoms,omitempty"`
	ShippingCost      float64            `bson:"shippingCost" json:"shippingCost"`
	Payment           PaymentInfo        `bson:"payment" json:"payment"`
	Warehouse         WarehouseInfo      `bson:"warehouse" json:"warehouse"`
	PurchaseOrderCode *string            `bson:"purchaseOrderCode,omitempty" json:"purchaseOrderCode,omitempty"`
	TotalAmount       float64            `bson:"totalAmount" json:"totalAmount"`
	TotalVAT          float64            `bson:"totalVAT" json:"totalVAT"`
	GrandTotal        float64            `bson:"grandTotal" json:"grandTotal"`
}

type PurchaseItem struct {
	ProductID        string   `bson:"productId" json:"productId"`
	ProductName      string   `bson:"productName" json:"productName"`
	ProductCode      string   `bson:"productCode" json:"productCode"`
	Quantity         int      `bson:"quantity" json:"quantity"`
	UnitPrice        float64  `bson:"unitPrice" json:"unitPrice"`
	PreformProductID *string  `bson:"preformProductId,omitempty" json:"preformProductId,omitempty"`
	PreformUnitPrice *float64 `bson:"preformUnitPrice,omitempty" json:"preformUnitPrice,omitempty"`
	// CommissionPerUnit คือค่าคอมต่อชิ้น (เช่น จากรายการนำเข้า) ซึ่งไม่คิด VAT
	// จึงไม่รวมใน UnitPrice/TotalPrice แต่นับเป็นต้นทุนจริงของสินค้า
	CommissionPerUnit *float64 `bson:"commissionPerUnit,omitempty" json:"commissionPerUnit,omitempty"`
	// ImportVAT คือ VAT จริงตามใบขนสินค้าของทั้งรายการนี้ (บาท) ใช้เฉพาะ vatType = "import"
	ImportVAT  *float64 `bson:"importVAT,omitempty" json:"importVAT,omitempty"`
	TotalPrice float64  `bson:"totalPrice" json:"totalPrice"`
}

// VATTypeImport คือรายการซื้อจากการนำเข้า: VAT ไม่ได้คิด 7% จากราคา แต่ใช้ตัวเลขจริงตามใบขนสินค้า
// สต็อกเข้าสต็อก VAT แต่ต้นทุนสินค้าเก็บแยกที่ Price.PurchaseImport
const VATTypeImport = "import"

// ImportCustomsInfo คือข้อมูลใบขนสินค้าที่ใช้คำนวณ VAT ของรายการซื้อนำเข้า
type ImportCustomsInfo struct {
	ImportID        string     `bson:"importId" json:"importId"`
	ImportCode      string     `bson:"importCode" json:"importCode"`
	DeclarationNo   string     `bson:"declarationNo" json:"declarationNo"`
	DeclarationDate *time.Time `bson:"declarationDate,omitempty" json:"declarationDate,omitempty"`
	GoodsVAT        float64    `bson:"goodsVAT" json:"goodsVAT"`     // VAT ค่าของ
	FreightVAT      float64    `bson:"freightVAT" json:"freightVAT"` // VAT ค่าเฟรท
	ImportDuty      float64    `bson:"importDuty" json:"importDuty"` // อากรขาเข้า (รวมอยู่ในราคาต่อชิ้นแล้ว)
}

// IsImport บอกว่าเป็นรายการซื้อนำเข้า (VAT ตามใบขน) หรือไม่
func (p *Purchase) IsImport() bool {
	return p.VATType == VATTypeImport
}

// ImportCostPerUnit คือต้นทุนนำเข้าต่อชิ้น = ต้นทุนก่อน VAT + VAT จริงต่อชิ้น
func (item PurchaseItem) ImportCostPerUnit() float64 {
	price := item.EffectiveUnitPrice()
	if item.ImportVAT != nil && item.Quantity > 0 {
		price += *item.ImportVAT / float64(item.Quantity)
	}
	return price
}

// EffectiveUnitPrice คือต้นทุนจริงต่อชิ้นที่ใช้คำนวณราคาซื้อของสินค้า:
// ราคาต่อชิ้น + ราคาพรีฟอร์ม + ค่าคอม
func (item PurchaseItem) EffectiveUnitPrice() float64 {
	price := item.UnitPrice
	if item.PreformUnitPrice != nil {
		price += *item.PreformUnitPrice
	}
	if item.CommissionPerUnit != nil {
		price += *item.CommissionPerUnit
	}
	return price
}

type PaymentInfo struct {
	IsPaid          bool         `bson:"isPaid" json:"isPaid"`
	PaymentMethod   *string      `bson:"paymentMethod,omitempty" json:"paymentMethod,omitempty"`
	OurAccount      *string      `bson:"ourAccount,omitempty" json:"ourAccount,omitempty"`
	OurAccountInfo  *BankAccount `bson:"ourAccountInfo,omitempty" json:"ourAccountInfo,omitempty"`
	CustomerAccount *string      `bson:"customerAccount,omitempty" json:"customerAccount,omitempty"`
	PaymentDate     *time.Time   `bson:"paymentDate,omitempty" json:"paymentDate,omitempty"`
}

type BankAccount struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AccountNumber string `json:"accountNumber"`
	BankName      string `json:"bankName"`
	AccountType   string `json:"accountType"`
	IsActive      bool   `json:"isActive"`
}

type WarehouseInfo struct {
	IsUpdated      bool            `bson:"isUpdated" json:"isUpdated"`
	Notes          *string         `bson:"notes,omitempty" json:"notes,omitempty"`
	ActualShipping float64         `bson:"actualShipping" json:"actualShipping"`
	Items          []WarehouseItem `bson:"items" json:"items"`
}

type WarehouseItem struct {
	ProductID   string  `bson:"productId" json:"productId"`
	ProductName string  `bson:"productName" json:"productName"`
	Quantity    int     `bson:"quantity" json:"quantity"`
	Boxes       int     `bson:"boxes" json:"boxes"`
	Notes       *string `bson:"notes,omitempty" json:"notes,omitempty"`
}

type PurchaseRequest struct {
	PurchaseCode      *string            `json:"purchaseCode,omitempty" bson:"purchaseCode,omitempty"`
	PurchaseDate      time.Time          `json:"purchaseDate" bson:"purchaseDate"`
	SupplierID        string             `json:"supplierId" bson:"supplierId"`
	InvoiceNumber     *string            `json:"invoiceNumber,omitempty" bson:"invoiceNumber,omitempty"`
	Notes             *string            `json:"notes,omitempty" bson:"notes,omitempty"`
	Items             []PurchaseItem     `json:"items" bson:"items"`
	IsVAT             bool               `json:"isVAT" bson:"isVAT"`
	VATType           string             `json:"vatType" bson:"vatType"` // "exclusive", "inclusive" or "import"
	ImportCustoms     *ImportCustomsInfo `json:"importCustoms,omitempty" bson:"importCustoms,omitempty"`
	ShippingCost      float64            `json:"shippingCost" bson:"shippingCost"`
	Payment           PaymentInfo        `json:"payment" bson:"payment"`
	Warehouse         WarehouseInfo      `json:"warehouse" bson:"warehouse"`
	PurchaseOrderCode *string            `json:"purchaseOrderCode,omitempty" bson:"purchaseOrderCode,omitempty"`
}

// calculateTotals returns the pre-VAT total, VAT and grand total for the request.
func (pr *PurchaseRequest) calculateTotals() (totalAmount, totalVAT, grandTotal float64) {
	if pr.VATType == VATTypeImport {
		// รายการนำเข้าเป็น VAT เสมอ
		pr.IsVAT = true
	} else {
		pr.ImportCustoms = nil
	}

	for _, item := range pr.Items {
		totalAmount += item.TotalPrice
	}

	if !pr.IsVAT {
		return totalAmount, 0, totalAmount
	}
	switch pr.VATType {
	case "inclusive":
		// VAT ใน: ราคารวม VAT แล้ว ต้องถอด VAT ออก
		// ราคาก่อน VAT = ราคารวม / 1.07
		// VAT = ราคารวม - ราคาก่อน VAT
		totalVAT = utils.RoundTo2(totalAmount - (totalAmount / 1.07))
		grandTotal = totalAmount // ราคาที่กรอกคือราคารวม VAT แล้ว
	case VATTypeImport:
		// นำเข้า: VAT จริงตามใบขนสินค้า (ไม่คิด 7% จากราคา)
		for _, item := range pr.Items {
			if item.ImportVAT != nil {
				totalVAT += *item.ImportVAT
			}
		}
		totalVAT = utils.RoundTo2(totalVAT)
		grandTotal = totalAmount + totalVAT
	default:
		// VAT นอก (exclusive): ราคา + VAT 7%
		totalVAT = utils.RoundTo2(totalAmount * 0.07)
		grandTotal = totalAmount + totalVAT
	}
	return totalAmount, totalVAT, grandTotal
}

func (pr *PurchaseRequest) ToPurchase() *Purchase {
	now := utils.NowInThailand()

	totalAmount, totalVAT, grandTotal := pr.calculateTotals()

	return &Purchase{
		PurchaseCode:      "", // Will be populated by handler
		InvoiceNumber:     pr.InvoiceNumber,
		CreatedAt:         now,
		UpdatedAt:         now,
		PurchaseDate:      pr.PurchaseDate,
		SupplierID:        pr.SupplierID,
		SupplierName:      "",  // Will be populated from supplier data
		ContactName:       nil, // Will be populated from supplier data
		SupplierCode:      nil, // Will be populated from supplier data
		TaxID:             nil, // Will be populated from supplier data
		Address:           nil, // Will be populated from supplier data
		Phone:             nil, // Will be populated from supplier data
		Notes:             pr.Notes,
		Items:             pr.Items,
		IsVAT:             pr.IsVAT,
		VATType:           pr.VATType,
		ImportCustoms:     pr.ImportCustoms,
		ShippingCost:      pr.ShippingCost,
		Payment:           pr.Payment,
		Warehouse:         pr.Warehouse,
		PurchaseOrderCode: pr.PurchaseOrderCode,
		TotalAmount:       totalAmount,
		TotalVAT:          totalVAT,
		GrandTotal:        grandTotal,
	}
}

func (p *Purchase) UpdateFromRequest(pr *PurchaseRequest) {
	totalAmount, totalVAT, grandTotal := pr.calculateTotals()

	if pr.PurchaseCode != nil && *pr.PurchaseCode != "" {
		p.PurchaseCode = *pr.PurchaseCode
	}
	p.PurchaseDate = pr.PurchaseDate
	p.SupplierID = pr.SupplierID
	p.InvoiceNumber = pr.InvoiceNumber
	p.Notes = pr.Notes
	p.Items = pr.Items
	p.IsVAT = pr.IsVAT
	p.VATType = pr.VATType
	p.ImportCustoms = pr.ImportCustoms
	p.ShippingCost = pr.ShippingCost
	p.Payment = pr.Payment
	p.Warehouse = pr.Warehouse
	p.PurchaseOrderCode = pr.PurchaseOrderCode
	p.TotalAmount = totalAmount
	p.TotalVAT = totalVAT
	p.GrandTotal = grandTotal
	p.UpdatedAt = utils.NowInThailand()
}
