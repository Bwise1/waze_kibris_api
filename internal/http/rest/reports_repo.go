package rest

import (
	"context"
	"errors"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportRepo struct {
	DB *pgxpool.Pool
}

var (
	ErrReportNotFound = errors.New("report not found")
	ErrUpdateFailed   = errors.New("failed to update report")
	ErrDeleteFailed   = errors.New("failed to delete report")
)

// Create inserts a new report
func (api *API) CreateReportRepo(ctx context.Context, report model.CreateReportRequest) (model.Report, error) {
	query := `
        INSERT INTO reports (
            user_id, type, subtype, position, description, severity,
            expires_at, image_url, report_source, report_status
        ) VALUES (
            $1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326), $6, $7,
            $8, $9, $10, $11
        ) RETURNING id, created_at, updated_at, verified_count, active,
            resolved, comments_count, upvotes_count, downvotes_count
    `
	var newReport model.Report
	err := api.DB.QueryRow(ctx, query,
		report.UserID, report.Type, report.Subtype, report.Longitude, report.Latitude,
		report.Description, report.Severity, report.ExpiresAt, report.ImageURL,
		report.ReportSource, report.ReportStatus,
	).Scan(
		&newReport.ID, &newReport.CreatedAt, &newReport.UpdatedAt, &newReport.VerifiedCount,
		&newReport.Active, &newReport.Resolved, &newReport.CommentsCount,
		&newReport.UpvotesCount, &newReport.DownvotesCount,
	)
	if err != nil {
		return model.Report{}, err
	}
	return newReport, nil
}

// GetByID retrieves a report by ID
func (api *API) GetReportByIDRepo(ctx context.Context, id string) (model.Report, error) {
	query := `
        SELECT
            id, user_id, type, subtype, ST_X(position) as longitude,
            ST_Y(position) as latitude, description, severity, verified_count,
            active, resolved, created_at, updated_at, expires_at, image_url,
            report_source, report_status, comments_count, upvotes_count, downvotes_count
        FROM reports
        WHERE id = $1
    `
	var report model.Report
	err := api.DB.QueryRow(ctx, query, id).Scan(
		&report.ID, &report.UserID, &report.Type, &report.Subtype,
		&report.Longitude, &report.Latitude, &report.Description, &report.Severity,
		&report.VerifiedCount, &report.Active, &report.Resolved, &report.CreatedAt,
		&report.UpdatedAt, &report.ExpiresAt, &report.ImageURL, &report.ReportSource,
		&report.ReportStatus, &report.CommentsCount, &report.UpvotesCount,
		&report.DownvotesCount,
	)
	if err == pgx.ErrNoRows {
		return model.Report{}, ErrReportNotFound
	}
	return report, err
}

// GetNearby retrieves reports within a specified radius
func (api *API) GetNearbyReportsRepo(ctx context.Context, lat, lon, radiusMeters float64) ([]model.Report, error) {
	query := `
        SELECT
            id, user_id, type, subtype, ST_X(position) as longitude,
            ST_Y(position) as latitude, description, severity, verified_count,
            active, resolved, created_at, updated_at, expires_at, image_url,
            report_source, report_status, comments_count, upvotes_count, downvotes_count
        FROM reports
        WHERE ST_DWithin(
            position,
            ST_SetSRID(ST_MakePoint($1, $2), 4326),
            $3
        )
        AND active = true
        AND expires_at > NOW()
    `
	rows, err := api.DB.Query(ctx, query, lon, lat, radiusMeters)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.Report
	for rows.Next() {
		var report model.Report
		err := rows.Scan(
			&report.ID, &report.UserID, &report.Type, &report.Subtype,
			&report.Longitude, &report.Latitude, &report.Description, &report.Severity,
			&report.VerifiedCount, &report.Active, &report.Resolved, &report.CreatedAt,
			&report.UpdatedAt, &report.ExpiresAt, &report.ImageURL, &report.ReportSource,
			&report.ReportStatus, &report.CommentsCount, &report.UpvotesCount,
			&report.DownvotesCount,
		)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

// Update updates an existing report
func (api *API) UpdateReportRepo(ctx context.Context, report model.Report) error {
	query := `
        UPDATE reports
        SET
            type = $1,
            subtype = $2,
            position = ST_SetSRID(ST_MakePoint($3, $4), 4326),
            description = $5,
            severity = $6,
            active = $7,
            resolved = $8,
            expires_at = $9,
            image_url = $10,
            report_status = $11,
            updated_at = NOW()
        WHERE id = $12 AND user_id = $13
        RETURNING updated_at
    `
	result, err := api.DB.Exec(ctx, query,
		report.Type, report.Subtype, report.Longitude, report.Latitude,
		report.Description, report.Severity, report.Active, report.Resolved,
		report.ExpiresAt, report.ImageURL, report.ReportStatus,
		report.ID, report.UserID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUpdateFailed
	}
	return nil
}

// Delete soft deletes a report by setting active to false
func (api *API) DeleteReportRepo(ctx context.Context, id string, userID string) error {
	query := `
        UPDATE reports
        SET active = false, updated_at = NOW()
        WHERE id = $1 AND user_id = $2
    `
	result, err := api.DB.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrDeleteFailed
	}
	return nil
}

// UpdateVotes updates the vote counts for a report
func (api *API) UpdateReportVotesRepo(ctx context.Context, id string, upvotes, downvotes int) error {
	query := `
        UPDATE reports
        SET
            upvotes_count = upvotes_count + $1,
            downvotes_count = downvotes_count + $2,
            updated_at = NOW()
        WHERE id = $3
    `
	result, err := api.DB.Exec(ctx, query, upvotes, downvotes, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUpdateFailed
	}
	return nil
}

// IncrementVerifiedCount increments the verified count for a report
func (api *API) IncrementVerifiedCountRepo(ctx context.Context, id string) error {
	query := `
        UPDATE reports
        SET
            verified_count = verified_count + 1,
            updated_at = NOW()
        WHERE id = $1
    `
	result, err := api.DB.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUpdateFailed
	}
	return nil
}

// GetUserReports retrieves all reports for a specific user
func (api *API) GetUserReportsRepo(ctx context.Context, userID string) ([]model.Report, error) {
	query := `
        SELECT
            id, user_id, type, subtype, ST_X(position) as longitude,
            ST_Y(position) as latitude, description, severity, verified_count,
            active, resolved, created_at, updated_at, expires_at, image_url,
            report_source, report_status, comments_count, upvotes_count, downvotes_count
        FROM reports
        WHERE user_id = $1
        ORDER BY created_at DESC
    `
	rows, err := api.DB.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.Report
	for rows.Next() {
		var report model.Report
		err := rows.Scan(
			&report.ID, &report.UserID, &report.Type, &report.Subtype,
			&report.Longitude, &report.Latitude, &report.Description, &report.Severity,
			&report.VerifiedCount, &report.Active, &report.Resolved, &report.CreatedAt,
			&report.UpdatedAt, &report.ExpiresAt, &report.ImageURL, &report.ReportSource,
			&report.ReportStatus, &report.CommentsCount, &report.UpvotesCount,
			&report.DownvotesCount,
		)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}
