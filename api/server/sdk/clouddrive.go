package sdk

import (
	"context"

	"github.com/libopenstorage/openstorage/api"
	"github.com/libopenstorage/openstorage/api/errors"
	"github.com/libopenstorage/openstorage/cluster"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CloudDriveServer struct {
	server serverAccessor
}

func (cd *CloudDriveServer) cluster() cluster.Cluster {
	return cd.server.cluster()
}

func (cd *CloudDriveServer) Detach(
	ctx context.Context, req *api.SdkCloudDriveSetDetachRequest) (*api.SdkCloudDriveSetDetachResponse, error) {
	if cd.cluster() == nil {
		return nil, status.Error(codes.Unavailable, errors.ErrResourceNotInitialized.Error())
	}

	return cd.cluster().Detach(ctx, req)
}
