package unifi // nolint: testpackage

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const integrationTestSiteID = "11111111-1111-4111-8111-111111111111"

// Device list envelope and statistics payload as returned by a Network 10.6 console.
const (
	integrationDeviceListBody = `{"offset":0,"limit":200,"count":2,"totalCount":2,"data":[
		{"id":"dev-1","name":"switch-1","macAddress":"00:11:22:33:44:01","model":"USW Pro 24 PoE","state":"ONLINE"},
		{"id":"dev-2","name":"ap-1","macAddress":"00:11:22:33:44:02","model":"U6 Enterprise","state":"ONLINE"}]}`
	integrationDeviceStatsBody = `{"uptimeSec": 10, "cpuUtilizationPct": 5, "memoryUtilizationPct": 40,
		"uplink": {"txRateBps": 1, "rxRateBps": 2},
		"interfaces": {"radios": [{"frequencyGHz": 5, "txRetriesPct": 0.5}]}}`
)

func integrationDeviceServer(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()

	requested := &[]string{}
	base := "/proxy/network/integration/v1/sites/" + integrationTestSiteID + "/devices"
	mux := http.NewServeMux()

	mux.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
		*requested = append(*requested, r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(integrationDeviceListBody))
	})
	mux.HandleFunc(base+"/", func(w http.ResponseWriter, r *http.Request) {
		*requested = append(*requested, r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(integrationDeviceStatsBody))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv, requested
}

func integrationDeviceClient(srv *httptest.Server) *Unifi {
	return &Unifi{
		Client: &http.Client{},
		Config: &Config{URL: srv.URL, APIKey: "integration-key", DebugLog: discardLogs, ErrorLog: discardLogs},
		new:    true,
	}
}

// The statistics payload carries neither a device ID nor a name, so every
// series exported from it used to collapse onto an empty device_id label.
// The ID comes from the request, the identity from the device list.
func TestGetAllIntegrationDeviceStatsCarriesIdentity(t *testing.T) {
	t.Parallel()

	srv, requested := integrationDeviceServer(t)
	c := integrationDeviceClient(srv)
	site := &IntegrationSite{ID: integrationTestSiteID, InternalReference: "default", Name: "Default"}

	stats, err := c.GetAllIntegrationDeviceStats(site)
	require.NoError(t, err)
	require.Len(t, stats, 2)

	base := "/proxy/network/integration/v1/sites/" + integrationTestSiteID + "/devices"
	a := assert.New(t)
	a.Equal([]string{base, base + "/dev-1/statistics/latest", base + "/dev-2/statistics/latest"}, *requested)

	a.Equal("dev-1", stats[0].DeviceID)
	a.Equal("switch-1", stats[0].Name)
	a.Equal("00:11:22:33:44:01", stats[0].MAC)
	a.Equal("USW Pro 24 PoE", stats[0].Model)
	a.Equal("Default", stats[0].SiteName)

	a.Equal("dev-2", stats[1].DeviceID)
	a.Equal("ap-1", stats[1].Name)
	a.Equal("00:11:22:33:44:02", stats[1].MAC)

	require.Len(t, stats[0].Uplinks, 1)
	require.Len(t, stats[0].Radios, 1)
	a.InDelta(2, stats[0].Uplinks[0].RxRateBps.Val, 0)
}

func TestGetIntegrationDeviceStatsSetsDeviceID(t *testing.T) {
	t.Parallel()

	srv, _ := integrationDeviceServer(t)
	c := integrationDeviceClient(srv)
	site := &IntegrationSite{ID: integrationTestSiteID, InternalReference: "default", Name: "Default"}

	stats, err := c.GetIntegrationDeviceStats(site, "dev-9")
	require.NoError(t, err)

	assert.Equal(t, "dev-9", stats.DeviceID)
	assert.Equal(t, "Default", stats.SiteName)
	assert.InDelta(t, 5, stats.CPUUtilizationPct.Val, 0)
}

func TestGetIntegrationDevicesRequiresAPIKey(t *testing.T) {
	t.Parallel()

	srv, _ := integrationDeviceServer(t)
	c := integrationDeviceClient(srv)
	c.APIKey = ""

	_, err := c.GetIntegrationDevices(&IntegrationSite{ID: integrationTestSiteID, Name: "Default"})
	require.ErrorIs(t, err, ErrAPIKeyRequired)
}
