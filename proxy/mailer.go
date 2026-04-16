package proxy

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	smtp_client "net/smtp"
	"strings"

	server "github.com/emersion/go-smtp"

	"eGate/config"
)

type Session struct {
	cfg        *config.Config
	backend    *Backend
	remoteAddr string
	authDone   bool
	from       string
	to         []string
	data       []byte
}

func (s *Session) Mail(from string, opts *server.MailOptions) error {
	s.backend.Mu.RLock()
	entry := s.backend.Whitelist.Entries[s.remoteAddr]
	s.backend.Mu.RUnlock()

	if entry.PasswordHash != "" && !s.authDone {
		return errors.New("authentication required")
	}
	s.from = from
	return nil
}

// Add the 'opts' parameter to match the interface
func (s *Session) Rcpt(to string, opts *server.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

// Support newer go-smtp versions
func (s *Session) RcptWithOpts(to string, _ *server.RcptOptions) error {
	return s.Rcpt(to, nil)
}

func (s *Session) Data(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.data = data
	return s.sendToRemoteSMTP()
}

func (s *Session) sendToRemoteSMTP() error {
	remoteCfg := s.cfg.Remote
	hostPort := net.JoinHostPort(remoteCfg.Host, remoteCfg.Port)

	conn, err := tls.Dial("tcp", hostPort, &tls.Config{ServerName: remoteCfg.Host})
	if err != nil {
		return err
	}
	defer conn.Close()

	c, err := smtp_client.NewClient(conn, remoteCfg.Host)
	if err != nil {
		return err
	}
	defer c.Quit()

	auth := smtp_client.PlainAuth("", remoteCfg.User, remoteCfg.Password, remoteCfg.Host)
	if err = c.Auth(auth); err != nil {
		return err
	}

	if err = c.Mail(remoteCfg.From); err != nil {
		return err
	}
	for _, rcpt := range s.to {
		if err = c.Rcpt(rcpt); err != nil {
			return err
		}
	}

	msg := rewriteFromHeader(s.data, remoteCfg.From)
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}

func (s *Session) Reset()        { s.from = ""; s.to = nil; s.data = nil }
func (s *Session) Logout() error { return nil }

func rewriteFromHeader(raw []byte, newFrom string) []byte {
	str := string(raw)
	parts := strings.SplitN(str, "\r\n\r\n", 2)
	if len(parts) != 2 {
		return raw
	}
	headers := parts[0]
	body := parts[1]

	lines := strings.Split(headers, "\r\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "from:") {
			lines[i] = "From: " + newFrom
			found = true
			break
		}
	}
	if !found {
		lines = append([]string{"From: " + newFrom}, lines...)
	}
	return []byte(strings.Join(lines, "\r\n") + "\r\n\r\n" + body)
}
