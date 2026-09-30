package unifi_test

import (
	"encoding/json"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unpoller/unifi/v6"
)

func TestIntegrationDeviceStats(t *testing.T) {
	t.Parallel()

	var s unifi.IntegrationDeviceStats

	require.NoError(t, gofakeit.Struct(&s))
}

func TestIntegrationDeviceRadioStats(t *testing.T) {
	t.Parallel()

	var s unifi.IntegrationDeviceRadioStats

	require.NoError(t, gofakeit.Struct(&s))
}

func TestIntegrationDeviceUplinkStats(t *testing.T) {
	t.Parallel()

	var s unifi.IntegrationDeviceUplinkStats

	require.NoError(t, gofakeit.Struct(&s))
}

// Payload observed on a Network 10.6 console: no deviceId, a single "uplink"
// object and the radios nested under "interfaces".
const integrationDeviceStatsNetworkBody = `{
  "uptimeSec": 435380,
  "lastHeartbeatAt": "2026-09-28T09:47:00Z",
  "nextHeartbeatAt": "2026-09-28T09:48:00Z",
  "loadAverage1Min": 3.21,
  "loadAverage5Min": 2.97,
  "loadAverage15Min": 2.92,
  "cpuUtilizationPct": 61.2,
  "memoryUtilizationPct": 35.5,
  "uplink": {"txRateBps": 1000000000, "rxRateBps": 1000000000},
  "interfaces": {"radios": [
    {"frequencyGHz": 2.4, "txRetriesPct": 1.5},
    {"frequencyGHz": 5, "txRetriesPct": 0.2},
    {"frequencyGHz": 6, "txRetriesPct": 0}
  ]}
}`

func TestIntegrationDeviceStatsUnmarshalsNetworkPayload(t *testing.T) {
	t.Parallel()

	var s unifi.IntegrationDeviceStats

	require.NoError(t, json.Unmarshal([]byte(integrationDeviceStatsNetworkBody), &s))

	a := assert.New(t)
	a.InDelta(61.2, s.CPUUtilizationPct.Val, 0.001)
	a.InDelta(35.5, s.MemoryUtilizationPct.Val, 0.001)
	a.InDelta(2.92, s.LoadAverage15Min.Val, 0.001)
	a.InDelta(435380, s.UptimeSec.Val, 0)
	a.Empty(s.DeviceID, "the payload carries no device ID; the caller sets it")

	require.Len(t, s.Uplinks, 1, "the single uplink object is exposed as a one-element slice")
	a.InDelta(1000000000, s.Uplinks[0].RxRateBps.Val, 0)
	a.InDelta(1000000000, s.Uplinks[0].TxRateBps.Val, 0)

	require.Len(t, s.Radios, 3, "radios are read from interfaces.radios")
	a.Equal("2.4", s.Radios[0].FrequencyGHz.Txt)
	a.InDelta(1.5, s.Radios[0].TxRetriesPct.Val, 0.001)
	a.Equal("6", s.Radios[2].FrequencyGHz.Txt)
}

// The flat shape the struct exposes must keep decoding, so a firmware that
// answers with "uplinks"/"radios" arrays is not broken by the nested mapping.
func TestIntegrationDeviceStatsUnmarshalsFlatArrays(t *testing.T) {
	t.Parallel()

	body := `{"cpuUtilizationPct": 1,
		"uplinks": [{"rxRateBps": 1, "txRateBps": 2}, {"rxRateBps": 3, "txRateBps": 4}],
		"radios": [{"frequencyGHz": 2.4, "txRetriesPct": 9}]}`

	var s unifi.IntegrationDeviceStats

	require.NoError(t, json.Unmarshal([]byte(body), &s))
	assert.Len(t, s.Uplinks, 2)
	assert.Len(t, s.Radios, 1)
	assert.InDelta(t, 4, s.Uplinks[1].TxRateBps.Val, 0)
}

func TestIntegrationDevice(t *testing.T) {
	t.Parallel()

	var d unifi.IntegrationDevice

	require.NoError(t, gofakeit.Struct(&d))
}
