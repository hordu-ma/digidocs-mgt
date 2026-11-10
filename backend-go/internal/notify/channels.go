// Package notify delivers notifications to channels outside the app (group
// robot webhooks, email). In-app notifications are stored by the repository;
// this package only handles the optional external copies.
package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

// Recipient is one addressee of an external message.
type Recipient struct {
	DisplayName string
	Email       string
}

// Message is a channel-neutral notification.
type Message struct {
	Title      string
	Body       string
	Link       string // absolute URL when PUBLIC_BASE_URL is set, else empty
	Recipients []Recipient
}

// Channel is one external delivery target.
type Channel interface {
	Name() string
	Deliver(ctx context.Context, msg Message) error
}

// WebhookChannel posts to a group robot. Kind selects the payload shape:
// "wecom" (企业微信群机器人), "dingtalk" (钉钉自定义机器人) or "generic" JSON.
type WebhookChannel struct {
	URL    string
	Kind   string
	Client *http.Client
}

func (c WebhookChannel) Name() string { return "webhook:" + c.Kind }

func (c WebhookChannel) Deliver(ctx context.Context, msg Message) error {
	names := make([]string, 0, len(msg.Recipients))
	for _, r := range msg.Recipients {
		names = append(names, r.DisplayName)
	}
	text := "### " + msg.Title + "\n" + msg.Body
	if len(names) > 0 {
		text += "\n\n> 提醒：" + strings.Join(names, "、")
	}
	if msg.Link != "" {
		text += "\n\n[在 DigiDocs 中查看](" + msg.Link + ")"
	}

	var payload any
	switch c.Kind {
	case "wecom":
		payload = map[string]any{"msgtype": "markdown", "markdown": map[string]any{"content": text}}
	case "dingtalk":
		payload = map[string]any{"msgtype": "markdown", "markdown": map[string]any{"title": msg.Title, "text": text}}
	default:
		payload = map[string]any{"title": msg.Title, "body": msg.Body, "link": msg.Link, "recipients": names}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		// The URL embeds the robot key; never echo it in an error.
		return fmt.Errorf("webhook request failed: %s", redactURLError(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

// redactURLError drops the request URL (which carries the robot key) and
// keeps only the operation and the underlying network error.
func redactURLError(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Op + ": " + ue.Err.Error()
	}
	return "request error"
}

// SMTPChannel sends one email per recipient that has an address. Port 465
// uses implicit TLS; other ports use STARTTLS when the server offers it.
type SMTPChannel struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (c SMTPChannel) Name() string { return "smtp" }

func (c SMTPChannel) Deliver(ctx context.Context, msg Message) error {
	var firstErr error
	for _, r := range msg.Recipients {
		if strings.TrimSpace(r.Email) == "" {
			continue
		}
		if err := c.send(ctx, r, msg); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (c SMTPChannel) send(ctx context.Context, to Recipient, msg Message) error {
	body := msg.Body
	if msg.Link != "" {
		body += "\r\n\r\n查看详情：" + msg.Link
	}
	data := BuildMail(c.From, to.Email, "[DigiDocs] "+msg.Title, body)
	addr := net.JoinHostPort(c.Host, fmt.Sprint(c.Port))
	var auth smtp.Auth
	if c.Username != "" {
		auth = smtp.PlainAuth("", c.Username, c.Password, c.Host)
	}
	if c.Port != 465 {
		return smtp.SendMail(addr, auth, c.From, []string{to.Email}, data)
	}
	dialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: 10 * time.Second}, Config: &tls.Config{ServerName: c.Host}}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(c.From); err != nil {
		return err
	}
	if err := client.Rcpt(to.Email); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// BuildMail renders a minimal UTF-8 plain-text message with an RFC 2047
// encoded subject. Header values are stripped of CR/LF to block injection.
func BuildMail(from, to, subject, body string) []byte {
	clean := func(s string) string { return strings.NewReplacer("\r", "", "\n", "").Replace(s) }
	var b strings.Builder
	b.WriteString("From: " + clean(from) + "\r\n")
	b.WriteString("To: " + clean(to) + "\r\n")
	b.WriteString("Subject: =?UTF-8?B?" + base64Std(clean(subject)) + "?=\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	enc := base64Std(body)
	for len(enc) > 76 {
		b.WriteString(enc[:76] + "\r\n")
		enc = enc[76:]
	}
	b.WriteString(enc + "\r\n")
	return []byte(b.String())
}
