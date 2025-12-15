package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Room struct {
	ID        string    `json:"id,omitempty"`
	Code      string    `json:"code"`
	OwnerID   string    `json:"owner_id"`
	Title     string    `json:"title"`
	IsPrivate bool      `json:"is_private"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// CreateRoom inserts a new room into Supabase database using the create_room_with_owner function.
func CreateRoom(ownerID, title string, isPrivate bool) (*Room, error) {
	loadEnv()

	// Call the PostgreSQL function with owner_id parameter
	payload := map[string]interface{}{
		"_owner_id":   ownerID,
		"_title":      title,
		"_is_private": isPrivate,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, _ := http.NewRequest("POST", supabaseURL+"/rest/v1/rpc/create_room_with_owner", bytes.NewReader(payloadBytes))
	req.Header.Set("Authorization", "Bearer "+supabaseAnonKey)
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase create room failed (status %d): %s", resp.StatusCode, body)
	}

	// The function returns {"room_id": "...", "code": "..."}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode room response: %w", err)
	}

	return &Room{
		ID:        result["room_id"],
		Code:      result["code"],
		OwnerID:   ownerID,
		Title:     title,
		IsPrivate: isPrivate,
	}, nil
}
