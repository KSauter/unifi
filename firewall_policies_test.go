package unifi_test

import (
	"encoding/json"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unpoller/unifi/v6"
)

func TestFirewallPolicy(t *testing.T) {
	t.Parallel()

	var p unifi.FirewallPolicy

	require.NoError(t, gofakeit.Struct(&p))
}

// Trimmed from a Network 10.6 console: the v2 firewall-policies endpoint
// reports a cumulative hit counter and the time of the last match per policy.
// Policies that never matched carry null in both fields.
func TestFirewallPolicyUnmarshalsHitCounters(t *testing.T) {
	t.Parallel()

	body := `[
		{"_id": "0123456789abcdef01234567", "name": "Allow Return Traffic", "action": "ALLOW", "enabled": true,
		 "index": 10000, "hits": 1104968290, "last_hit": 1790155251444},
		{"_id": "0123456789abcdef01234569", "name": "lan-to-management", "action": "ALLOW", "enabled": true,
		 "index": 10000, "hits": null, "last_hit": null}
	]`

	var policies []*unifi.FirewallPolicy

	require.NoError(t, json.Unmarshal([]byte(body), &policies))
	require.Len(t, policies, 2)

	a := assert.New(t)
	a.InDelta(1104968290, policies[0].Hits.Val, 0)
	a.InDelta(1790155251444, policies[0].LastHit.Val, 0)
	a.Equal("1104968290", policies[0].Hits.Txt)

	a.InDelta(0, policies[1].Hits.Val, 0, "null hits decode as zero")
	a.InDelta(0, policies[1].LastHit.Val, 0, "null last_hit decodes as zero")
}
