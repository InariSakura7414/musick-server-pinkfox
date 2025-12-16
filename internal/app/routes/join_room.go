package routes

import (
	"encoding/json"
	"log"
	"time"

	"musick-server/internal/app/services"

	"github.com/DarthPestilane/easytcp"
)

type JoinRoomRequest struct {
	Code   string `json:"code"`
	UserID string `json:"user_id"`
}

type JoinRoomResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	RoomID    string `json:"room_id,omitempty"`
	Code      string `json:"code,omitempty"`
	Title     string `json:"title,omitempty"`
	OwnerID   string `json:"owner_id,omitempty"`
	IsPrivate bool   `json:"is_private,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

func RegisterJoinRoomRoutes(s *easytcp.Server) {
	s.AddRoute(202, handleJoinRoom)
}

func handleJoinRoom(ctx easytcp.Context) {
	req := ctx.Request()

	if !services.IsAuthenticated(ctx.Session()) {
		sendJoinRoomError(ctx, "not authenticated")
		return
	}

	var jr JoinRoomRequest
	if err := json.Unmarshal(req.Data(), &jr); err != nil {
		sendJoinRoomError(ctx, "invalid request format")
		return
	}

	if jr.Code == "" || jr.UserID == "" {
		sendJoinRoomError(ctx, "code and user_id are required")
		return
	}

	session := services.GetSession(ctx.Session())
	if session == nil || session.UserID != jr.UserID {
		sendJoinRoomError(ctx, "user_id mismatch")
		return
	}

	room, err := services.JoinRoomByCode(jr.Code, jr.UserID)
	if err != nil {
		log.Printf("failed to join room: %v", err)
		sendJoinRoomError(ctx, "failed to join room")
		return
	}

	// Track membership for broadcasts
	services.AddSessionToRoom(room.ID, ctx.Session())

	resp := JoinRoomResponse{
		Success:   true,
		Message:   "joined room",
		RoomID:    room.ID,
		Code:      room.Code,
		Title:     room.Title,
		OwnerID:   room.OwnerID,
		IsPrivate: room.IsPrivate,
		CreatedAt: room.CreatedAt.Format(time.RFC3339),
	}

	data, _ := json.Marshal(resp)
	ctx.SetResponseMessage(easytcp.NewMessage(req.ID(), data))
}

func sendJoinRoomError(ctx easytcp.Context, msg string) {
	resp := JoinRoomResponse{Success: false, Message: msg}
	data, _ := json.Marshal(resp)
	ctx.SetResponseMessage(easytcp.NewMessage(ctx.Request().ID(), data))
}
