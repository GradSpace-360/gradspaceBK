package services

import (
	"fmt"
	"gradspaceBK/config"

	"github.com/resend/resend-go/v2"
)

type EmailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text,omitempty"`
	HTML    string `json:"html,omitempty"`
}


func SendEmail(to, subject, text, html string) error {
    apiKey := config.GetResendAPIKey()
    if apiKey == "" {
        return fmt.Errorf("resend API key is not set in the configuration")
    }

    client := resend.NewClient(apiKey)

    params := &resend.SendEmailRequest{
        From:    "gradSpace <admin@gradspace.me>",
        To:      []string{to},
        Subject: subject,
        Text:    text,
        Html:    html,
    }

    _, err := client.Emails.Send(params)
    if err != nil {
        return fmt.Errorf("failed to send email: %w", err)
    }

    return nil
}