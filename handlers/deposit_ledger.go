package handlers

import (
	"context"
	"fmt"
	"sort"
	"time"

	"goodpack-server/models"
	"goodpack-server/repository"
	"goodpack-server/utils"
)

// DepositLedger คำนวณยอดที่ใช้ไป/คงเหลือของเงินมัดจำจากรายการขาย/ซื้อที่หักมัดจำ
// และตรวจสอบการหักมัดจำตอนบันทึกรายการขาย/ซื้อ
type DepositLedger struct {
	depositRepo  *repository.DepositRepository
	saleRepo     *repository.SaleRepository
	purchaseRepo *repository.PurchaseRepository
}

func NewDepositLedger(depositRepo *repository.DepositRepository, saleRepo *repository.SaleRepository, purchaseRepo *repository.PurchaseRepository) *DepositLedger {
	return &DepositLedger{depositRepo: depositRepo, saleRepo: saleRepo, purchaseRepo: purchaseRepo}
}

// DepositValidationError คือข้อผิดพลาดจากข้อมูลที่ผู้ใช้ส่งมา (ตอบกลับเป็น 400)
type DepositValidationError struct{ Message string }

func (e *DepositValidationError) Error() string { return e.Message }

// Enrich ใส่ประวัติการหักและยอดคงเหลือให้เงินมัดจำ
// excludeDocType/excludeDocID ใช้ตอนแก้ไขรายการขาย/ซื้อ เพื่อไม่นับยอดที่เอกสารนั้นหักไว้เดิม
func (l *DepositLedger) Enrich(ctx context.Context, deposits []*models.Deposit, excludeDocType, excludeDocID string) error {
	if len(deposits) == 0 {
		return nil
	}
	ids := make([]string, 0, len(deposits))
	for _, d := range deposits {
		ids = append(ids, d.ID.Hex())
	}

	usages := map[string][]models.DepositUsage{}
	add := func(docType, docID, docCode string, apps []models.DepositApplication, docDate time.Time) {
		if docType == excludeDocType && docID == excludeDocID {
			return
		}
		for _, a := range apps {
			usages[a.DepositID] = append(usages[a.DepositID], models.DepositUsage{
				DocType: docType, DocID: docID, DocCode: docCode, DocDate: docDate, Amount: a.Amount,
			})
		}
	}

	sales, err := l.saleRepo.GetByDepositIDs(ctx, ids)
	if err != nil {
		return err
	}
	for _, s := range sales {
		add(models.DepositDocSale, s.ID.Hex(), s.SaleCode, s.DepositApplications, s.SaleDate)
	}
	purchases, err := l.purchaseRepo.GetByDepositIDs(ctx, ids)
	if err != nil {
		return err
	}
	for _, p := range purchases {
		add(models.DepositDocPurchase, p.ID.Hex(), p.PurchaseCode, p.DepositApplications, p.PurchaseDate)
	}

	for _, d := range deposits {
		list := usages[d.ID.Hex()]
		sort.Slice(list, func(i, j int) bool { return list[i].DocDate.Before(list[j].DocDate) })
		d.ApplyUsages(list)
	}
	return nil
}

// ResolveApplications ตรวจและเติมข้อมูลการหักมัดจำของรายการขาย/ซื้อ:
// มัดจำต้องเป็นประเภทที่ถูกต้อง เป็นของลูกค้า/ซัพพลายเออร์เจ้าเดียวกับเอกสาร
// และยอดที่หักต้องไม่เกินยอดคงเหลือ (ไม่นับยอดที่เอกสารนี้หักไว้เดิม)
// allowVATDeposits: เอกสารเป็น VAT (VAT นอก/VAT ใน) จึงใช้มัดจำที่ออกใบกำกับภาษีแล้วได้ (หักก่อน VAT)
func (l *DepositLedger) ResolveApplications(ctx context.Context, apps []models.DepositApplication, depositType, partyID, docID string, allowVATDeposits bool) ([]models.DepositApplication, error) {
	apps = models.NormalizeDepositApplications(apps)
	if len(apps) == 0 {
		return []models.DepositApplication{}, nil
	}

	deposits := make([]*models.Deposit, 0, len(apps))
	for _, a := range apps {
		d, err := l.depositRepo.GetByID(ctx, a.DepositID)
		if err != nil {
			return nil, &DepositValidationError{Message: "ไม่พบเงินมัดจำที่เลือก"}
		}
		if d.Type != depositType {
			return nil, &DepositValidationError{Message: fmt.Sprintf("เงินมัดจำ %s ไม่ใช่ประเภทที่ใช้กับรายการนี้", d.DepositCode)}
		}
		if d.PartyID != partyID {
			return nil, &DepositValidationError{Message: fmt.Sprintf("เงินมัดจำ %s เป็นของ %s ไม่ใช่คู่ค้าของรายการนี้", d.DepositCode, d.PartyName)}
		}
		if d.IsVAT && !allowVATDeposits {
			return nil, &DepositValidationError{Message: fmt.Sprintf("เงินมัดจำ %s มี VAT (ออกใบกำกับภาษีแล้ว) ใช้หักได้เฉพาะรายการที่เป็น VAT นอก/VAT ใน", d.DepositCode)}
		}
		deposits = append(deposits, d)
	}

	if err := l.Enrich(ctx, deposits, models.DocTypeForDeposit(depositType), docID); err != nil {
		return nil, err
	}

	for i, d := range deposits {
		if !d.CanCover(apps[i].Amount) {
			return nil, &DepositValidationError{Message: fmt.Sprintf(
				"เงินมัดจำ %s คงเหลือ %s บาท ไม่พอหัก %s บาท",
				d.DepositCode, formatBaht(d.RemainingAmount), formatBaht(apps[i].Amount))}
		}
		apps[i].DepositCode = d.DepositCode
		apps[i].BeforeVAT = d.IsVAT
	}
	return apps, nil
}

func formatBaht(v float64) string {
	return fmt.Sprintf("%.2f", utils.RoundTo2(v))
}
