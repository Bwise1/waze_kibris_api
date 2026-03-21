package firebaseapp

import (
	"context"
	"fmt"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// NewAuthClient returns a Firebase Auth client for verifying ID tokens.
// credentialsPath may be empty: callers should treat (nil, nil) as "Firebase not configured".
// If empty, GOOGLE_APPLICATION_CREDENTIALS is used when set.
func NewAuthClient(ctx context.Context, credentialsPath string) (*auth.Client, error) {
	path := credentialsPath
	if path == "" {
		path = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}
	if path == "" {
		return nil, nil
	}

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(path))
	if err != nil {
		return nil, fmt.Errorf("firebase.NewApp: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase Auth: %w", err)
	}
	return client, nil
}
