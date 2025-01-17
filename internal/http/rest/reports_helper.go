package rest

import (
	"context"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/bwise1/waze_kibris/util/values"
)

func (api *API) CreateReportHelper(ctx context.Context, report model.CreateReportRequest) (model.CreateReportResponse, string, string, error) {
	newReport, err := api.CreateReportRepo(ctx, report)
	if err != nil {
		return model.CreateReportResponse{}, values.Error, "Failed to create report", err
	}
	return newReport, values.Created, "Report created successfully", nil
}

func (api *API) GetReportByIDHelper(ctx context.Context, reportID string) (model.Report, string, string, error) {
	report, err := api.GetReportByIDRepo(ctx, reportID)
	if err != nil {
		if err == ErrReportNotFound {
			return model.Report{}, values.NotFound, "Report not found", err
		}
		return model.Report{}, values.Error, "Failed to fetch report", err
	}
	return report, values.Success, "Report fetched successfully", nil
}

func (api *API) GetNearbyReportsHelper(ctx context.Context, params model.NearbyReportsParams) ([]model.Report, string, string, error) {
	reports, err := api.GetNearbyReportsRepo(ctx, params)
	if err != nil {
		return nil, values.Error, "Failed to fetch nearby reports", err
	}
	return reports, values.Success, "Nearby reports fetched successfully", nil
}

// func (api *API) GetAllReportsHelper(ctx context.Context) ([]model.Report, string, string, error) {
// 	reports, err := api.GetAllReports()
// 	if err != nil {
// 		return nil, values.Error, "Failed to fetch all reports", err
// 	}
// 	return reports, values.Success, "All reports fetched successfully", nil
// }

func (api *API) UpdateReportHelper(ctx context.Context, report model.Report) (string, string, error) {
	err := api.UpdateReportRepo(ctx, report)
	if err != nil {
		if err == ErrUpdateFailed {
			return values.NotFound, "Report not found", err
		}
		return values.Error, "Failed to update report", err
	}
	return values.Success, "Report updated successfully", nil
}

func (api *API) DeleteReportHelper(ctx context.Context, id string, userID string) (string, string, error) {
	err := api.DeleteReportRepo(ctx, id, userID)
	if err != nil {
		if err == ErrDeleteFailed {
			return values.NotFound, "Report not found", err
		}
		return values.Error, "Failed to delete report", err
	}
	return values.Success, "Report deleted successfully", nil
}
