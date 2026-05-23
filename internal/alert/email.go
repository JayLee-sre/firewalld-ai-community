package alert

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
)

// EmailAlerter sends alerts via SMTP email.
type EmailAlerter struct {
	host     string
	port     int
	username string
	password string
	from     string
	to       []string
	throttle *ThrottleMap
}

// NewEmailAlerter creates a new email alerter.
func NewEmailAlerter(host string, port int, username, password, from string, to []string, throttleMin int) *EmailAlerter {
	return &EmailAlerter{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		to:       to,
		throttle: NewThrottleMap(throttleMin),
	}
}

func (e *EmailAlerter) Name() string { return "email" }

func (e *EmailAlerter) Send(alert Alert) error {
	if !e.throttle.ShouldSend(alert.RuleID + ":" + alert.SourceIP) {
		return nil
	}

	subject := fmt.Sprintf("[ZhiYu-WAF] %s - %s", alert.Severity, alert.Title)
	body := fmt.Sprintf("Severity: %s\nSource IP: %s\nRule: %s\nTime: %s\n\n%s",
		alert.Severity, alert.SourceIP, alert.RuleID, alert.Timestamp.Format("2006-01-02 15:04:05"), alert.Message)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		e.from, strings.Join(e.to, ","), subject, body)

	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	auth := smtp.PlainAuth("", e.username, e.password, e.host)

	// Try TLS first, fall back to plain
	tlsConfig := &tls.Config{ServerName: e.host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		// Fall back to STARTTLS or plain
		if err := smtp.SendMail(addr, auth, e.from, e.to, []byte(msg)); err != nil {
			log.Printf("alert email send failed: %v", err)
			return err
		}
		return nil
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, e.host)
	if err != nil {
		log.Printf("alert email client failed: %v", err)
		return err
	}
	defer client.Quit()

	if err = client.Auth(auth); err != nil {
		return err
	}
	if err = client.Mail(e.from); err != nil {
		return err
	}
	for _, to := range e.to {
		if err = client.Rcpt(to); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}
	return w.Close()
}

// helper for non-TLS STARTTLS
func dialSTARTTLS(addr string, host string) (*smtp.Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return nil, err
	}
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return nil, err
		}
	}
	return client, nil
}
