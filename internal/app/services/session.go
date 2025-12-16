package services

import (
	"sync"

	"github.com/DarthPestilane/easytcp"
)

// UserSession holds authenticated user data for a connection.
type UserSession struct {
	UserID        string
	Email         string
	UserName      string
	Authenticated bool
}

var (
	sessions   = make(map[interface{}]*UserSession)
	sessionsMu sync.RWMutex
)

// StoreSession saves user info for the connection's lifetime.
func StoreSession(sess easytcp.Session, userID, email, userName string) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	sessions[sess.ID()] = &UserSession{
		UserID:        userID,
		Email:         email,
		UserName:      userName,
		Authenticated: true,
	}
}

// GetSession retrieves the user session, returns nil if not found.
func GetSession(sess easytcp.Session) *UserSession {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	return sessions[sess.ID()]
}

// RemoveSession cleans up session data when connection closes.
func RemoveSession(sess easytcp.Session) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	delete(sessions, sess.ID())
}

// IsAuthenticated checks if the session is authenticated.
func IsAuthenticated(sess easytcp.Session) bool {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	userSession, exists := sessions[sess.ID()]
	return exists && userSession.Authenticated
}
