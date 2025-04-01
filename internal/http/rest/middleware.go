package rest

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bwise1/waze_kibris/util/tracing"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/golang-jwt/jwt"
	"github.com/lucsky/cuid"
)

// RequestTracing handles the request tracing context
func RequestTracing(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		requestSource := r.Header.Get(values.HeaderRequestSource)
		if requestSource == "" {
			errM := errors.New("X-Request-Source is empty")

			writeErrorResponse(w, errM, values.Error, errM.Error())
			return
		}

		requestID := r.Header.Get(values.HeaderRequestID)
		if requestID == "" {
			requestID = cuid.New()
		}

		tracingContext := tracing.Context{
			RequestID:     requestID,
			RequestSource: requestSource,
		}

		ctx = context.WithValue(ctx, values.ContextTracingKey, tracingContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(fn)
}

// requireLogin
func (api *API) RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := strings.Split(r.Header.Get("Authorization"), " ")
		if len(authorization) != 2 || authorization[0] != "Bearer" {
			writeErrorResponse(w, errors.New(values.NotAuthorised), values.NotAuthorised, "not-authorized")
			return
		}

		claims, err := api.verifyToken(authorization[1], false)
		if err != nil {
			if err.Error() == "token expired" {
				// Handle the expired token case
				writeErrorResponse(w, err, values.TokenExpired, "token-expired")
				return
			}
			writeErrorResponse(w, err, values.NotAuthorised, "invalid-token")
			return
		}

		dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Get additional user info from database if needed
		user, err := api.GetUserByID(dbCtx, claims.UserID)
		if err != nil {
			writeErrorResponse(w, err, values.NotAuthorised, "user-not-found")
			return
		}

		// Add minimal information to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user_id", user.ID.String())
		// ctx = context.WithValue(ctx, "user", user) // Add full user object if needed
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (api *API) verifyToken(tokenString string, isRefresh bool) (*TokenClaims, error) {
	// Determine the correct secret key based on token type
	secret := api.Config.JwtSecret
	if isRefresh {
		secret = api.Config.RefreshSecret
	}

	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is correct
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Println("unexpected signing method")
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})

	// Specifically handle token expiration
	if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&jwt.ValidationErrorExpired != 0 {
			log.Println("token expired")
			return nil, fmt.Errorf("token expired")
		}
	}

	// Check for errors or invalid token
	if err != nil || !token.Valid {
		log.Println("error verifying token", err)
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Println("error extracting claims")
		return nil, fmt.Errorf("invalid claims")
	}

	// Check the token type (use "typ" instead of "type")
	tokenType, _ := claims["typ"].(string)
	log.Println("token type", tokenType, "expected", isRefresh)
	if (isRefresh && tokenType != "refresh") || (!isRefresh && tokenType != "access") {
		log.Println("invalid token type")
		return nil, fmt.Errorf("invalid token type")
	}

	// Extract user ID
	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user id")
	}

	// Log extracted user ID and token type
	log.Println("user id", userID)
	log.Println("token type", tokenType)

	// Return the extracted claims
	return &TokenClaims{
		UserID: userID,
		Type:   tokenType,
		Exp:    int64(claims["exp"].(float64)),
	}, nil
}
