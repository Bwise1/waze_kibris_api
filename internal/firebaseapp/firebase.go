package firebaseapp

import (
	"context"
	"fmt"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// NewApp returns a Firebase app from the service account JSON path, or nil if unset.
func NewApp(ctx context.Context, credentialsPath string) (*firebase.App, error) {
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
	return app, nil
}

// NewAuthClient returns a Firebase Auth client for verifying ID tokens.
func NewAuthClient(ctx context.Context, credentialsPath string) (*auth.Client, error) {
	app, err := NewApp(ctx, credentialsPath)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, nil
	}
	return app.Auth(ctx)
}

// InitAuthAndMessaging returns Auth + FCM clients from a single Firebase app.
// If credentials are unset, returns (nil, nil, nil). Messaging may be nil if Messaging() fails.
func InitAuthAndMessaging(ctx context.Context, credentialsPath string) (*auth.Client, *messaging.Client, error) {
	app, err := NewApp(ctx, credentialsPath)
	if err != nil {
		return nil, nil, err
	}
	if app == nil {
		return nil, nil, nil
	}
	a, err := app.Auth(ctx)
	if err != nil {
		return nil, nil, err
	}
	m, err := app.Messaging(ctx)
	if err != nil {
		return a, nil, nil
	}
	return a, m, nil
}
