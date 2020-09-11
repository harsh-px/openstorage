package clouddrive

import (
	"context"

	"github.com/libopenstorage/openstorage/api"
	"github.com/libopenstorage/openstorage/api/errors"
)

// NewDefaultCloudDriveSetProvider returns an not suppported implementation of the cloud driveset server
func NewDefaultCloudDriveSetProvider() api.OpenStorageCloudDriveSetServer {
	return &UnsupportedCloudDriveSetProvider{}
}

type UnsupportedCloudDriveSetProvider struct {
}

func (n *UnsupportedCloudDriveSetProvider) Detach(
	ctx context.Context, req *api.SdkCloudDriveSetDetachRequest) (*api.SdkCloudDriveSetDetachResponse, error) {
	return nil, &errors.ErrNotSupported{}
}
