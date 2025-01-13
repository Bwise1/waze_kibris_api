package rest

import (
	"context"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportRepo struct {
	DB *pgxpool.Pool
}

func (api *API) CreateReportRepo(ctx context.Context, report model.Report) (string, error) {
	var reportID string
	stmt := `
        INSERT INTO reports (
            user_id, type, subtype, position, description, severity, expires_at, image_url, report_source, report_status
        ) VALUES (
            $1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326), $6, $7, $8, $9, $10, $11
        ) RETURNING id;
    `
	err := api.Deps.DB.Pool().QueryRow(ctx, stmt, report.UserID, report.Type, report.Subtype, report.Longitude, report.Latitude, report.Description, report.Severity, report.ExpiresAt, report.ImageURL, report.ReportSource, report.ReportStatus).Scan(&reportID)
	if err != nil {
		return "", err
	}
	return reportID, nil
}

func (api *API) GetNearbyReportsRepo(ctx context.Context, longitude, latitude float64, radius float64) ([]model.Report, error) {
	stmt := `
        SELECT
            id, user_id, type, subtype, ST_X(position) AS longitude, ST_Y(position) AS latitude, description, severity, verified_count, active, resolved, created_at, updated_at, expires_at, image_url, report_source, report_status, comments_count, upvotes_count, downvotes_count
        FROM
            reports
        WHERE
            ST_DWithin(position, ST_SetSRID(ST_MakePoint($1, $2), 4326), $3)
            AND active = true
            AND expires_at > NOW();
    `
	rows, err := api.DB.Query(ctx, stmt, longitude, latitude, radius)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.Report
	for rows.Next() {
		var report model.Report
		err := rows.Scan(&report.ID, &report.UserID, &report.Type, &report.Subtype, &report.Longitude, &report.Latitude, &report.Description, &report.Severity, &report.VerifiedCount, &report.Active, &report.Resolved, &report.CreatedAt, &report.UpdatedAt, &report.ExpiresAt, &report.ImageURL, &report.ReportSource, &report.ReportStatus, &report.CommentsCount, &report.UpvotesCount, &report.DownvotesCount)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (api *API) GetAllReportsRepo(ctx context.Context) ([]model.Report, error) {
	stmt := `
        SELECT
            id, user_id, type, subtype, ST_X(position) AS longitude, ST_Y(position) AS latitude, description, severity, verified_count, active, resolved, created_at, updated_at, expires_at, image_url, report_source, report_status, comments_count, upvotes_count, downvotes_count
        FROM
            reports;
    `
	rows, err := api.DB.Query(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.Report
	for rows.Next() {
		var report model.Report
		err := rows.Scan(&report.ID, &report.UserID, &report.Type, &report.Subtype, &report.Longitude, &report.Latitude, &report.Description, &report.Severity, &report.VerifiedCount, &report.Active, &report.Resolved, &report.CreatedAt, &report.UpdatedAt, &report.ExpiresAt, &report.ImageURL, &report.ReportSource, &report.ReportStatus, &report.CommentsCount, &report.UpvotesCount, &report.DownvotesCount)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}
