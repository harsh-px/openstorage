package vsphere

import (
	"os"
	"strings"

	"k8s.io/kubernetes/pkg/cloudprovider/providers/vsphere"
	"k8s.io/kubernetes/pkg/cloudprovider/providers/vsphere/vclib"
)

func getVSphereConfig() *vsphere.VSphereConfig {
	var cfg VSphereConfig
	cfg.Global.VCenterIP = os.Getenv("VSPHERE_VCENTER")
	cfg.Global.VCenterPort = os.Getenv("VSPHERE_VCENTER_PORT")
	cfg.Global.User = os.Getenv("VSPHERE_USER")
	cfg.Global.Password = os.Getenv("VSPHERE_PASSWORD")
	cfg.Global.Datacenters = os.Getenv("VSPHERE_DATACENTER")
	cfg.Global.DefaultDatastore = os.Getenv("VSPHERE_DATASTORE")
	cfg.Global.WorkingDir = os.Getenv("VSPHERE_WORKING_DIR")
	cfg.Global.VMName = os.Getenv("VSPHERE_VM_NAME")
	cfg.Global.InsecureFlag = false
	if strings.ToLower(os.Getenv("VSPHERE_INSECURE")) == "true" {
		cfg.Global.InsecureFlag = true
	}
	cfg.Workspace.VCenterIP = cfg.Global.VCenterIP
	cfg.Workspace.Datacenter = cfg.Global.Datacenters
	cfg.Workspace.DefaultDatastore = cfg.Global.DefaultDatastore
	cfg.Workspace.Folder = cfg.Global.WorkingDir
	return &cfg
}

func getVSphereConn(cfg *VSphereConfig) *vclib.VSphereConnection {
	vSphereConn := &vclib.VSphereConnection{
		Username:          cfg.Global.User,
		Password:          cfg.Global.Password,
		Hostname:          cfg.Global.VCenterIP,
		Insecure:          cfg.Global.InsecureFlag,
		RoundTripperCount: cfg.Global.RoundTripperCount,
		Port:              cfg.Global.VCenterPort,
	}
	return vSphereConn
}
