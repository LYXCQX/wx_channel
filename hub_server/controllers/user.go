package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"wx_channel/hub_server/database"
	"wx_channel/hub_server/middleware"
	"wx_channel/hub_server/models"

	"golang.org/x/crypto/bcrypt"
)

// ChangePassword allows users to change their password
func ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.ContextKeyUserID).(uint)

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    -1,
			"message": "Invalid request body",
		})
		return
	}

	// Validate input
	if req.OldPassword == "" || req.NewPassword == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    -1,
			"message": "旧密码和新密码不能为空",
		})
		return
	}

	if len(req.NewPassword) < 6 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    -1,
			"message": "新密码长度至少为 6 位",
		})
		return
	}

	// bcrypt 最多处理 72 字节
	if len(req.NewPassword) > 72 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    -1,
			"message": "密码过长（最多 72 字节）",
		})
		return
	}

	// Get user
	user, err := database.GetUserByID(userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    -1,
			"message": "用户不存在",
		})
		return
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    -1,
			"message": "旧密码错误",
		})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    -1,
			"message": "密码加密失败",
		})
		return
	}

	// Update password
	if err := database.UpdateUserPassword(user.ID, string(hashedPassword)); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    -1,
			"message": "密码更新失败",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"message": "密码修改成功",
	})
}

// GetTransactions returns the transaction history for the current user
func GetTransactions(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.ContextKeyUserID).(uint)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 20

	var transactions []models.Transaction
	var total int64

	database.DB.Model(&models.Transaction{}).Where("user_id = ?", userID).Count(&total)
	if err := database.DB.Where("user_id = ?", userID).
		Order("created_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&transactions).Error; err != nil {
		http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list":  transactions,
			"total": total,
			"page":  page,
		},
	})
}
