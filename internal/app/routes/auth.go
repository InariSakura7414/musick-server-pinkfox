package routes

import (
	"encoding/json"
	"log"

	"musick-server/internal/app/services"

	"github.com/DarthPestilane/easytcp"
)

type LoginRequest struct {
	Token string `json:"token"` // JWT from Supabase
}

type LoginResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	UserID   string `json:"user_id,omitempty"`
	UserName string `json:"user_name,omitempty"`
}

func RegisterAuthRoutes(s *easytcp.Server) {
	s.AddRoute(10, handleLogin)
}

func handleLogin(ctx easytcp.Context) {
	req := ctx.Request()

	var loginReq LoginRequest
	if err := json.Unmarshal(req.Data(), &loginReq); err != nil {
		sendError(ctx, "invalid request format")
		return
	}

	// Verify token with Supabase
	user, err := services.VerifyToken(loginReq.Token)
	if err != nil {
		log.Printf("token verification failed: %v", err)
		sendError(ctx, "authentication failed")
		return
	}

	log.Printf("user authenticated: %s (%s)", user.Email, user.ID)

	// Store session data for the connection's lifetime
	services.StoreSession(ctx.Session(), user.ID, user.Email)

	resp := LoginResponse{
		Success:  true,
		Message:  "authenticated",
		UserID:   user.ID,
		UserName: user.GetUserName(),
	}

	respData, _ := json.Marshal(resp)
	ctx.SetResponseMessage(easytcp.NewMessage(req.ID(), respData))
}

func sendError(ctx easytcp.Context, msg string) {
	resp := LoginResponse{Success: false, Message: msg}
	data, _ := json.Marshal(resp)
	ctx.SetResponseMessage(easytcp.NewMessage(ctx.Request().ID(), data))
}
