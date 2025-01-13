// internal/http/rest/reports_helper.go
package rest

import (
	"context"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/bwise1/waze_kibris/util/values"
)

func (api *API) CreateReportHelper(ctx context.Context, report model.Report) (string, string, string, error) {
	reportID, err := api.CreateReportRepo(ctx, report)
	if err != nil {
		return "", values.Error, "Failed to create report", err
	}
	return reportID, values.Created, "Report created successfully", nil
}

func (api *API) GetNearbyReportsHelper(ctx context.Context, longitude, latitude, radius float64) ([]model.Report, string, string, error) {
	reports, err := api.GetNearbyReportsRepo(ctx, longitude, latitude, radius)
	if err != nil {
		return nil, values.Error, "Failed to fetch nearby reports", err
	}
	return reports, values.Success, "Nearby reports fetched successfully", nil
}

func (api *API) GetAllReportsHelper(ctx context.Context) ([]model.Report, string, string, error) {
	reports, err := api.GetAllReportsRepo(ctx)
	if err != nil {
		return nil, values.Error, "Failed to fetch all reports", err
	}
	return reports, values.Success, "All reports fetched successfully", nil
}
