// Package mailer sends the down/recovery notification emails over SMTP.
package mailer

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
	"text/template"
	"time"
)

// Config holds SMTP connection details, read from the environment so the
// same binary works against any provider without code changes.
type Config struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

// ConfigFromEnv reads SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, and
// SMTP_FROM. Returns an error naming whichever required variable is unset.
func ConfigFromEnv() (Config, error) {
	c := Config{
		Host: os.Getenv("SMTP_HOST"),
		Port: os.Getenv("SMTP_PORT"),
		User: os.Getenv("SMTP_USER"),
		Pass: os.Getenv("SMTP_PASS"),
		From: os.Getenv("SMTP_FROM"),
	}
	if c.Host == "" || c.Port == "" || c.From == "" {
		return Config{}, fmt.Errorf("SMTP_HOST, SMTP_PORT, and SMTP_FROM are required")
	}
	return c, nil
}

type Mailer struct {
	cfg Config
}

func New(cfg Config) *Mailer {
	return &Mailer{cfg: cfg}
}

// Send delivers a plain-text email. Port 465 uses implicit TLS; any other
// port (587, 25) uses STARTTLS when the server offers it.
func (m *Mailer) Send(to, subject, body string) error {
	addr := net.JoinHostPort(m.cfg.Host, m.cfg.Port)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n", m.cfg.From, to, subject, body)

	var auth smtp.Auth
	if m.cfg.User != "" {
		auth = smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, m.cfg.Host)
	}

	if m.cfg.Port == "465" {
		return m.sendImplicitTLS(addr, auth, to, msg)
	}
	return smtp.SendMail(addr, auth, m.cfg.From, []string{to}, []byte(msg))
}

func (m *Mailer) sendImplicitTLS(addr string, auth smtp.Auth, to, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.cfg.Host})
	if err != nil {
		return fmt.Errorf("dialing %s: %w", addr, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp handshake: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("closing message: %w", err)
	}
	return client.Quit()
}

var downTemplate = template.Must(template.New("down").Parse(strings.TrimSpace(`
{{.Name}} missed its heartbeat.

Last seen:  {{.LastSeen}}
Expected every {{.Interval}} (plus {{.Grace}} grace).
It has been silent for {{.Silent}}.

This will keep alerting every {{.Repeat}} until it checks in again.
`)))

var downOnceTemplate = template.Must(template.New("down_once").Parse(strings.TrimSpace(`
{{.Name}} missed its heartbeat.

Last seen:  {{.LastSeen}}
Expected every {{.Interval}} (plus {{.Grace}} grace).
It has been silent for {{.Silent}}.
`)))

var upTemplate = template.Must(template.New("up").Parse(strings.TrimSpace(`
{{.Name}} checked in again and is back up.

It was down for {{.Downtime}}.
`)))

type downData struct {
	Name     string
	LastSeen string
	Interval string
	Grace    string
	Repeat   string
	Silent   string
}

// DownEmail renders the subject/body for a missed-heartbeat alert. repeat
// is the resend interval; zero means this is a one-time alert.
func DownEmail(name string, lastSeen time.Time, interval, grace, repeat, silentFor time.Duration) (subject, body string) {
	d := downData{
		Name:     name,
		LastSeen: lastSeen.Format(time.RFC1123),
		Interval: interval.String(),
		Grace:    grace.String(),
		Silent:   silentFor.Round(time.Second).String(),
	}
	var sb strings.Builder
	if repeat > 0 {
		d.Repeat = repeat.String()
		downTemplate.Execute(&sb, d)
	} else {
		downOnceTemplate.Execute(&sb, d)
	}
	return fmt.Sprintf("[deadman] %s missed its heartbeat", name), sb.String()
}

// UpEmail renders the subject/body for a recovery notice.
func UpEmail(name string, downtime time.Duration) (subject, body string) {
	var sb strings.Builder
	upTemplate.Execute(&sb, struct{ Name, Downtime string }{name, downtime.Round(time.Second).String()})
	return fmt.Sprintf("[deadman] %s is back up", name), sb.String()
}
