package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// JoinRoomByCode looks up room by code and inserts membership. Returns room details.
func JoinRoomByCode(code, userID string) (*Room, error) {
	loadEnv()

	// Step 1: find room details by code
	findReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/rest/v1/rooms?code=eq.%s&select=id,code,owner_id,title,is_private,created_at&limit=1", supabaseURL, code), nil)
	findReq.Header.Set("Authorization", "Bearer "+supabaseAPIKey)
	findReq.Header.Set("apikey", supabaseAPIKey)

	findResp, err := http.DefaultClient.Do(findReq)
	if err != nil {
		return nil, fmt.Errorf("lookup room by code: %w", err)
	}
	defer findResp.Body.Close()

	if findResp.StatusCode != 200 {
		body, _ := io.ReadAll(findResp.Body)
		return nil, fmt.Errorf("lookup room failed (status %d): %s", findResp.StatusCode, body)
	}

	var rooms []struct {
		ID        string    `json:"id"`
		Code      string    `json:"code"`
		OwnerID   string    `json:"owner_id"`
		Title     string    `json:"title"`
		IsPrivate bool      `json:"is_private"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.NewDecoder(findResp.Body).Decode(&rooms); err != nil {
		return nil, fmt.Errorf("decode room lookup: %w", err)
	}
	if len(rooms) == 0 {
		return nil, fmt.Errorf("room not found")
	}
	room := rooms[0]
	roomID := room.ID

	// Step 2: insert membership (idempotent upsert on PK room_id+account_id)
	payload := map[string]interface{}{
		"room_id":    roomID,
		"account_id": userID,
		"role":       "member",
	}
	body, _ := json.Marshal(payload)

	insertReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/rest/v1/room_members", supabaseURL), bytes.NewReader(body))
	insertReq.Header.Set("Authorization", "Bearer "+supabaseAPIKey)
	insertReq.Header.Set("apikey", supabaseAPIKey)
	insertReq.Header.Set("Content-Type", "application/json")
	insertReq.Header.Set("Prefer", "resolution=ignore-duplicates")

	insertResp, err := http.DefaultClient.Do(insertReq)
	if err != nil {
		return nil, fmt.Errorf("insert membership: %w", err)
	}
	defer insertResp.Body.Close()

	if insertResp.StatusCode != 201 && insertResp.StatusCode != 204 {
		b, _ := io.ReadAll(insertResp.Body)
		return nil, fmt.Errorf("insert membership failed (status %d): %s", insertResp.StatusCode, b)
	}

	return &Room{
		ID:        room.ID,
		Code:      room.Code,
		OwnerID:   room.OwnerID,
		Title:     room.Title,
		IsPrivate: room.IsPrivate,
		CreatedAt: room.CreatedAt,
	}, nil
}
