package unifi

import (
	"encoding/json"
	"fmt"
)

// GetIntegrationDeviceStats returns statistics for a single device from the Integration/v1 API.
func (u *Unifi) GetIntegrationDeviceStats(site *IntegrationSite, deviceID string) (*IntegrationDeviceStats, error) {
	if u == nil {
		return nil, ErrNilUnifi
	}

	if u.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}

	if site == nil {
		return nil, ErrNoSiteProvided
	}

	if site.ID == "" {
		return nil, fmt.Errorf("site %q has an empty ID; cannot construct Integration/v1 API path", site.Name)
	}

	if deviceID == "" {
		return nil, fmt.Errorf("deviceID must not be empty")
	}

	u.DebugLog("Polling Integration/v1 for device stats, site %s device %s", site.Name, deviceID)

	path := fmt.Sprintf(APIIntegrationDeviceStatsPath, site.ID, deviceID)

	body, err := u.GetJSON(path)
	if err != nil {
		return nil, fmt.Errorf("fetching device stats for site %s device %s: %w", site.Name, deviceID, err)
	}

	var stats IntegrationDeviceStats

	if err := json.Unmarshal(body, &stats); err != nil {
		return nil, fmt.Errorf("parsing device stats for site %s device %s: %w", site.Name, deviceID, err)
	}

	// The payload does not echo the device ID; the one we asked for is authoritative.
	stats.DeviceID = deviceID
	stats.SiteName = site.Name

	return &stats, nil
}

// UnmarshalJSON accepts the shape Network 9.3.43+ actually returns, a single
// "uplink" object and the radios nested under "interfaces", as well as the
// flat "uplinks" and "radios" arrays the struct exposes.
func (s *IntegrationDeviceStats) UnmarshalJSON(b []byte) error {
	type plain IntegrationDeviceStats

	var wire struct {
		plain

		Uplink     *IntegrationDeviceUplinkStats `json:"uplink"`
		Interfaces struct {
			Radios []IntegrationDeviceRadioStats `json:"radios"`
		} `json:"interfaces"`
	}

	if err := json.Unmarshal(b, &wire); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}

	*s = IntegrationDeviceStats(wire.plain)

	if len(s.Uplinks) == 0 && wire.Uplink != nil {
		s.Uplinks = []IntegrationDeviceUplinkStats{*wire.Uplink}
	}

	if len(s.Radios) == 0 && len(wire.Interfaces.Radios) > 0 {
		s.Radios = wire.Interfaces.Radios
	}

	return nil
}

// GetIntegrationDevices returns the Integration/v1 device list for a site,
// including the identity fields (ID, name, MAC, model) the statistics
// endpoint omits.
func (u *Unifi) GetIntegrationDevices(site *IntegrationSite) ([]*IntegrationDevice, error) {
	if u == nil {
		return nil, ErrNilUnifi
	}

	if u.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}

	if site == nil {
		return nil, ErrNoSiteProvided
	}

	if site.ID == "" {
		return nil, fmt.Errorf("site %q has an empty ID; cannot construct Integration/v1 API path", site.Name)
	}

	u.DebugLog("Polling Integration/v1 for devices, site %s", site.Name)

	devices, err := getIntegrationList[*IntegrationDevice](u, fmt.Sprintf(APIIntegrationDevicesPath, site.ID))
	if err != nil {
		return nil, fmt.Errorf("fetching device list for site %s: %w", site.Name, err)
	}

	for _, dev := range devices {
		dev.SiteName = site.Name
	}

	return devices, nil
}

// GetAllIntegrationDeviceStats returns statistics for all devices in a site.
// If fetching stats for a device fails, partial results collected so far are returned alongside the error.
func (u *Unifi) GetAllIntegrationDeviceStats(site *IntegrationSite) ([]*IntegrationDeviceStats, error) {
	if u == nil {
		return nil, ErrNilUnifi
	}

	if u.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}

	if site == nil {
		return nil, ErrNoSiteProvided
	}

	if site.ID == "" {
		return nil, fmt.Errorf("site %q has an empty ID; cannot construct Integration/v1 API path", site.Name)
	}

	u.DebugLog("Polling Integration/v1 for all device stats, site %s", site.Name)

	devices, err := u.GetIntegrationDevices(site)
	if err != nil {
		return nil, err
	}

	result := make([]*IntegrationDeviceStats, 0, len(devices))

	skipped := 0

	for _, dev := range devices {
		if dev.ID == "" {
			skipped++

			continue
		}

		stats, err := u.GetIntegrationDeviceStats(site, dev.ID)
		if err != nil {
			// DebugLog provides supplementary context; the returned error is the primary signal for the caller.
			if skipped > 0 {
				u.DebugLog("Skipped %d/%d devices with empty IDs in site %s before error", skipped, len(devices), site.Name)
			}

			return result, fmt.Errorf("fetching stats for device %s in site %s: %w", dev.ID, site.Name, err)
		}

		// Identity comes from the device list; the statistics payload has none.
		stats.Name = dev.Name
		stats.MAC = dev.MACAddress
		stats.Model = dev.Model

		result = append(result, stats)
	}

	// DebugLog rather than ErrorLog: an empty device ID is a server data quality issue (e.g.,
	// a device mid-adoption), not a code error. ErrorLog would produce false production alerts.
	if skipped > 0 {
		u.DebugLog("Skipped %d/%d devices with empty IDs in site %s", skipped, len(devices), site.Name)
	}

	return result, nil
}
