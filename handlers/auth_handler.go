package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"goodpack-server/middleware"
	"goodpack-server/models"
	"goodpack-server/repository"
	"goodpack-server/utils"
)

type AuthHandler struct {
	userRepo  *repository.UserRepository
	jwtSecret string
	jwtExpiry time.Duration
}

func NewAuthHandler(userRepo *repository.UserRepository, jwtSecret string, jwtExpiry time.Duration) *AuthHandler {
	return &AuthHandler{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		jwtExpiry: jwtExpiry,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"Username and password are required"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.FindByUsername(r.Context(), req.Username)
	if err != nil {
		http.Error(w, `{"error":"Invalid username or password"}`, http.StatusUnauthorized)
		return
	}

	if !user.IsActive {
		http.Error(w, `{"error":"Account is disabled"}`, http.StatusForbidden)
		return
	}

	if !user.CheckPassword(req.Password) {
		http.Error(w, `{"error":"Invalid username or password"}`, http.StatusUnauthorized)
		return
	}

	token, err := h.generateToken(user)
	if err != nil {
		http.Error(w, `{"error":"Failed to generate token"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.LoginResponse{
		Token: token,
		User:  *user,
	})
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		http.Error(w, `{"error":"Old password and new password are required"}`, http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 6 {
		http.Error(w, `{"error":"New password must be at least 6 characters"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}

	if !user.CheckPassword(req.OldPassword) {
		http.Error(w, `{"error":"Old password is incorrect"}`, http.StatusBadRequest)
		return
	}

	if err := user.SetPassword(req.NewPassword); err != nil {
		http.Error(w, `{"error":"Failed to update password"}`, http.StatusInternalServerError)
		return
	}
	user.UpdatedAt = utils.NowInThailand()

	if err := h.userRepo.Update(r.Context(), userID, user); err != nil {
		http.Error(w, `{"error":"Failed to save password"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Password updated successfully"})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	callerRole := middleware.GetUserRole(r)
	if callerRole != models.RoleSuperAdmin {
		http.Error(w, `{"error":"Super admin access required"}`, http.StatusForbidden)
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"Username and password are required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		http.Error(w, `{"error":"Password must be at least 6 characters"}`, http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = models.RoleSuperAdmin
	}

	if req.DisplayName == "" {
		req.DisplayName = req.Username
	}

	now := utils.NowInThailand()
	user := &models.User{
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Role:        req.Role,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := user.SetPassword(req.Password); err != nil {
		http.Error(w, `{"error":"Failed to hash password"}`, http.StatusInternalServerError)
		return
	}

	if err := h.userRepo.Create(r.Context(), user); err != nil {
		if isDuplicateKeyError(err) {
			http.Error(w, `{"error":"Username already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"Failed to create user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	callerRole := middleware.GetUserRole(r)
	if callerRole != models.RoleSuperAdmin {
		http.Error(w, `{"error":"Super admin access required"}`, http.StatusForbidden)
		return
	}

	users, err := h.userRepo.GetAll(r.Context())
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch users"}`, http.StatusInternalServerError)
		return
	}

	if users == nil {
		users = []*models.User{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	callerRole := middleware.GetUserRole(r)
	if callerRole != models.RoleSuperAdmin {
		http.Error(w, `{"error":"Super admin access required"}`, http.StatusForbidden)
		return
	}

	callerID := middleware.GetUserID(r)
	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, `{"error":"User ID is required"}`, http.StatusBadRequest)
		return
	}

	if callerID == userID {
		http.Error(w, `{"error":"Cannot delete your own account"}`, http.StatusBadRequest)
		return
	}

	if err := h.userRepo.Delete(r.Context(), userID); err != nil {
		http.Error(w, `{"error":"Failed to delete user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User deleted successfully"})
}

func (h *AuthHandler) generateToken(user *models.User) (string, error) {
	now := time.Now()
	claims := &jwt.RegisteredClaims{
		Subject:   user.ID.Hex(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(h.jwtExpiry)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

func isDuplicateKeyError(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") || contains(err.Error(), "E11000"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
