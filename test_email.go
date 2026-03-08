package main

import (
	"log"

	"github.com/bwise1/waze_kibris/config"
	smtp "github.com/bwise1/waze_kibris/util/email"
)

func main() {
	// Load config from .env
	cfg := config.New()

	// Initialize mailer
	mailer := smtp.NewMailer(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPassword,
		cfg.SMTPFrom,
	)

	// Test data for the email template
	data := struct {
		Code string
	}{
		Code: "TEST123",
	}

	// Send test email
	recipient := "oguntoyebenjamin2@gmail.com"
	err := mailer.Send(recipient, data, "verifyEmail.tmpl")
	if err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}

	log.Printf("Test email successfully sent to %s", recipient)
}
