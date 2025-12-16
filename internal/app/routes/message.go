package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"musick-server/internal/app/services"

	"github.com/DarthPestilane/easytcp"
)

type SendMessageRequest struct {
	UserID string `json:"user_id"`
	RoomID string `json:"room_id"`
	Body   string `json:"body"`
}

type SendMessageResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	ID         int64  `json:"id,omitempty"`
	RoomID     string `json:"room_id,omitempty"`
	SenderID   string `json:"sender_id,omitempty"`
	SenderName string `json:"sender_name,omitempty"`
	Body       string `json:"body,omitempty"`
	SentAt     string `json:"sent_at,omitempty"`
}

type FetchMessagesRequest struct {
	RoomID        string `json:"room_id"`
	BeforeID      string `json:"before_id"`
	Limit         int    `json:"limit"`
	UserID        string `json:"user_id"`
	IncludeSystem bool   `json:"include_system"`
}

type FetchMessagesResponse struct {
	Success             bool             `json:"success"`
	Message             string           `json:"message"`
	Messages            []FetchedMessage `json:"messages,omitempty"`
	HasMore             bool             `json:"has_more"`
	NextBeforeID        string           `json:"next_before_id,omitempty"`
	NextBeforeCreatedAt string           `json:"next_before_created_at,omitempty"`
}

type FetchedMessage struct {
	ID         int64  `json:"id"`
	RoomID     string `json:"room_id"`
	SenderID   string `json:"sender_id"`
	SenderName string `json:"sender_name,omitempty"`
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
}

func RegisterMessageRoutes(s *easytcp.Server) {
	s.AddRoute(301, handleSendMessage)
	s.AddRoute(310, handleFetchMessages)
}

func handleSendMessage(ctx easytcp.Context) {
	req := ctx.Request()
	log.Printf("301 send message: id=%d bytes=%d", req.ID(), len(req.Data()))

	if !services.IsAuthenticated(ctx.Session()) {
		sendMessageError(ctx, "not authenticated")
		return
	}

	var msgReq SendMessageRequest
	if err := json.Unmarshal(req.Data(), &msgReq); err != nil {
		sendMessageError(ctx, "invalid request format")
		return
	}

	if msgReq.UserID == "" || msgReq.RoomID == "" || msgReq.Body == "" {
		sendMessageError(ctx, "user_id, room_id, and body are required")
		return
	}

	session := services.GetSession(ctx.Session())
	if session == nil || session.UserID != msgReq.UserID {
		sendMessageError(ctx, "user_id mismatch")
		return
	}

	saved, err := services.CreateMessage(msgReq.RoomID, msgReq.UserID, session.UserName, msgReq.Body)

	if err != nil {
		log.Printf("failed to send message: %v", err)
		sendMessageError(ctx, "failed to send message")
		return
	}

	log.Printf("301 send message: saved id=%d room=%s sender=%s", saved.ID, saved.RoomID, saved.SenderID)

	// Ensure sender is tracked in the room for broadcasts.
	services.AddSessionToRoom(msgReq.RoomID, ctx.Session())

	// Broadcast to all sessions in the room (including sender) on route 302.
	broadcast := SendMessageResponse{
		Success:    true,
		Message:    "message delivered",
		ID:         saved.ID,
		RoomID:     saved.RoomID,
		SenderID:   saved.SenderID,
		SenderName: saved.SenderName,
		Body:       saved.Body,
		SentAt:     saved.SentAt.Format(time.RFC3339),
	}
	if b, err := json.Marshal(broadcast); err == nil {
		services.BroadcastToRoom(msgReq.RoomID, easytcp.NewMessage(302, b), nil)
	}

	resp := SendMessageResponse{
		Success:    true,
		Message:    "message sent",
		ID:         saved.ID,
		RoomID:     saved.RoomID,
		SenderID:   saved.SenderID,
		SenderName: saved.SenderName,
		Body:       saved.Body,
		SentAt:     saved.SentAt.Format(time.RFC3339),
	}

	data, _ := json.Marshal(resp)
	ctx.SetResponseMessage(easytcp.NewMessage(req.ID(), data))
}

func handleFetchMessages(ctx easytcp.Context) {
	req := ctx.Request()
	log.Printf("310 fetch messages: id=%d bytes=%d", req.ID(), len(req.Data()))

	if !services.IsAuthenticated(ctx.Session()) {
		sendFetchMessagesError(ctx, "not authenticated")
		return
	}

	var fmReq FetchMessagesRequest
	if err := json.Unmarshal(req.Data(), &fmReq); err != nil {
		sendFetchMessagesError(ctx, "invalid request format")
		return
	}

	if fmReq.RoomID == "" || fmReq.UserID == "" {
		sendFetchMessagesError(ctx, "room_id and user_id are required")
		return
	}

	session := services.GetSession(ctx.Session())
	if session == nil || session.UserID != fmReq.UserID {
		sendFetchMessagesError(ctx, "user_id mismatch")
		return
	}

	if fmReq.Limit == 0 {
		fmReq.Limit = 50
	}

	msgs, hasMore, err := services.ListMessages(fmReq.RoomID, fmReq.BeforeID, fmReq.Limit, fmReq.IncludeSystem)
	if err != nil {
		log.Printf("failed to fetch messages: %v", err)
		sendFetchMessagesError(ctx, "failed to fetch messages")
		return
	}

	fetched := make([]FetchedMessage, 0, len(msgs))
	for _, m := range msgs {
		fetched = append(fetched, FetchedMessage{
			ID:         m.ID,
			RoomID:     m.RoomID,
			SenderID:   m.SenderID,
			SenderName: m.SenderName,
			Body:       m.Body,
			CreatedAt:  m.SentAt.Format(time.RFC3339),
		})
	}

	resp := FetchMessagesResponse{
		Success:  true,
		Message:  "messages fetched",
		Messages: fetched,
		HasMore:  hasMore,
	}

	if len(msgs) > 0 {
		last := msgs[len(msgs)-1]
		resp.NextBeforeID = fmt.Sprintf("%d", last.ID)
		resp.NextBeforeCreatedAt = last.SentAt.Format(time.RFC3339)
	}

	data, _ := json.Marshal(resp)
	ctx.SetResponseMessage(easytcp.NewMessage(req.ID(), data))
}

func sendFetchMessagesError(ctx easytcp.Context, msg string) {
	resp := FetchMessagesResponse{Success: false, Message: msg}
	data, _ := json.Marshal(resp)
	ctx.SetResponseMessage(easytcp.NewMessage(ctx.Request().ID(), data))
}

func sendMessageError(ctx easytcp.Context, msg string) {
	resp := SendMessageResponse{Success: false, Message: msg}
	data, _ := json.Marshal(resp)
	ctx.SetResponseMessage(easytcp.NewMessage(ctx.Request().ID(), data))
}
