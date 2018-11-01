package softlayer

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/libopenstorage/openstorage/pkg/storageops"
	"github.com/sirupsen/logrus"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/helpers/location"
	"github.com/softlayer/softlayer-go/helpers/network"
	"github.com/softlayer/softlayer-go/helpers/product"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
	"github.com/softlayer/softlayer-go/sl"
)

type slOps struct {
	softlayerUsername    string
	softlayerAPIKey      string
	softlayerEndpointUrl string
	softlayerTimeout     int
	datacenter           string
	session              *session.Session
}

type SoftLayerStorageType string

const (
	Endurance   SoftLayerStorageType = "endurance"
	Performance SoftLayerStorageType = "performance"
)

const (
	blockStorage       = "block"
	storagePackageType = "STORAGE_AS_A_SERVICE"
	storageMask        = "id,billingItem.orderItem.order.id"
	itemMask           = "id,capacity,description,units,keyName,capacityMinimum,capacityMaximum,prices[id,categories[id,name,categoryCode],capacityRestrictionMinimum,capacityRestrictionMaximum,capacityRestrictionType,locationGroupId],itemCategory[categoryCode]"
)

var (
	// Map IOPS value to endurance storage tier keyName in SoftLayer_Product_Item
	enduranceIopsMap = map[float64]string{
		0.25: "LOW_INTENSITY_TIER",
		2:    "READHEAVY_TIER",
		4:    "WRITEHEAVY_TIER",
		10:   "10_IOPS_PER_GB",
	}

	// Map IOPS value to endurance storage tier capacityRestrictionMaximum/capacityRestrictionMinimum in SoftLayer_Product_Item
	enduranceCapacityRestrictionMap = map[float64]int{
		0.25: 100,
		2:    200,
		4:    300,
		10:   1000,
	}

	snapshotDay = map[string]string{
		"0": "SUNDAY",
		"1": "MONDAY",
		"2": "TUESDAY",
		"3": "WEDNESDAY",
		"4": "THURSDAY",
		"5": "FRIDAY",
		"6": "SATURDAY",
	}
)

type SoftLayerOsFormatType string

const (
	Linux                       SoftLayerOsFormatType = "Linux"
	HyperV                      SoftLayerOsFormatType = "Hyper-V"
	VMware                      SoftLayerOsFormatType = "VMWare"
	Xen                         SoftLayerOsFormatType = "Xen"
	defaultSoftlayerEndpointUrl                       = "https://api.softlayer.com/rest/v3"
)

type SoftLayerBlockDisk struct {
	StorageType      SoftLayerStorageType
	IOPS             float64
	Capacity         int
	SnapshotCapacity int
	OSFormatType     SoftLayerOsFormatType
	HourlyBilling    bool

	// private fields
	id int
}

// NewClient creates a new softlayer client
func NewClient(datacenter string) (storageops.Ops, error) {
	if len(datacenter) == 0 {
		return nil, fmt.Errorf("Softlayer datacenter is required for initializing client")
	}

	ops := &slOps{
		datacenter: datacenter,
	}

	/*ops.softlayerUsername, err = storageops.GetEnvValueStrict("SL_API_KEY")
	if err != nil {
		return nil, err
	}

	ops.softlayerEndpointUrl, err = storageops.GetEnvValueStrict("SL_ENDPOINT_URL")
	if err != nil {
		ops.softlayerEndpointUrl = defaultSoftlayerEndpointUrl
	}

	softlayerTimeout, err := storageops.GetEnvValueStrict("SL_TIMEOUT")
	if err == nil {
		timeoutInt, err := strconv.Atoi(softlayerTimeout)
		if err != nil {
			return nil, err
		}
		ops.softlayerTimeout = timeoutInt
	}*/

	ops.session = session.New()
	ops.session.AppendUserAgent(fmt.Sprint("libopenstorage/softlayer-client"))

	logrus.Infof("[debug] Using ops: %s", ops)

	return ops, nil
}

func (s *slOps) Name() string { return "softlayer" }

func (s *slOps) InstanceID() string {
	return "" // TODO return instance name
}

func (s *slOps) ApplyTags(
	diskName string,
	labels map[string]string) error {
	return nil
}

func (s *slOps) Attach(diskName string) (string, error) {
	return "", nil // TODO not supported
}

func (s *slOps) Create(
	template interface{},
	labels map[string]string,
) (interface{}, error) {
	diskSpec, typeOk := template.(*SoftLayerBlockDisk)
	if !typeOk {
		return nil, fmt.Errorf("invalid disk spec provided for softlayer disk: %v", template)
	}

	storageType := strings.ToLower(string(diskSpec.StorageType))
	iops := diskSpec.IOPS
	capacity := diskSpec.Capacity
	snapshotCapacity := diskSpec.SnapshotCapacity
	osFormatType := diskSpec.OSFormatType
	if len(osFormatType) == 0 {
		osFormatType = Linux
	}
	osType, err := network.GetOsTypeByName(s.session, string(osFormatType))
	hourlyBilling := diskSpec.HourlyBilling
	datacenter := s.datacenter

	storageOrderContainer, err := s.buildStorageProductOrderContainer(
		storageType, iops, capacity, snapshotCapacity, blockStorage, datacenter, hourlyBilling)
	if err != nil {
		return nil, fmt.Errorf("Error while creating storage:%s", err)
	}

	log.Println("[INFO] Creating storage")

	var receipt datatypes.Container_Product_Order_Receipt

	switch diskSpec.StorageType {
	case Endurance:
		receipt, err = services.GetProductOrderService(s.session.SetRetries(0)).PlaceOrder(
			&datatypes.Container_Product_Order_Network_Storage_AsAService{
				Container_Product_Order: storageOrderContainer,
				OsFormatType: &datatypes.Network_Storage_Iscsi_OS_Type{
					Id:      osType.Id,
					KeyName: osType.KeyName,
				},
				VolumeSize: &capacity,
			}, sl.Bool(false))
	case Performance:
		receipt, err = services.GetProductOrderService(s.session.SetRetries(0)).PlaceOrder(
			&datatypes.Container_Product_Order_Network_Storage_AsAService{
				Container_Product_Order: storageOrderContainer,
				OsFormatType: &datatypes.Network_Storage_Iscsi_OS_Type{
					Id:      osType.Id,
					KeyName: osType.KeyName,
				},
				Iops:       sl.Int(int(iops)),
				VolumeSize: &capacity,
			}, sl.Bool(false))
	default:
		return nil, fmt.Errorf("Error during creation of storage: Invalid storageType %s", storageType)
	}

	if err != nil {
		return nil, fmt.Errorf("Error during creation of storage: %s", err)
	}

	// Find the storage device
	blockStorage, err := s.findStorageByOrderId(*receipt.OrderId)
	if err != nil {
		return nil, fmt.Errorf("Error during creation of storage: %s", err)
	}
	diskSpec.id = *blockStorage.Id

	// Wait for storage availability
	_, err = s.waitForStorageAvailable(diskSpec)
	if err != nil {
		return nil, fmt.Errorf(
			"Error waiting for storage (%d) to become ready: %s", diskSpec.id, err)
	}

	// SoftLayer changes the device ID after completion of provisioning. It is necessary to refresh device ID.
	blockStorage, err = s.findStorageByOrderId(*receipt.OrderId)
	if err != nil {
		return nil, fmt.Errorf("Error during creation of storage: %s", err)
	}
	diskSpec.id = *blockStorage.Id

	log.Printf("[INFO] Storage ID: %d", diskSpec.id)

	/* TODO check return type
	* err = resourceIBMStorageBlockUpdate(diskSpec, meta)
	if err != nil {
		return nil, err
	}*/

	return diskSpec, nil
}

// Waits for storage provisioning
func (s *slOps) waitForStorageAvailable(d *SoftLayerBlockDisk) (interface{}, error) {
	log.Printf("Waiting for storage (%d) to be available.", d.id)
	id := d.id
	stateConf := &resource.StateChangeConf{
		Pending: []string{"retry", "provisioning"},
		Target:  []string{"available"},
		Refresh: func() (interface{}, string, error) {
			// Check active transactions
			service := services.GetNetworkStorageService(s.session)
			result, err := service.Id(id).Mask("activeTransactionCount").GetObject()
			if err != nil {
				if apiErr, ok := err.(sl.Error); ok && apiErr.StatusCode == 404 {
					return nil, "", fmt.Errorf("Error retrieving storage: %s", err)
				}
				return false, "retry", nil
			}

			log.Println("Checking active transactions.")
			if *result.ActiveTransactionCount > 0 {
				return result, "provisioning", nil
			}

			// Check volume status.
			log.Println("Checking volume status.")
			resultStr := ""
			err = s.session.DoRequest(
				"SoftLayer_Network_Storage",
				"getObject",
				nil,
				&sl.Options{Id: &id, Mask: "volumeStatus"},
				&resultStr,
			)
			if err != nil {
				return false, "retry", nil
			}

			if !strings.Contains(resultStr, "PROVISION_COMPLETED") &&
				!strings.Contains(resultStr, "Volume Provisioning has completed") {
				return result, "provisioning", nil
			}

			return result, "available", nil
		},
		Timeout:    45 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}

	return stateConf.WaitForState()
}

func (s *slOps) findStorageByOrderId(orderId int) (datatypes.Network_Storage, error) {
	filterPath := "networkStorage.billingItem.orderItem.order.id"

	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  []string{"complete"},
		Refresh: func() (interface{}, string, error) {
			storage, err := services.GetAccountService(s.session).
				Filter(filter.Build(
					filter.Path(filterPath).
						Eq(strconv.Itoa(orderId)))).
				Mask(storageMask).
				GetNetworkStorage()
			if err != nil {
				return datatypes.Network_Storage{}, "", err
			}

			if len(storage) == 1 {
				return storage[0], "complete", nil
			} else if len(storage) == 0 {
				return nil, "pending", nil
			} else {
				return nil, "", fmt.Errorf("Expected one Storage: %s", err)
			}
		},
		Timeout:        45 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     10 * time.Second,
		NotFoundChecks: 300,
	}

	pendingResult, err := stateConf.WaitForState()

	if err != nil {
		return datatypes.Network_Storage{}, err
	}

	var result, ok = pendingResult.(datatypes.Network_Storage)

	if ok {
		return result, nil
	}

	return datatypes.Network_Storage{},
		fmt.Errorf("Cannot find Storage with order id '%d'", orderId)
}

func (s *slOps) buildStorageProductOrderContainer(storageType string,
	iops float64,
	capacity int,
	snapshotCapacity int,
	storageProtocol string,
	datacenter string,
	hourlyBilling bool) (datatypes.Container_Product_Order, error) {

	// Get a package type)
	pkg, err := product.GetPackageByType(s.session, storagePackageType)
	if err != nil {
		return datatypes.Container_Product_Order{}, err
	}

	// Get all prices
	productItems, err := product.GetPackageProducts(s.session, *pkg.Id, itemMask)
	if err != nil {
		return datatypes.Container_Product_Order{}, err
	}

	// Add IOPS price
	targetItemPrices := []datatypes.Product_Item_Price{}

	if storageType == "Performance" {
		price, err := getPriceByCategory(productItems, "storage_as_a_service")
		if err != nil {
			return datatypes.Container_Product_Order{}, err
		}
		targetItemPrices = append(targetItemPrices, price)
		price, err = getPriceByCategory(productItems, "storage_"+storageProtocol)
		if err != nil {
			return datatypes.Container_Product_Order{}, err
		}
		targetItemPrices = append(targetItemPrices, price)

		price, err = getSaaSPerformSpacePrice(productItems, capacity)
		if err != nil {
			return datatypes.Container_Product_Order{}, err
		}
		targetItemPrices = append(targetItemPrices, price)

		price, err = getSaaSPerformIOPSPrice(productItems, capacity, int(iops))
		if err != nil {
			return datatypes.Container_Product_Order{}, err
		}
		targetItemPrices = append(targetItemPrices, price)

	} else {

		price, err := getPriceByCategory(productItems, "storage_as_a_service")
		if err != nil {
			return datatypes.Container_Product_Order{}, err
		}
		targetItemPrices = append(targetItemPrices, price)
		price, err = getPriceByCategory(productItems, "storage_"+storageProtocol)
		if err != nil {
			return datatypes.Container_Product_Order{}, err
		}
		targetItemPrices = append(targetItemPrices, price)

		price, err = getSaaSEnduranceSpacePrice(productItems, capacity, iops)
		if err != nil {
			return datatypes.Container_Product_Order{}, err
		}
		targetItemPrices = append(targetItemPrices, price)

		price, err = getSaaSEnduranceTierPrice(productItems, iops)
		if err != nil {
			return datatypes.Container_Product_Order{}, err
		}
		targetItemPrices = append(targetItemPrices, price)

	}

	if snapshotCapacity > 0 {
		price, err := getSaaSSnapshotSpacePrice(productItems, snapshotCapacity, iops, storageType)
		if err != nil {
			return datatypes.Container_Product_Order{}, err
		}
		targetItemPrices = append(targetItemPrices, price)

	}

	// Lookup the data center ID
	dc, err := location.GetDatacenterByName(s.session, datacenter)
	if err != nil {
		return datatypes.Container_Product_Order{},
			fmt.Errorf("No data centers matching %s could be found", datacenter)
	}

	productOrderContainer := datatypes.Container_Product_Order{
		PackageId:        pkg.Id,
		Location:         sl.String(strconv.Itoa(*dc.Id)),
		Prices:           targetItemPrices,
		Quantity:         sl.Int(1),
		UseHourlyPricing: sl.Bool(hourlyBilling),
	}

	return productOrderContainer, nil
}

func getPriceByCategory(productItems []datatypes.Product_Item, priceCategory string) (datatypes.Product_Item_Price, error) {
	for _, item := range productItems {
		price := getPrice(item.Prices, priceCategory, "", 0)
		if price.Id != nil {
			return price, nil
		}
	}

	return datatypes.Product_Item_Price{},
		fmt.Errorf("No product items matching with category %s could be found", priceCategory)
}

/*func resourceIBMStorageBlockUpdate(d *schema.ResourceData, meta interface{}) error {
	sess := meta.(ClientSession).SoftLayerSession()
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	storage, err := services.GetNetworkStorageService(sess).
		Id(id).
		Mask(storageDetailMask).
		GetObject()

	if err != nil {
		return fmt.Errorf("Error updating storage information: %s", err)
	}

	// Update allowed_ip_addresses
	if d.HasChange("allowed_ip_addresses") {
		err := updateAllowedIpAddresses(d, sess, storage)
		if err != nil {
			return fmt.Errorf("Error updating storage information: %s", err)
		}
	}

	// Update allowed_subnets
	if d.HasChange("allowed_subnets") {
		err := updateAllowedSubnets(d, sess, storage)
		if err != nil {
			return fmt.Errorf("Error updating storage information: %s", err)
		}
	}

	// Update allowed_virtual_guest_ids
	if d.HasChange("allowed_virtual_guest_ids") {
		err := updateAllowedVirtualGuestIds(d, sess, storage)
		if err != nil {
			return fmt.Errorf("Error updating storage information: %s", err)
		}
	}

	// Update allowed_hardware_ids
	if d.HasChange("allowed_hardware_ids") {
		err := updateAllowedHardwareIds(d, sess, storage)
		if err != nil {
			return fmt.Errorf("Error updating storage information: %s", err)
		}
	}

	// Update notes
	if d.HasChange("notes") {
		err := updateNotes(d, sess, storage)
		if err != nil {
			return fmt.Errorf("Error updating storage information: %s", err)
		}
	}

	return resourceIBMStorageBlockRead(d, meta)
}

func resourceIBMStorageBlockRead(d *schema.ResourceData, meta interface{}) error {
	sess := meta.(ClientSession).SoftLayerSession()
	storageId, _ := strconv.Atoi(d.Id())

	storage, err := services.GetNetworkStorageService(sess).
		Id(storageId).
		Mask(storageDetailMask).
		GetObject()

	if err != nil {
		return fmt.Errorf("Error retrieving storage information: %s", err)
	}

	storageType := strings.Fields(*storage.StorageType.Description)[0]

	// Calculate IOPS
	iops, err := getIops(storage, storageType)
	if err != nil {
		return fmt.Errorf("Error retrieving storage information: %s", err)
	}

	d.Set("type", storageType)
	d.Set("capacity", *storage.CapacityGb)
	d.Set("volumename", *storage.Username)
	d.Set("hostname", *storage.ServiceResourceBackendIpAddress)
	d.Set("iops", iops)
	if storage.SnapshotCapacityGb != nil {
		snapshotCapacity, _ := strconv.Atoi(*storage.SnapshotCapacityGb)
		d.Set("snapshot_capacity", snapshotCapacity)
	}

	// Parse data center short name from ServiceResourceName. For example,
	// if SoftLayer API returns "'serviceResourceName': 'PerfStor Aggr aggr_staasdal0601_p01'",
	// the data center short name is "dal06".
	r, _ := regexp.Compile("[a-zA-Z]{3}[0-9]{2}")
	d.Set("datacenter", r.FindString(*storage.ServiceResourceName))

	allowedHostInfoList := make([]map[string]interface{}, 0)

	// Read allowed_ip_addresses
	allowedIpaddressesList := make([]string, 0, len(storage.AllowedIpAddresses))
	for _, allowedIpaddress := range storage.AllowedIpAddresses {
		singleHost := make(map[string]interface{})
		singleHost["id"] = *allowedIpaddress.SubnetId
		singleHost["username"] = *allowedIpaddress.AllowedHost.Credential.Username
		singleHost["password"] = *allowedIpaddress.AllowedHost.Credential.Password
		singleHost["host_iqn"] = *allowedIpaddress.AllowedHost.Name
		allowedHostInfoList = append(allowedHostInfoList, singleHost)
		allowedIpaddressesList = append(allowedIpaddressesList, *allowedIpaddress.IpAddress)
	}
	d.Set("allowed_ip_addresses", allowedIpaddressesList)

	// Read allowed_virtual_guest_ids and allowed_host_info
	allowedVirtualGuestInfoList := make([]map[string]interface{}, 0)
	allowedVirtualGuestIdsList := make([]int, 0, len(storage.AllowedVirtualGuests))

	for _, allowedVirtualGuest := range storage.AllowedVirtualGuests {
		singleVirtualGuest := make(map[string]interface{})
		singleVirtualGuest["id"] = *allowedVirtualGuest.Id
		singleVirtualGuest["username"] = *allowedVirtualGuest.AllowedHost.Credential.Username
		singleVirtualGuest["password"] = *allowedVirtualGuest.AllowedHost.Credential.Password
		singleVirtualGuest["host_iqn"] = *allowedVirtualGuest.AllowedHost.Name
		allowedHostInfoList = append(allowedHostInfoList, singleVirtualGuest)
		allowedVirtualGuestInfoList = append(allowedVirtualGuestInfoList, singleVirtualGuest)
		allowedVirtualGuestIdsList = append(allowedVirtualGuestIdsList, *allowedVirtualGuest.Id)
	}
	d.Set("allowed_virtual_guest_ids", allowedVirtualGuestIdsList)
	d.Set("allowed_virtual_guest_info", allowedVirtualGuestInfoList)

	// Read allowed_hardware_ids and allowed_host_info
	allowedHardwareInfoList := make([]map[string]interface{}, 0)
	allowedHardwareIdsList := make([]int, 0, len(storage.AllowedHardware))
	for _, allowedHW := range storage.AllowedHardware {
		singleHardware := make(map[string]interface{})
		singleHardware["id"] = *allowedHW.Id
		singleHardware["username"] = *allowedHW.AllowedHost.Credential.Username
		singleHardware["password"] = *allowedHW.AllowedHost.Credential.Password
		singleHardware["host_iqn"] = *allowedHW.AllowedHost.Name
		allowedHostInfoList = append(allowedHostInfoList, singleHardware)
		allowedHardwareInfoList = append(allowedHardwareInfoList, singleHardware)
		allowedHardwareIdsList = append(allowedHardwareIdsList, *allowedHW.Id)
	}
	d.Set("allowed_hardware_ids", allowedHardwareIdsList)
	d.Set("allowed_hardware_info", allowedHardwareInfoList)
	d.Set("allowed_host_info", allowedHostInfoList)

	if storage.OsType != nil {
		d.Set("os_format_type", *storage.OsType.Name)
	}

	if storage.Notes != nil {
		d.Set("notes", *storage.Notes)
	}

	if storage.BillingItem != nil {
		d.Set("hourly_billing", storage.BillingItem.HourlyFlag)
	}

	return nil
}

func getIops(storage datatypes.Network_Storage, storageType string) (float64, error) {
	switch storageType {
	case enduranceType:
		for _, property := range storage.Properties {
			if *property.Type.Keyname == "PROVISIONED_IOPS" {
				provisionedIops, err := strconv.Atoi(*property.Value)
				if err != nil {
					return 0, err
				}
				enduranceIops := float64(provisionedIops / *storage.CapacityGb)
				if enduranceIops < 1 {
					enduranceIops = 0.25
				}
				return enduranceIops, nil
			}
		}
	case performanceType:
		if storage.Iops == nil {
			return 0, fmt.Errorf("Failed to retrieve iops information.")
		}
		iops, err := strconv.Atoi(*storage.Iops)
		if err != nil {
			return 0, err
		}
		return float64(iops), nil
	}
	return 0, fmt.Errorf("Invalid storage type %s", storageType)
}*/

func (s *slOps) DeleteFrom(id, _ string) error {
	return s.Delete(id)
}

func (s *slOps) Delete(id string) error {
	storageService := services.GetNetworkStorageService(s.session)
	storageID, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	// Get billing item associated with the storage
	billingItem, err := storageService.Id(storageID).GetBillingItem()

	if err != nil {
		return fmt.Errorf("Error while looking up billing item associated with the storage: %s", err)
	}

	if billingItem.Id == nil {
		return fmt.Errorf("Error while looking up billing item associated with the storage: No billing item for ID:%d", storageID)
	}

	success, err := services.GetBillingItemService(s.session).Id(*billingItem.Id).CancelService()
	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("SoftLayer reported an unsuccessful cancellation")
	}
	return nil
}

func (s *slOps) Detach(devicePath string) error {
	return s.detachInternal(devicePath, "") // TODO return not supported
}

func (s *slOps) DetachFrom(devicePath, instanceName string) error {
	return s.detachInternal(devicePath, instanceName)
}

func (s *slOps) detachInternal(devicePath, instanceName string) error {
	return "", storageops.ErrNotSupported
	return nil // TODO return not supported
}

func (s *slOps) DeviceMappings() (map[string]string, error) {
	return nil, storageops.ErrNotSupported
}

func (s *slOps) DevicePath(diskName string) (string, error) {
	return "", storageops.ErrNotSupported
}

func (s *slOps) Enumerate(
	volumeIds []*string,
	labels map[string]string,
	setIdentifier string,
) (map[string][]interface{}, error) {
	return "", storageops.ErrNotSupported
}

func (s *slOps) FreeDevices(
	blockDeviceMappings []interface{},
	rootDeviceName string,
) ([]string, error) {
	return nil, fmt.Errorf("function not implemented")
}

func (s *slOps) GetDeviceID(disk interface{}) (string, error) {
	createdDevice, typeOk := returnObj.(*SoftLayerBlockDisk)
	if !typeOk {
		return "", fmt.Errorf("received invalid disk in the get device ID call")
	}

	return fmt.Sprintf("%s", createdDevice.id), nil
}

func (s *slOps) Inspect(diskNames []*string) ([]interface{}, error) {
	return nil, nil
}

func (s *slOps) RemoveTags(
	diskName string,
	labels map[string]string,
) error {
	return storageops.ErrNotSupported
}

func (s *slOps) Snapshot(
	disk string,
	readonly bool,
) (interface{}, error) {
	return nil, storageops.ErrNotSupported
}

func (s *slOps) SnapshotDelete(snapID string) error {
	return storageops.ErrNotSupported
}

func (s *slOps) Tags(diskName string) (map[string]string, error) {
	return nil, storageops.ErrNotSupported
}

// Describe current instance.
func (s *slOps) Describe() (interface{}, error) {
	return nil, storageops.ErrNotSupported
}

func getPrice(prices []datatypes.Product_Item_Price, category, restrictionType string, restrictionValue int) datatypes.Product_Item_Price {
	for _, price := range prices {

		if price.LocationGroupId != nil || *price.Categories[0].CategoryCode != category {
			continue
		}

		if restrictionType != "" && restrictionValue > 0 {

			capacityRestrictionMinimum, _ := strconv.Atoi(*price.CapacityRestrictionMinimum)
			capacityRestrictionMaximum, _ := strconv.Atoi(*price.CapacityRestrictionMaximum)
			if restrictionType != *price.CapacityRestrictionType || restrictionValue < capacityRestrictionMinimum || restrictionValue > capacityRestrictionMaximum {
				continue
			}

		}

		return price

	}

	return datatypes.Product_Item_Price{}

}

func getSaaSPerformSpacePrice(productItems []datatypes.Product_Item, size int) (datatypes.Product_Item_Price, error) {

	for _, item := range productItems {

		category, ok := sl.GrabOk(item, "ItemCategory.CategoryCode")
		if ok && category != "performance_storage_space" {
			continue
		}
		if item.CapacityMinimum == nil || item.CapacityMaximum == nil {
			continue
		}

		capacityMinimum, _ := strconv.Atoi(*item.CapacityMinimum)
		capacityMaximum, _ := strconv.Atoi(*item.CapacityMaximum)

		if size < capacityMinimum ||
			size > capacityMaximum {
			continue
		}

		keyname := fmt.Sprintf("%d_%d_GBS", capacityMinimum, capacityMaximum)
		if *item.KeyName != keyname {
			continue
		}

		price := getPrice(item.Prices, "performance_storage_space", "", 0)
		if price.Id != nil {
			return price, nil
		}
	}

	return datatypes.Product_Item_Price{},
		fmt.Errorf("Could not find price for performance storage space")

}

func getSaaSPerformIOPSPrice(productItems []datatypes.Product_Item, size, iops int) (datatypes.Product_Item_Price, error) {

	for _, item := range productItems {

		category, ok := sl.GrabOk(item, "ItemCategory.CategoryCode")
		if ok && category != "performance_storage_iops" {
			continue
		}

		if item.CapacityMinimum == nil || item.CapacityMaximum == nil {
			continue
		}

		capacityMinimum, _ := strconv.Atoi(*item.CapacityMinimum)
		capacityMaximum, _ := strconv.Atoi(*item.CapacityMaximum)

		if iops < capacityMinimum ||
			iops > capacityMaximum {
			continue
		}

		price := getPrice(item.Prices, "performance_storage_iops", "STORAGE_SPACE", size)
		if price.Id != nil {
			return price, nil
		}
	}

	return datatypes.Product_Item_Price{},
		fmt.Errorf("Could not find price for iops for the given volume")

}

func getSaaSEnduranceSpacePrice(productItems []datatypes.Product_Item, size int, iops float64) (datatypes.Product_Item_Price, error) {

	var keyName string
	if iops != 0.25 {
		tiers := int(iops)
		keyName = fmt.Sprintf("STORAGE_SPACE_FOR_%d_IOPS_PER_GB", tiers)
	} else {

		keyName = "STORAGE_SPACE_FOR_0_25_IOPS_PER_GB"

	}

	for _, item := range productItems {

		if *item.KeyName != keyName {
			continue
		}

		if item.CapacityMinimum == nil || item.CapacityMaximum == nil {
			continue
		}

		capacityMinimum, _ := strconv.Atoi(*item.CapacityMinimum)
		capacityMaximum, _ := strconv.Atoi(*item.CapacityMaximum)

		if size < capacityMinimum ||
			size > capacityMaximum {
			continue
		}

		price := getPrice(item.Prices, "performance_storage_space", "", 0)
		if price.Id != nil {
			return price, nil
		}
	}

	return datatypes.Product_Item_Price{},
		fmt.Errorf("Could not find price for endurance storage space")

}

func getSaaSEnduranceTierPrice(productItems []datatypes.Product_Item, iops float64) (datatypes.Product_Item_Price, error) {

	targetCapacity := enduranceCapacityRestrictionMap[iops]

	for _, item := range productItems {

		category, ok := sl.GrabOk(item, "ItemCategory.CategoryCode")
		if ok && category != "storage_tier_level" {
			continue
		}

		if int(*item.Capacity) != targetCapacity {
			continue
		}

		price := getPrice(item.Prices, "storage_tier_level", "", 0)
		if price.Id != nil {
			return price, nil
		}
	}

	return datatypes.Product_Item_Price{},
		fmt.Errorf("Could not find price for endurance tier level")

}

func getSaaSSnapshotSpacePrice(productItems []datatypes.Product_Item, size int, iops float64, volumeType string) (datatypes.Product_Item_Price, error) {

	var targetValue int
	var targetRestrictionType string
	if volumeType == "Performance" {
		targetValue = int(iops)
		targetRestrictionType = "IOPS"
	} else {

		targetValue = enduranceCapacityRestrictionMap[iops]
		targetRestrictionType = "STORAGE_TIER_LEVEL"

	}

	for _, item := range productItems {

		if int(*item.Capacity) != size {
			continue
		}

		price := getPrice(item.Prices, "storage_snapshot_space", targetRestrictionType, targetValue)
		if price.Id != nil {
			return price, nil
		}
	}

	return datatypes.Product_Item_Price{},
		fmt.Errorf("Could not find price for snapshot space")
}
