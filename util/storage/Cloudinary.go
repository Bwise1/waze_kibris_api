package storage

import (
	"context"
	"log"

	"github.com/bwise1/waze_kibris/config"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type Cloudinary struct {
	CLD *cloudinary.Cloudinary
}

func NewCloudinary(cfg *config.Config) *Cloudinary {
	cld, err := cloudinary.NewFromParams(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret)
	if err != nil {
		log.Fatalf("Failed to initialize Cloudinary: %v", err)
	}

	return &Cloudinary{CLD: cld}
}

func (c *Cloudinary) UploadImage(ctx context.Context, filePath string, folder string) (string, error) {
	resp, err := c.CLD.Upload.Upload(ctx, filePath, uploader.UploadParams{Folder: folder})
	if err != nil {
		return "", err
	}
	return resp.SecureURL, nil
}
