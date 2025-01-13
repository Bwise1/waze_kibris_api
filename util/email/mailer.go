package smtp

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/smtp"

	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/assets"
)

type Mailer struct {
	smtpHost     string
	smtpPort     string
	smtpUser     string
	smtpPassword string
	smtpFrom     string
}

func NewMailer(host string, port int, username, password, from string) *Mailer {
	return &Mailer{
		smtpHost:     host,
		smtpPort:     fmt.Sprintf("%d", port),
		smtpUser:     username,
		smtpPassword: password,
		smtpFrom:     from,
	}
}

func (m *Mailer) Send(recipient string, data interface{}, patterns ...string) error {
	log.Println("Here", m)
	for i := range patterns {
		patterns[i] = "emails/" + patterns[i]

	}

	// Create an email message
	msg := composeEmail(recipient, m.smtpFrom, patterns, data)

	// Establish an SMTP connection and send the email
	auth := smtp.PlainAuth("", m.smtpUser, m.smtpPassword, m.smtpHost)
	err := sendEmail(m.smtpHost, m.smtpPort, auth, m.smtpFrom, recipient, msg)
	log.Println(err)
	return err
}

func composeEmail(recipient, sender string, patterns []string, data interface{}) []byte {
	// Create a new buffer to store the email message
	var buf bytes.Buffer

	// Write the "To" and "From" headers
	fmt.Fprintf(&buf, "To: %s\r\n", recipient)
	fmt.Fprintf(&buf, "From: %s\r\n", sender)

	// Load and execute templates for subject, plain text, and HTML
	subjectTemplate, plainBodyTemplate, htmlBodyTemplate := loadTemplates(patterns)

	subject := executeTemplate(subjectTemplate, data)
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)

	// Start the MIME structure for a multipart/alternative email
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=boundary-string\r\n")
	fmt.Fprintf(&buf, "\r\n")
	fmt.Fprintf(&buf, "--boundary-string\r\n")

	plainBody := executeTemplate(plainBodyTemplate, data)
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=\"utf-8\"\r\n")
	fmt.Fprintf(&buf, "Content-Transfer-Encoding: quoted-printable\r\n")
	fmt.Fprintf(&buf, "\r\n")
	fmt.Fprintf(&buf, "%s\r\n", plainBody)

	if htmlBodyTemplate != nil {
		htmlBody := executeTemplate(htmlBodyTemplate, data)
		fmt.Fprintf(&buf, "--boundary-string\r\n")
		fmt.Fprintf(&buf, "Content-Type: text/html; charset=\"utf-8\"\r\n")
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: quoted-printable\r\n")
		fmt.Fprintf(&buf, "\r\n")
		fmt.Fprintf(&buf, "%s\r\n", htmlBody)
	}

	// Close the MIME structure
	fmt.Fprintf(&buf, "--boundary-string--\r\n")

	// Convert the buffer to a byte slice and return it
	return buf.Bytes()
}

func loadTemplates(patterns []string) (*template.Template, *template.Template, *template.Template) {
	ts := template.New("")

	subjectTemplate := loadTemplate(ts, patterns, "subject")
	plainBodyTemplate := loadTemplate(ts, patterns, "plainBody")
	htmlBodyTemplate := loadTemplate(ts, patterns, "htmlBody")

	return subjectTemplate, plainBodyTemplate, htmlBodyTemplate
}

func loadTemplate(ts *template.Template, patterns []string, name string) *template.Template {
	ts, err := ts.Funcs(util.TemplateFuncs).ParseFS(assets.EmbeddedFiles, patterns...)
	if err != nil {
		// Handle the error appropriately
		fmt.Println("Error parsing template:", err)
		return nil
	}

	return ts.Lookup(name)
}

func executeTemplate(tmpl *template.Template, data interface{}) string {
	var buffer bytes.Buffer
	if tmpl != nil {
		if err := tmpl.Execute(&buffer, data); err != nil {
			// Handle the error appropriately
			fmt.Println("Error executing template:", err)
			return ""
		}
	}
	return buffer.String()
}

func sendEmail(smtpHost, smtpPort string, auth smtp.Auth, from, recipient string, msg []byte) error {
	for i := 1; i <= 3; i++ {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // Set to true to skip server certificate verification (not recommended in production)
		}
		client, err := smtp.Dial(smtpHost + ":" + smtpPort)
		if err != nil {
			return err
		}

		err = client.StartTLS(tlsConfig)
		if err != nil {
			return err
		}

		err = client.Auth(auth)
		if err != nil {
			client.Quit()
			return err
		}

		err = client.Mail(from)
		if err != nil {
			client.Quit()
			return err
		}

		err = client.Rcpt(recipient)
		if err != nil {
			client.Quit()
			return err
		}

		writer, err := client.Data()
		if err != nil {
			client.Quit()
			return err
		}

		_, err = writer.Write(msg)
		if err != nil {
			writer.Close()
			client.Quit()
			return err
		}

		err = writer.Close()
		if err != nil {
			client.Quit()
			return err
		}

		client.Quit()
		return nil
	}

	return fmt.Errorf("failed to send email after 3 attempts")
}
