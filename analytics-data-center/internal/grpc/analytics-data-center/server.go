package analyticsdatacenter

import (
	"analyticDataCenter/analytics-data-center/internal/lib/validate"
	"context"

	analyticsv1 "github.com/alexardishev/proto_auth/gen/go/analytics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AnalyticsDataCenter interface {
	StartETLProcess(ctx context.Context, idSchema int64) (taskID string, err error)
}

const (
	emptyValue = 0
)

type serverAPI struct {
	analyticsv1.UnimplementedAnalyticsServer
	analyticsDataCenter AnalyticsDataCenter
}
type CheckValidateStartETL struct {
	SchemaID int64 `validate:"required" json:"user_id,omitempty"`
}

func RegisterServerAPI(gRPC *grpc.Server, analyticsDataCenter AnalyticsDataCenter) {
	analyticsv1.RegisterAnalyticsServer(gRPC, &serverAPI{analyticsDataCenter: analyticsDataCenter})
}

func (s *serverAPI) StartETLProcess(ctx context.Context, req *analyticsv1.StartETLProcessRequest) (*analyticsv1.StartETLProcessResponse, error) {
	validate.Validate(&CheckValidateStartETL{req.GetShemaID()})

	if req.ShemaID == emptyValue || req.ShemaID < emptyValue {
		return nil, status.Error(codes.InvalidArgument, "shemaID не может быть меньше 1")
	}

	taskID, err := s.analyticsDataCenter.StartETLProcess(ctx, req.GetShemaID())
	if err != nil {
		return nil, status.Error(codes.Internal, "процесс не удалось запустить!")
	}
	return &analyticsv1.StartETLProcessResponse{
		TaskID: taskID,
	}, nil
}
