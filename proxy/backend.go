package proxy

import (
	"errors"
	"net"
	"sync"

	"eGate/config"

	server "github.com/emersion/go-smtp"

	"golang.org/x/crypto/bcrypt"
)

type Backend struct {
	Cfg       *config.Config
	Whitelist *config.Whitelist
	Mu        sync.RWMutex
}

func (b *Backend) UpdateWhitelist(newWl *config.Whitelist) {
	b.Mu.Lock()
	defer b.Mu.Unlock()
	b.Whitelist = newWl
}

func (b *Backend) NewSession(c *server.Conn) (server.Session, error) {
	host, _, _ := net.SplitHostPort(c.Conn().RemoteAddr().String())

	b.Mu.RLock()
	_, ok := b.Whitelist.Entries[host]
	b.Mu.RUnlock()

	if !ok {
		return nil, errors.New("access denied: IP not whitelisted")
	}
	return &Session{cfg: b.Cfg, backend: b, remoteAddr: host}, nil
}

func (b *Backend) Login(state *server.Conn, username, password string) (server.Session, error) {
	host, _, _ := net.SplitHostPort(state.Hostname())

	b.Mu.RLock()
	entry, ok := b.Whitelist.Entries[host]
	b.Mu.RUnlock()

	if !ok {
		return nil, errors.New("IP not whitelisted")
	}

	if entry.PasswordHash != "" {
		err := bcrypt.CompareHashAndPassword([]byte(entry.PasswordHash), []byte(password))
		if err != nil {
			return nil, errors.New("invalid credentials")
		}
	}

	return &Session{cfg: b.Cfg, backend: b, remoteAddr: host, authDone: true}, nil
}
