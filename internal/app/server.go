package app

import (
	"log"

	"musick-server/internal/app/routes"
	"musick-server/internal/app/services"

	"github.com/DarthPestilane/easytcp"
)

// Server wraps easytcp.Server and centralizes route registration.
type Server struct {
	srv *easytcp.Server
}

// New creates a configured server and registers all routes.
func New() *Server {
	srv := easytcp.NewServer(&easytcp.ServerOption{Packer: easytcp.NewDefaultPacker()})

	// Log when clients connect/disconnect.
	srv.OnSessionCreate = func(sess easytcp.Session) {
		addr := sess.Conn().RemoteAddr().String()
		log.Printf("client connected: %s", addr)
	}
	srv.OnSessionClose = func(sess easytcp.Session) {
		addr := sess.Conn().RemoteAddr().String()
		log.Printf("client disconnected: %s", addr)
		services.RemoveSession(sess)
		services.RemoveSessionFromAllRooms(sess)
	}

	registerRoutes(srv)

	return &Server{srv: srv}
}

// Run starts listening on the provided address.
func (s *Server) Run(addr string) error {
	log.Printf("listening on %s", addr)
	return s.srv.Run(addr)
}

// registerRoutes wires all message handlers.

func registerRoutes(s *easytcp.Server) {
	// Route 1: echo request body back to the sender.
	routes.RegisterEchoRoutes(s)

	// Route 10: authenticate user via Supabase JWT.
	routes.RegisterAuthRoutes(s)

	// Route 201: create room.
	// Route 210: list rooms.
	routes.RegisterRoomRoutes(s)

	// Route 202: join room by code.
	routes.RegisterJoinRoomRoutes(s)

	// Route 301: post message to Supabase.
	routes.RegisterMessageRoutes(s)
}
