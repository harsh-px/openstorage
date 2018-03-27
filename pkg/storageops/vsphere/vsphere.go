package vsphere

type vsphereOps struct {
}

// Name returns name of the storage operations driver
func (ops *vsphereOps) Name() string {
}

// Create volume based on input template volume and also apply given labels.
func (ops *vsphereOps) Create(template interface{}, labels map[string]string) (interface{}, error) {
}

// GetDeviceID returns ID/Name of the given device/disk or snapshot
func (ops *vsphereOps) GetDeviceID(template interface{}) (string, error) {
}

// Attach volumeID.
// Return attach path.
func (ops *vsphereOps) Attach(volumeID string) (string, error) {
}

// Detach volumeID.
func (ops *vsphereOps) Detach(volumeID string) error {
}

// Delete volumeID.
func (ops *vsphereOps) Delete(volumeID string) error {
}

// Desribe an instance
func (ops *vsphereOps) Describe() (interface{}, error) {
}

// FreeDevices returns free block devices on the instance.
// blockDeviceMappings is a data structure that contains all block devices on
// the instance and where they are mapped to
func (ops *vsphereOps) FreeDevices(blockDeviceMappings []interface{}, rootDeviceName string) ([]string, error) {
}

// Inspect volumes specified by volumeID
func (ops *vsphereOps) Inspect(volumeIds []*string) ([]interface{}, error) {
}

// DeviceMappings returns map[local_attached_volume_path]->volume ID/NAME
func (ops *vsphereOps) DeviceMappings() (map[string]string, error) {
}

// Enumerate volumes that match given filters. Organize them into
// sets identified by setIdentifier.
// labels can be nil, setIdentifier can be empty string.
func (ops *vsphereOps) Enumerate(volumeIds []*string,
	labels map[string]string,
	setIdentifier string,
) (map[string][]interface{}, error) {
}

// DevicePath for the given volume i.e path where it's attached
func (ops *vsphereOps) DevicePath(volumeID string) (string, error) {
}

// Snapshot the volume with given volumeID
func (ops *vsphereOps) Snapshot(volumeID string, readonly bool) (interface{}, error) {
}

// SnapshotDelete deletes the snapshot with given ID
func (ops *vsphereOps) SnapshotDelete(snapID string) error {
}

// ApplyTags will apply given labels/tags on the given volume
func (ops *vsphereOps) ApplyTags(volumeID string, labels map[string]string) error {
}

// RemoveTags removes labels/tags from the given volume
func (ops *vsphereOps) RemoveTags(volumeID string, labels map[string]string) error {
}

// Tags will list the existing labels/tags on the given volume
func (ops *vsphereOps) Tags(volumeID string) (map[string]string, error) {
}
