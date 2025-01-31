package config

import (
	"log"

	"github.com/caarlos0/env/v11"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                int    `env:"PORT" default:"8080"`
	Dsn                 string `env:"DSN" default:"localhost:3306"`
	JwtSecret           string `env:"JWT_SECRET"`
	JwtExpires          string `env:"JWT_EXPIRES"`
	RefreshSecret       string `env:"REFRESH_SECRET"`
	RefreshExpiry       string `env:"REFRESH_EXPIRY"`
	SMTPHost            string `env:"SMTP_HOST"`
	SMTPPort            int    `env:"SMTP_PORT"`
	SMTPUser            string `env:"SMTP_USER"`
	SMTPPassword        string `env:"SMTP_PASSWORD"`
	SMTPFrom            string `env:"SMTP_FROM"`
	CloudinaryCloudName string `env:"CLOUDINARY_CLOUD_NAME"`
	CloudinaryAPIKey    string `env:"CLOUDINARY_API_KEY"`
	CloudinaryAPISecret string `env:"CLOUDINARY_API_SECRET"`
	GoogleClientID      string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret  string `env:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURL   string `env:"GOOGLE_REDIRECT_URL"`
}

func New() *Config {
	if loadErr := godotenv.Load(".env"); loadErr != nil {
		log.Println("[Env]: unable to load .env file %v", loadErr)
	}

	var cfg Config

	if parseErr := env.Parse(&cfg); parseErr != nil {
		log.Println("[Env]: failed to parse environment variables: %v", parseErr)
	}

	return &cfg
}
