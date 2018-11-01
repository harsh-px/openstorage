package softlayer

import (
	"fmt"
	"testing"

	"github.com/libopenstorage/openstorage/pkg/storageops"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestAll(t *testing.T) {
	_, err := storageops.GetEnvValueStrict("SL_USERNAME")
	if err != nil {
		t.Skipf("SL_USERNAME not defined. Skipping softlayer tests")
	}

	_, err = storageops.GetEnvValueStrict("SL_API_KEY")
	if err != nil {
		t.Skipf("SL_API_KEY not defined. Skipping softlayer tests")
	}

	datacenter, err := storageops.GetEnvValueStrict("SL_DATACENTER")
	if err != nil {
		t.Skipf("SL_DATACENTER not defined. Skipping softlayer tests")
	}

	slOps, err := NewClient(datacenter)
	require.NoError(t, err, "failed to create softlayer client")

	testDevice := &SoftLayerBlockDisk{
		StorageType:   Endurance,
		Capacity:      25,
		HourlyBilling: false,
		IOPS:          2.0,
	}

	returnObj, err := slOps.Create(testDevice, nil)
	require.NoError(t, err, "failed to create device")
	require.NotNil(t, returnObj, "got nil device for create call")

	createdDevice, typeOk := returnObj.(*SoftLayerBlockDisk)
	require.True(t, typeOk, "got invalid device for softlayer block dev")

	logrus.Infof("[debug] created softlayer device with ID: %d", createdDevice.id)

	err = slOps.Delete(fmt.Sprintf("%d", createdDevice.id))
	require.NoError(t, err, "failed to delete device")
	logrus.Infof("[debug] deleted softlayer device with ID: %d", createdDevice.id)
}
