package rest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/golang-jwt/jwt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/api/idtoken"
)

// func GenerateTokenPair(userID uuid.UUID) (*TokenPair, error)
// func ValidateToken(token string) (*Claims, error)
// func HashPassword(password string) (string, error)
// func VerifyPassword(hashedPwd, plainPwd string) error
// func GenerateVerificationToken() string

type TokenClaims struct {
	UserID string `json:"sub"`
	Type   string `json:"typ"`
	Exp    int64  `json:"exp"`
}

// Simplified token creation
func (api *API) createToken(id string) (string, time.Time, error) {
	log.Println("Creating token for user", id)
	exp_time, err := time.ParseDuration(api.Config.JwtExpires)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(exp_time)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": id, // subject (user ID)
		"exp": expiresAt.Unix(),
		"iat": time.Now().Unix(),
		"typ": "access",
	})

	tokenString, err := token.SignedString([]byte(api.Config.JwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}
	return tokenString, expiresAt, nil
}

func (api *API) createRefreshToken(id string) (string, time.Time, error) {
	exp_time, err := time.ParseDuration(api.Config.RefreshExpiry)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(exp_time)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": id, // subject (user ID)
		"exp": expiresAt.Unix(),
		"iat": time.Now().Unix(),
		"typ": "refresh",
	})

	tokenString, err := token.SignedString([]byte(api.Config.RefreshSecret))
	if err != nil {
		log.Println("error signing refresh token", err)
		return "", time.Time{}, err
	}
	log.Println("refresh token", tokenString)
	return tokenString, expiresAt, nil
}

func (api *API) CreateNewUser(req model.RegisterRequest) (model.VerifyCodeResponse, string, string, error) {
	var err error
	var ctx = context.TODO()

	req.Email = strings.Trim(req.Email, " ")

	err = util.ValidEmail(req.Email)
	if err != nil {
		return model.VerifyCodeResponse{}, values.NotAllowed, "Invalid email address provided", err
	}

	exists, err := api.EmailExists(ctx, req.Email)
	if err != nil {
		return model.VerifyCodeResponse{}, values.Error, "Error checking email", err
	}

	if exists {
		return model.VerifyCodeResponse{}, values.Conflict, "Email already exists", nil
	}

	user := model.User{
		ID:           util.GenerateUUID(),
		Email:        req.Email,
		AuthProvider: "email",
	}

	err = api.CreateNewUserRepo(ctx, user)
	if err != nil {
		return model.VerifyCodeResponse{}, values.Error, "Error creating new user", err
	}

	// Generate verification code
	code := util.GenerateVerificationCode()
	// Store verification code
	expiresAt := time.Now().Add(1 * time.Hour) // Code expires in 1 hour
	tokenType := "register"
	err = api.StoreVerificationCode(ctx, user.ID.String(), user.Email, code, tokenType, expiresAt)
	if err != nil {
		return model.VerifyCodeResponse{}, values.Error, "Failed to store verification code", err
	}

	log.Println("Verification code:", code)
	go func() {
		// Send verification email
		emailData := map[string]interface{}{
			"Code": code,
		}

		err = api.Mailer.Send(user.Email, emailData, "verifyEmail.tmpl")
		if err != nil {
			log.Println(values.Error, "Failed to send verification email", err)
		}
	}()

	LoginResponse := model.VerifyCodeResponse{
		ID:    user.ID.String(),
		Email: user.Email,
	}

	return LoginResponse, values.Created, "User created successfully", nil
}

func (api *API) LoginUser(req model.LoginRequest) (model.VerifyCodeResponse, string, string, error) {
	var err error
	var ctx = context.TODO()

	req.Email = strings.Trim(req.Email, " ")

	err = util.ValidEmail(req.Email)
	if err != nil {
		return model.VerifyCodeResponse{}, values.NotAllowed, "Invalid email address provided", err
	}

	user, err := api.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return model.VerifyCodeResponse{}, values.NotFound, "User not found", err
	}

	// Generate verification code
	code := util.GenerateVerificationCode()
	// Store verification code
	log.Println("Verification code:", code)
	expiresAt := time.Now().Add(1 * time.Hour) // Code expires in 1 hour
	tokenType := "login"
	err = api.StoreVerificationCode(ctx, user.ID.String(), user.Email, code, tokenType, expiresAt)
	if err != nil {
		return model.VerifyCodeResponse{}, values.Error, "Failed to store verification code", err
	}
	go func() {
		// Send verification email
		emailData := map[string]interface{}{
			"Code": code,
		}
		err = api.Mailer.Send(user.Email, emailData, "verifyEmail.tmpl")
		if err != nil {
			log.Println(values.Error, "Failed to send verification email", err)
		}
	}()

	LoginResponse := model.VerifyCodeResponse{
		ID:    user.ID.String(),
		Email: user.Email,
	}

	return LoginResponse, values.Success, "Verification code sent", nil
}

func (api *API) VerifyCodeHelper(req model.VerifyCodeRequest) (model.LoginResponse, string, string, error) {
	var err error
	var ctx = context.TODO()

	// Input validation
	if err := util.ValidEmail(req.Email); err != nil {
		return model.LoginResponse{}, values.BadRequestBody, "Invalid email format", err
	}

	if len(req.Code) != 4 {
		return model.LoginResponse{}, values.BadRequestBody, "Invalid verification code format", fmt.Errorf("code must be 4 digits")
	}

	//  if !isValidVerificationType(req.Type) {
	//     return model.User{}, values.BadRequest, "Invalid verification type", fmt.Errorf("unknown verification type: %s", req.Type)
	// }

	// Check if the code is valid
	userID, err := api.VerifyCodeRepo(ctx, req.Code, req.Type, req.Email)
	if err != nil {
		log.Println("Error verifying code", err)
		return model.LoginResponse{}, values.NotAuthorised, "Invalid or expired verification code", err
	}

	if req.Type == "register" {
		// Update the user's email verification status
		err = api.UpdateEmailVerifiedStatus(ctx, userID)
		if err != nil {
			return model.LoginResponse{}, values.Error, "Failed to update email verification status", err
		}
	} else if req.Type == "login" {
		// Handle login verification logic if needed
	}

	// Retrieve the updated user
	user, err := api.GetUserByID(ctx, userID)
	if err != nil {
		return model.LoginResponse{}, values.Error, "Failed to retrieve user", err
	}

	token, _, err := api.createToken(userID)
	if err != nil {
		return model.LoginResponse{}, values.Error, fmt.Sprintf("%s [CrTk]", values.SystemErr), err
	}
	//TODO: after verification invalidate the verification code

	refreshToken, expiresAt, err := api.createRefreshToken(userID)
	if err != nil {
		return model.LoginResponse{}, values.Error, fmt.Sprintf("%s [CrRfTk]", values.SystemErr), err
	}
	// log.Println("Refresh token", refreshToken, "expires at", expiresAt)
	// Store the refresh token in the database
	err = api.StoreRefreshToken(ctx, user.ID.String(), refreshToken, expiresAt)
	if err != nil {
		return model.LoginResponse{}, values.Error, "Failed to store refresh token", err
	}

	loggedInUser := model.LoginResponse{
		User: &model.LoginUserResponse{
			ID:                user.ID,
			FirstName:         user.FirstName,
			LastName:          user.LastName,
			Username:          user.Username,
			Email:             user.Email,
			IsVerified:        user.IsVerified,
			PreferredLanguage: user.PreferredLanguage,
		},
		Token:        token,
		RefreshToken: refreshToken, // refreshToken,
	}
	return loggedInUser, values.Success, "Verification successful", nil
}

func (api *API) ResendVerificationCode(req model.ResendCodeRequest) (string, string, error) {
	var err error
	var ctx = context.TODO()

	req.Email = strings.Trim(req.Email, " ")

	err = util.ValidEmail(req.Email)
	if err != nil {
		return values.NotAllowed, "Invalid email address provided", err
	}

	user, err := api.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return values.NotFound, "User not found", err
	}

	// Generate verification code
	code := util.GenerateVerificationCode()
	// Store verification code
	log.Println("Verification code:", code)
	expiresAt := time.Now().Add(1 * time.Hour) // Code expires in 1 hour
	tokenType := "register"
	err = api.StoreVerificationCode(ctx, user.ID.String(), user.Email, code, tokenType, expiresAt)
	if err != nil {
		return values.Error, "Failed to store verification code", err
	}
	go func() {
		// Send verification email
		emailData := map[string]interface{}{
			"Name": user.FirstName,
			"Code": code,
		}
		err = api.Mailer.Send(user.Email, emailData, "verifyEmail.tmpl")
		if err != nil {
			log.Println(values.Error, "Failed to send verification email", err)
		}
	}()

	return values.Success, "Verification code sent", nil
}

// func (api *API) LogUserOut(userID int) (bool, error) {
// 	err := api.invalidateRefreshToken(context.TODO(), userID)
// 	if err != nil {
// 		return false, err
// 	}
// 	return true, nil
// }

// }

func (api *API) verifyGoogleIDToken(idToken string) (*model.NewUserInfo, error) {
	tokenValidator, err := idtoken.NewValidator(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %v", err)
	}

	payload, err := tokenValidator.Validate(context.Background(), idToken, api.Config.GoogleClientID)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	userInfo := &model.NewUserInfo{
		ID:        payload.Subject,
		Email:     payload.Claims["email"].(string),
		FirstName: payload.Claims["given_name"].(string),
		LastName:  payload.Claims["family_name"].(string),
	}

	return userInfo, nil
}

// Helper function to generate and store tokens to reduce duplication
func (api *API) generateAndStoreTokens(user model.User) (model.LoginResponse, string, string, error) {
	// Generate and store tokens

	ctx := context.TODO()
	token, _, err := api.createToken(user.ID.String())
	if err != nil {
		return model.LoginResponse{}, values.Error, "Failed to create access token", err
	}

	refreshToken, expiresAt, err := api.createRefreshToken(user.ID.String())
	if err != nil {
		return model.LoginResponse{}, values.Error, "Failed to create refresh token", err
	}

	err = api.StoreRefreshToken(ctx, user.ID.String(), refreshToken, expiresAt)
	if err != nil {
		return model.LoginResponse{}, values.Error, "Failed to store refresh token", err
	}

	user, err = api.GetUserByID(ctx, user.ID.String())
	if err != nil {
		return model.LoginResponse{}, values.Error, "Failed to retrieve user", err
	}

	// Prepare and return the response
	response := model.LoginResponse{
		User: &model.LoginUserResponse{
			ID:                user.ID,
			FirstName:         user.FirstName,
			LastName:          user.LastName,
			Email:             user.Email,
			IsVerified:        user.IsVerified,
			PreferredLanguage: user.PreferredLanguage,
		},
		Token:        token,
		RefreshToken: refreshToken,
	}

	return response, values.Success, "Login successful", nil
}

func (api *API) GoogleLogin(idToken string) (model.LoginResponse, string, string, error) {
	var ctx = context.TODO()

	// Step 1: Verify the Google ID token
	userInfo, err := api.verifyGoogleIDToken(idToken)
	if err != nil {
		return model.LoginResponse{}, values.NotAuthorised, "Invalid Google ID token", err
	}

	email := userInfo.Email
	googleUserID := userInfo.ID

	// Step 2: Check if the Google account is already linked to any user
	authRecord, err := api.GetUserAuthProviderByProviderID(ctx, "google", googleUserID)
	log.Println("auth get user error", err)

	// Fix: Check for pgx.ErrNoRows instead of sql.ErrNoRows
	if err == nil {
		// Google account is linked to a user; fetch the user
		user, err := api.GetUserByID(ctx, authRecord.UserID.String())
		if err != nil {
			return model.LoginResponse{}, values.Error, "Failed to retrieve user", err
		}
		if user.Email != email {
			return model.LoginResponse{}, values.Conflict, "Google account is linked to a different email", nil
		}

		// Generate tokens for the existing user
		return api.generateAndStoreTokens(user)
	} else if errors.Is(err, pgx.ErrNoRows) || err.Error() == "no rows in result set" {
		log.Println("Google account not linked; checking if user exists by email")
		// Google account not linked; check if user exists by email
		user, err := api.GetUserByEmail(ctx, email)
		if err != nil {
			// Check if the error is specifically "no rows found"
			if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) || err.Error() == "no rows in result set" {
				// No user exists; register a new user
				newUser := model.User{
					ID:           util.GenerateUUID(),
					Email:        email,
					FirstName:    &userInfo.FirstName,
					LastName:     &userInfo.LastName,
					AuthProvider: "google",
					IsVerified:   true, // Google has verified the email
				}
				newGUser, err := api.CreateGoogleUserRepo(ctx, newUser)
				if err != nil {
					return model.LoginResponse{}, values.Error, "Failed to create new user", err
				}

				// Link the Google account
				authRecord := model.UserAuthProvider{
					UserID:         newGUser.ID,
					AuthProvider:   "google",
					AuthProviderID: googleUserID,
				}
				_, err = api.InsertUserAuthProvider(ctx, authRecord)
				if err != nil {
					return model.LoginResponse{}, values.Error, "Failed to link Google account", err
				}

				return api.generateAndStoreTokens(newGUser)
			} else {
				return model.LoginResponse{}, values.Error, "Database error", err
			}
		} else {
			// User exists but not linked to Google; link the account
			authRecord := model.UserAuthProvider{
				UserID:         user.ID,
				AuthProvider:   "google",
				AuthProviderID: googleUserID,
			}
			_, err = api.InsertUserAuthProvider(ctx, authRecord)
			if err != nil {
				if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" { // unique_violation
					return model.LoginResponse{}, values.Conflict, "Google account is already linked to another user", err
				}
				return model.LoginResponse{}, values.Error, "Failed to link Google account", err
			}

			return api.generateAndStoreTokens(user)
		}
	} else {
		return model.LoginResponse{}, values.Error, "Database error checking Google linkage", err
	}
}

func (api *API) RefreshAccessToken(ctx context.Context, refreshToken string) (string, string, error) {
	// Validate the refresh token
	claims, err := api.verifyToken(refreshToken, true)
	if err != nil {
		return "", "", fmt.Errorf("invalid or expired refresh token")
	}

	// Ensure the token type is "refresh"
	if claims.Type != "refresh" {
		return "", "", fmt.Errorf("invalid token type")
	}

	// Check if the refresh token is revoked or expired in the database
	userID := claims.UserID
	err = api.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("refresh token validation failed: %w", err)
	}

	// Generate a new access token
	accessToken, _, err := api.createToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Optionally, generate a new refresh token
	newRefreshToken, expiresAt, err := api.createRefreshToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	// Store the new refresh token and revoke the old one
	err = api.StoreRefreshToken(ctx, userID, newRefreshToken, expiresAt)
	if err != nil {
		return "", "", fmt.Errorf("failed to store new refresh token: %w", err)
	}

	err = api.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	return accessToken, newRefreshToken, nil
}

func (api *API) generateLink() {

}
