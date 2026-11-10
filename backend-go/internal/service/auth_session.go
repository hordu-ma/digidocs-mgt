package service

import (
	"context"
	"log"
	"sync"
	"time"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/repository"
)

// sessionCacheTTL bounds how long a disabled account or a password change can
// take to cut off existing tokens, in exchange for not hitting the database on
// every request.
const sessionCacheTTL = 30 * time.Second

type sessionState struct {
	status            string
	passwordChangedAt int64 // unix seconds, 0 if never changed
	fetchedAt         time.Time
}

type sessionCache struct {
	mu    sync.Mutex
	items map[string]sessionState
}

func newSessionCache() *sessionCache { return &sessionCache{items: map[string]sessionState{}} }

// SessionValid reports whether a signature-valid, unexpired token may still be
// used: the account must be active and the token issued no earlier than the
// last password change. Before 2026-11-10 a changed password or a disabled
// account left existing tokens usable for up to two hours.
//
// A lookup failure is logged and allowed (fail open): the token itself is
// already verified, and this check is defense in depth; failing closed would
// log everyone out whenever the database blips.
func (s AuthService) SessionValid(ctx context.Context, claims auth.Claims) bool {
	reader, ok := s.userRepo.(repository.SessionStateReader)
	if !ok || claims.UserID == "" || s.sessions == nil {
		return true
	}
	state, ok := s.sessions.get(claims.UserID)
	if !ok {
		status, changedAt, err := reader.GetSessionState(ctx, claims.UserID)
		if err != nil {
			if err == ErrNotFound {
				return false
			}
			log.Printf("[auth] session state lookup failed user=%s: %v", claims.UserID, err)
			return true
		}
		state = sessionState{status: status, fetchedAt: time.Now()}
		if changedAt != nil {
			state.passwordChangedAt = changedAt.Unix()
		}
		s.sessions.put(claims.UserID, state)
	}
	if state.status != "active" {
		return false
	}
	return claims.IssuedAt >= state.passwordChangedAt
}

func (c *sessionCache) get(userID string) (sessionState, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	st, ok := c.items[userID]
	if !ok || time.Since(st.fetchedAt) > sessionCacheTTL {
		return sessionState{}, false
	}
	return st, true
}

func (c *sessionCache) put(userID string, st sessionState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[userID] = st
}

func (c *sessionCache) forget(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, userID)
}

// ReissueToken signs a fresh token for the user, e.g. right after a password
// change so the session that changed it keeps working.
func (s AuthService) ReissueToken(ctx context.Context, userID string) (string, error) {
	user, err := s.userRepo.GetUserCredentials(ctx, userID)
	if err != nil {
		return "", err
	}
	token, _, err := s.issueToken(user)
	return token, err
}
