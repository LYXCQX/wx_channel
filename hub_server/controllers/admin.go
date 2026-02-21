package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"wx_channel/hub_server/database"

	"github.com/gorilla/mux"
)

// GetUserList returns a list of all users (admin only)
func GetUserList(w http.ResponseWriter, r *http.Request) {
	users, _, err := database.GetAllUsers(0, 100)
	if err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"list": users,
	})
}

// GetStats returns system statistics (admin only)
func GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := database.GetSystemStats()
	if err != nil {
		http.Error(w, "Failed to fetch stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// UpdateUserCredits adjusts a user's credits (admin only)
func UpdateUserCredits(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID     uint  `json:"user_id"`
		Adjustment int64 `json:"adjustment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == 0 {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	if req.Adjustment == 0 {
		http.Error(w, "adjustment cannot be zero", http.StatusBadRequest)
		return
	}

	// 更新积分
	if err := database.AddCredits(req.UserID, req.Adjustment); err != nil {
		http.Error(w, "Failed to update credits", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Credits updated successfully",
	})
}

// UpdateUserRole changes a user's role (admin only)
func UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID uint   `json:"user_id"`
		Role   string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == 0 {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	if req.Role != "user" && req.Role != "admin" {
		http.Error(w, "role must be 'user' or 'admin'", http.StatusBadRequest)
		return
	}

	// 更新角色
	if err := database.UpdateUserRole(req.UserID, req.Role); err != nil {
		http.Error(w, "Failed to update role", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Role updated successfully",
	})
}

// DeleteUser permanently deletes a user (admin only)
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// 删除用户
	if err := database.DeleteUser(uint(userID)); err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User deleted successfully",
	})
}


// GetAllDevices returns all devices in the system (admin only)
func GetAllDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := database.GetAllDevices()
	if err != nil {
		http.Error(w, "Failed to fetch devices", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

// AdminUnbindDevice unbinds a device from its user (admin only)
func AdminUnbindDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID string `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}

	// 解绑设备
	if err := database.UnbindNode(req.DeviceID); err != nil {
		http.Error(w, "Failed to unbind device", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Device unbound successfully",
	})
}

// AdminDeleteDevice permanently deletes a device (admin only)
func AdminDeleteDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	if deviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}

	// 删除设备
	if err := database.DeleteNode(deviceID); err != nil {
		http.Error(w, "Failed to delete device", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Device deleted successfully",
	})
}

// GetAllTasks returns all tasks in the system (admin only)
func GetAllTasks(w http.ResponseWriter, r *http.Request) {
	tasks, count, err := database.GetAllTasks()
	if err != nil {
		http.Error(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"list":  tasks,
		"total": count,
	})
}

// AdminDeleteTask permanently deletes a task (admin only)
func AdminDeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskIDStr := vars["id"]

	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// 删除任务
	if err := database.DeleteTask(uint(taskID)); err != nil {
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Task deleted successfully",
	})
}

// GetAllSubscriptions returns all subscriptions in the system (admin only)
func GetAllSubscriptions(w http.ResponseWriter, r *http.Request) {
	subscriptions, err := database.GetAllSubscriptions()
	if err != nil {
		http.Error(w, "Failed to fetch subscriptions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriptions)
}

// AdminDeleteSubscription permanently deletes a subscription (admin only)
func AdminDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subIDStr := vars["id"]

	subID, err := strconv.ParseUint(subIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid subscription ID", http.StatusBadRequest)
		return
	}

	// 删除订阅
	if err := database.DeleteSubscription(uint(subID)); err != nil {
		http.Error(w, "Failed to delete subscription", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Subscription deleted successfully",
	})
}
