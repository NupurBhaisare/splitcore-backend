package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"

	"github.com/splitcore/backend/internal/database"
	"github.com/splitcore/backend/internal/migrations"
	"github.com/splitcore/backend/internal/routes"
)

func main() {
	os.Setenv("DATABASE_PATH", "/tmp/test_splitcore.db")
	os.Remove("/tmp/test_splitcore.db")

	if err := database.Init(); err != nil {
		fmt.Println("DB init failed:", err)
		return
	}
	defer database.Close()

	if err := migrations.RunAll(); err != nil {
		fmt.Println("Migration failed:", err)
		return
	}

	router := routes.NewRouter()

	// Register
	body, _ := json.Marshal(map[string]string{"email": "alice@test.com", "password": "password123"})
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	resp, _ := io.ReadAll(w.Body)
	fmt.Printf("Register: %d\n", w.Code)

	var authResp map[string]interface{}
	json.Unmarshal(resp, &authResp)
	token := authResp["data"].(map[string]interface{})["access_token"].(string)

	// Create Group
	body, _ = json.Marshal(map[string]string{"name": "Trip", "description": "", "icon_emoji": "🏖️", "currency_code": "USD"})
	req = httptest.NewRequest("POST", "/api/v1/groups", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	resp, _ = io.ReadAll(w.Body)
	fmt.Printf("CreateGroup: %d\n", w.Code)

	var grpResp map[string]interface{}
	json.Unmarshal(resp, &grpResp)
	groupID := grpResp["data"].(map[string]interface{})["group"].(map[string]interface{})["id"].(string)
	fmt.Printf("GroupID: %s\n", groupID)

	// Create Expense - use separate body buffers to avoid issues
	expBody, _ := json.Marshal(map[string]interface{}{
		"title":        "Dinner",
		"amount_cents": 5000,
		"currency_code": "USD",
		"category":     "food",
	})
	req = httptest.NewRequest("POST", "/api/v1/groups/"+groupID+"/expenses", bytes.NewBuffer(expBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	bodyOut, _ := io.ReadAll(w.Body)
	fmt.Printf("CreateExpense: status=%d body_len=%d\n", w.Code, len(bodyOut))
	if w.Code >= 300 {
		fmt.Printf("  Error body: %s\n", string(bodyOut))
	}

	// Get Balances
	req = httptest.NewRequest("GET", "/api/v1/groups/"+groupID+"/balances", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	bodyOut, _ = io.ReadAll(w.Body)
	fmt.Printf("GetBalances: status=%d body=%s\n", w.Code, string(bodyOut))

	fmt.Println("Done!")
}
