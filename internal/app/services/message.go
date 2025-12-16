package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Message struct {
	ID         int64     `json:"id"`
	RoomID     string    `json:"room_id"`
	SenderID   string    `json:"sender_id"`
	SenderName string    `json:"sender_name,omitempty"`
	Body       string    `json:"body"`
	Type       string    `json:"type"`
	SentAt     time.Time `json:"sent_at"`
}

// CreateMessage inserts a new message into Supabase messages table.
func CreateMessage(roomID, senderID, senderName, body string) (*Message, error) {
	loadEnv()

	payload := map[string]interface{}{
		"room_id":   roomID,
		"sender_id": senderID,
		"body":      body,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal message payload: %w", err)
	}

	req, _ := http.NewRequest("POST", supabaseURL+"/rest/v1/messages", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+supabaseAPIKey)
	req.Header.Set("apikey", supabaseAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 { // 201 Created expected with return=representation
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase message insert failed (status %d): %s", resp.StatusCode, bodyBytes)
	}

	var rows []Message
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode message response: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("message insert returned no rows")
	}

	msg := rows[0]
	msg.SenderName = senderName
	return &msg, nil
}

// ListMessages returns messages for a room ordered newest-first, with optional before-id pagination.
func ListMessages(roomID, beforeID string, limit int, includeSystem bool) ([]Message, bool, error) {
	loadEnv()

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	q := url.Values{}
	q.Set("room_id", "eq."+roomID)
	q.Set("select", "id,room_id,sender_id,body,type,sent_at")
	q.Set("order", "sent_at.desc,id.desc")
	q.Set("limit", fmt.Sprintf("%d", limit+1))
	if beforeID != "" {
		q.Set("id", "lt."+beforeID)
	}
	if !includeSystem {
		q.Set("type", "eq.text")
	}

	endpoint := fmt.Sprintf("%s/rest/v1/messages?%s", supabaseURL, q.Encode())
	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+supabaseAPIKey)
	req.Header.Set("apikey", supabaseAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("fetch messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("fetch messages failed (status %d): %s", resp.StatusCode, bodyBytes)
	}

	var rows []struct {
		ID       int64     `json:"id"`
		RoomID   string    `json:"room_id"`
		SenderID string    `json:"sender_id"`
		Body     string    `json:"body"`
		Type     string    `json:"type"`
		SentAt   time.Time `json:"sent_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, false, fmt.Errorf("decode messages: %w", err)
	}

	msgs := make([]Message, 0, len(rows))
	nameCache := make(map[string]string)
	for _, r := range rows {
		m := Message{
			ID:       r.ID,
			RoomID:   r.RoomID,
			SenderID: r.SenderID,
			Body:     r.Body,
			Type:     r.Type,
			SentAt:   r.SentAt,
		}

		if cached, ok := nameCache[r.SenderID]; ok {
			m.SenderName = cached
		} else {
			if name, err := fetchSenderName(r.SenderID); err == nil {
				m.SenderName = name
				nameCache[r.SenderID] = name
			}
		}
		msgs = append(msgs, m)
	}

	hasMore := false
	if len(msgs) > limit {
		hasMore = true
		msgs = msgs[:limit]
	}

	return msgs, hasMore, nil
}

// fetchSenderName gets user_name from auth admin endpoint using service key.
func fetchSenderName(userID string) (string, error) {
	if supabaseAPIKey == "" || supabaseURL == "" {
		return "", fmt.Errorf("supabase config missing")
	}
	endpoint := fmt.Sprintf("%s/auth/v1/admin/users/%s", supabaseURL, userID)
	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+supabaseAPIKey)
	req.Header.Set("apikey", supabaseAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("fetch user failed status %d: %s", resp.StatusCode, bodyBytes)
	}

	var user struct {
		UserMetadata map[string]interface{} `json:"user_metadata"`
		RawMeta      map[string]interface{} `json:"raw_user_meta_data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", err
	}

	if name, ok := user.UserMetadata["user_name"].(string); ok {
		return name, nil
	}
	if name, ok := user.RawMeta["user_name"].(string); ok {
		return name, nil
	}
	return "", nil
}
