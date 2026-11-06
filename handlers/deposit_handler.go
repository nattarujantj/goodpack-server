package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"goodpack-server/models"
	"goodpack-server/repository"
	"goodpack-server/utils"
)

type DepositHandler struct {
	depositRepo  *repository.DepositRepository
	customerRepo *repository.CustomerRepository
	supplierRepo *repository.SupplierRepository
	ledger       *DepositLedger
}

func NewDepositHandler(depositRepo *repository.DepositRepository, customerRepo *repository.CustomerRepository, supplierRepo *repository.SupplierRepository, ledger *DepositLedger) *DepositHandler {
	return &DepositHandler{depositRepo: depositRepo, customerRepo: customerRepo, supplierRepo: supplierRepo, ledger: ledger}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// fillParty ใส่ชื่อ/รหัสลูกค้าหรือซัพพลายเออร์ให้เงินมัดจำ
func (h *DepositHandler) fillParty(d *models.Deposit) error {
	if d.Type == models.DepositTypeCustomer {
		c, err := h.customerRepo.GetByID(d.PartyID)
		if err != nil {
			return &DepositValidationError{Message: "ไม่พบลูกค้า"}
		}
		d.PartyName = c.CompanyName
		if d.PartyName == "" {
			d.PartyName = c.ContactName
		}
		d.PartyCode = c.CustomerCode
		return nil
	}
	s, err := h.supplierRepo.GetByID(d.PartyID)
	if err != nil {
		return &DepositValidationError{Message: "ไม่พบซัพพลายเออร์"}
	}
	d.PartyName = s.CompanyName
	if d.PartyName == "" {
		d.PartyName = s.ContactName
	}
	d.PartyCode = s.SupplierCode
	return nil
}

// generateDepositCode สร้างรหัสมัดจำ: DR = รับมัดจำจากลูกค้า, DP = จ่ายมัดจำให้ซัพพลายเออร์
// เช่น DR-6911-0001 (เลขลำดับต่อเนื่องข้ามเดือน)
func (h *DepositHandler) generateDepositCode(ctx context.Context, depositType string) (string, error) {
	prefix := "DR-"
	if depositType == models.DepositTypeSupplier {
		prefix = "DP-"
	}
	seq, err := h.depositRepo.GetNextSequenceNumber(ctx, prefix)
	if err != nil {
		return "", err
	}
	now := utils.NowInThailand()
	return fmt.Sprintf("%s%02d%02d-%04d", prefix, (now.Year()+543)%100, int(now.Month()), seq), nil
}

// GetDeposits GET /deposits?type=customer|supplier&partyId=...&excludeDocId=...&available=true
// excludeDocId: ไม่นับยอดที่รายการขาย/ซื้อนี้หักไว้ (ใช้ตอนแก้ไขรายการ)
// available=true: คืนเฉพาะมัดจำที่ยังมียอดคงเหลือ (หลังไม่นับ excludeDocId)
func (h *DepositHandler) GetDeposits(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	depositType := q.Get("type")
	deposits, err := h.depositRepo.GetAll(ctx, depositType, q.Get("partyId"))
	if err != nil {
		http.Error(w, "Failed to fetch deposits", http.StatusInternalServerError)
		return
	}

	excludeDocID := q.Get("excludeDocId")
	excludeDocType := ""
	if excludeDocID != "" {
		excludeDocType = models.DocTypeForDeposit(depositType)
	}
	if err := h.ledger.Enrich(ctx, deposits, excludeDocType, excludeDocID); err != nil {
		http.Error(w, "Failed to calculate deposit balances", http.StatusInternalServerError)
		return
	}

	if q.Get("available") == "true" {
		filtered := []*models.Deposit{}
		for _, d := range deposits {
			if d.RemainingAmount > 0.005 {
				filtered = append(filtered, d)
			}
		}
		deposits = filtered
	}
	writeJSON(w, http.StatusOK, deposits)
}

func (h *DepositHandler) loadDeposit(ctx context.Context, w http.ResponseWriter, id string) *models.Deposit {
	d, err := h.depositRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Deposit not found", http.StatusNotFound)
		return nil
	}
	if err := h.ledger.Enrich(ctx, []*models.Deposit{d}, "", ""); err != nil {
		http.Error(w, "Failed to calculate deposit balance", http.StatusInternalServerError)
		return nil
	}
	return d
}

func (h *DepositHandler) GetDeposit(w http.ResponseWriter, r *http.Request) {
	if d := h.loadDeposit(r.Context(), w, mux.Vars(r)["id"]); d != nil {
		writeJSON(w, http.StatusOK, d)
	}
}

func (h *DepositHandler) CreateDeposit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req models.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	deposit := req.ToDeposit()
	if err := h.fillParty(deposit); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.DepositCode != nil && strings.TrimSpace(*req.DepositCode) != "" {
		code := strings.TrimSpace(*req.DepositCode)
		if existing, err := h.depositRepo.GetByCode(ctx, code); err == nil && existing != nil {
			http.Error(w, "รหัสมัดจำนี้มีอยู่แล้ว", http.StatusBadRequest)
			return
		}
		deposit.DepositCode = code
	} else {
		code, err := h.generateDepositCode(ctx, deposit.Type)
		if err != nil {
			http.Error(w, "Failed to generate deposit code", http.StatusInternalServerError)
			return
		}
		deposit.DepositCode = code
	}

	if err := h.depositRepo.Create(ctx, deposit); err != nil {
		http.Error(w, "Failed to create deposit", http.StatusInternalServerError)
		return
	}
	deposit.ApplyUsages(nil)
	writeJSON(w, http.StatusCreated, deposit)
}

func (h *DepositHandler) UpdateDeposit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	existing := h.loadDeposit(ctx, w, id)
	if existing == nil {
		return
	}

	var req models.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(existing.Usages) > 0 {
		if req.Type != existing.Type || strings.TrimSpace(req.PartyID) != existing.PartyID || req.IsVAT != existing.IsVAT {
			http.Error(w, "มัดจำนี้ถูกหักในรายการซื้อ/ขายแล้ว ไม่สามารถเปลี่ยนประเภท คู่ค้า หรือการมี VAT ได้", http.StatusBadRequest)
			return
		}
		preview := req.ToDeposit()
		if preview.Deductible() < existing.UsedAmount-0.005 {
			http.Error(w, fmt.Sprintf("ยอดมัดจำต้องไม่น้อยกว่ายอดที่หักไปแล้ว %s บาท", formatBaht(existing.UsedAmount)), http.StatusBadRequest)
			return
		}
	}

	if req.DepositCode != nil && strings.TrimSpace(*req.DepositCode) != "" {
		code := strings.TrimSpace(*req.DepositCode)
		if code != existing.DepositCode {
			if len(existing.Usages) > 0 {
				http.Error(w, "มัดจำนี้ถูกหักในรายการซื้อ/ขายแล้ว ไม่สามารถเปลี่ยนรหัสได้", http.StatusBadRequest)
				return
			}
			if other, err := h.depositRepo.GetByCode(ctx, code); err == nil && other != nil && other.ID.Hex() != id {
				http.Error(w, "รหัสมัดจำนี้มีอยู่แล้ว", http.StatusBadRequest)
				return
			}
		}
	}

	usages := existing.Usages
	existing.UpdateFromRequest(&req)
	if err := h.fillParty(existing); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.depositRepo.Update(ctx, id, existing); err != nil {
		http.Error(w, "Failed to update deposit", http.StatusInternalServerError)
		return
	}
	existing.ApplyUsages(usages)
	writeJSON(w, http.StatusOK, existing)
}

func (h *DepositHandler) DeleteDeposit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	existing := h.loadDeposit(ctx, w, id)
	if existing == nil {
		return
	}
	if len(existing.Usages) > 0 {
		codes := make([]string, 0, len(existing.Usages))
		for _, u := range existing.Usages {
			codes = append(codes, u.DocCode)
		}
		http.Error(w, "มัดจำนี้ถูกหักในรายการ "+strings.Join(codes, ", ")+" แล้ว กรุณาเอาการหักมัดจำออกจากรายการเหล่านั้นก่อนลบ", http.StatusConflict)
		return
	}
	if err := h.depositRepo.Delete(ctx, id); err != nil {
		http.Error(w, "Failed to delete deposit", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
