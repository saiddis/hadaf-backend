// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package email

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

// ISender is the low-level interface for sending an OTP email directly via SMTP.
type ISender interface {
	SendOTP(receiver string, code string) error
}

// SmtpSender is a concrete SMTP email sender backed by os environment variables.
type SmtpSender struct {
	host     string
	port     string
	email    string
	password string
}

// NewSmtpSender initialises an SmtpSender from the SMTP_* environment variables.
func NewSmtpSender() *SmtpSender {
	return &SmtpSender{
		host:     os.Getenv("SMTP_HOST"),     // e.g. smtp.gmail.com
		port:     os.Getenv("SMTP_PORT"),     // e.g. 587
		email:    os.Getenv("SMTP_EMAIL"),    // sender email address
		password: os.Getenv("SMTP_PASSWORD"), // app-specific password
	}
}

// SendOTP sends a plain OTP verification email to the given receiver address.
func (s *SmtpSender) SendOTP(receiver string, code string) error {
	if s.email == "" || s.password == "" {
		fmt.Println("[SMTP] Credentials missing, skipping email sending.")
		return nil
	}

	auth := smtp.PlainAuth("", s.email, s.password, s.host)
	addr := s.host + ":" + s.port

	// Sanitise the receiver address to remove stray whitespace.
	cleanReceiver := strings.TrimSpace(receiver)

	// Build a standards-compliant email message with CRLF line endings.
	headers := "From: " + s.email + "\r\n" +
		"To: " + cleanReceiver + "\r\n" +
		"Subject: Your Hadaf Verification Code\r\n" +
		"MIME-version: 1.0\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
		"\r\n" // The blank line separates headers from body.

	body := fmt.Sprintf(`<html>
<body>
    <h2>Hello!</h2>
    <p>Your verification code is: <b>%s</b></p>
    <p>Do not share this code with anyone.</p>
</body>
</html>`, code)

	msg := []byte(headers + body)

	if err := smtp.SendMail(addr, auth, s.email, []string{cleanReceiver}, msg); err != nil {
		return fmt.Errorf("smtp send error: %w", err)
	}

	return nil
}
