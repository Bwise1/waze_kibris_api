package rest

import (
	"context"
	"log"

	"firebase.google.com/go/v4/messaging"
)

// SendFCMToUser sends a data+notification message to all registered devices for a user.
// No-op if Firebase Messaging is not configured or user has no tokens.
func (api *API) SendFCMToUser(ctx context.Context, userID, title, body string, data map[string]string) error {
	if api.FirebaseMessaging == nil {
		return nil
	}
	tokens, err := api.GetFCMTokensForUser(ctx, userID)
	if err != nil || len(tokens) == 0 {
		return err
	}
	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}
	br, err := api.FirebaseMessaging.SendEachForMulticast(ctx, msg)
	if err != nil {
		return err
	}
	if br.FailureCount > 0 {
		log.Printf("FCM: sent %d failed %d for user %s", br.SuccessCount, br.FailureCount, userID)
	}
	return nil
}
