package rest

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/golang-jwt/jwt"
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
		return "", time.Time{}, err
	}

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
		Token: token,
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

func (api *API) generateLink() {

}
