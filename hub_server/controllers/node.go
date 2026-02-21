package controllers

import (
	"encoding/json"
	"net/http"
	"wx_channel/hub_server/database"
)

func GetNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := database.GetNodes()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}
