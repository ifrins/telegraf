package nginx_plus

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/influxdata/telegraf/testutil"
	//"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleStatusResponse = `
{
  "server_id": "fa7c7I3EnmTlpxWlTJfWP8",
  "version": "1.0.4",
  "go": "go1.9",
  "host": "0.0.0.0",
  "auth_required": false,
  "ssl_required": false,
  "tls_required": false,
  "tls_verify": false,
  "addr": "0.0.0.0",
  "max_connections": 65536,
  "ping_interval": 120000000000,
  "ping_max": 2,
  "http_host": "0.0.0.0",
  "http_port": 8222,
  "https_port": 0,
  "auth_timeout": 1,
  "max_control_line": 1024,
  "cluster": {
    "addr": "0.0.0.0",
    "cluster_port": 0,
    "auth_timeout": 1
  },
  "tls_timeout": 0.5,
  "port": 4222,
  "max_payload": 1048576,
  "start": "2017-11-28T09:45:32.564931772+01:00",
  "now": "2017-11-28T09:46:54.970636678+01:00",
  "uptime": "1m22s",
  "mem": 6193152,
  "cores": 8,
  "cpu": 0,
  "connections": 2,
  "total_connections": 2,
  "routes": 0,
  "remotes": 0,
  "in_msgs": 0,
  "out_msgs": 0,
  "in_bytes": 0,
  "out_bytes": 0,
  "slow_consumers": 0,
  "subscriptions": 25,
  "http_req_stats": {
    "/": 1,
    "/connz": 1,
    "/routez": 1,
    "/subsz": 1,
    "/varz": 2
  },
  "config_load_time": "2017-11-28T09:45:32.564931772+01:00"
}
`

func TestNginxPlusGeneratesMetrics(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rsp string

		if r.URL.Path == "/varz" {
			rsp = sampleStatusResponse
			w.Header()["Content-Type"] = []string{"application/json"}
		} else {
			panic("Cannot handle request")
		}

		fmt.Fprintln(w, rsp)
	}))
	defer ts.Close()

	n := &Nats{
		Urls: []string{fmt.Sprintf("%s", ts.URL)},
	}

	var acc testutil.Accumulator

	err_nginx := n.Gather(&acc)

	require.NoError(t, err_nginx)

	addr, err := url.Parse(ts.URL)
	if err != nil {
		panic(err)
	}

	host, port, err := net.SplitHostPort(addr.Host)
	if err != nil {
		host = addr.Host
		port = "4222"
	}

	acc.AssertContainsTaggedFields(
		t,
		"nats",
		map[string]interface{}{
			"connections":       int(2),
			"total_connections": int(2),
			"memory":            int(6193152),
			"used_cpu":          int(0),
			"routes":            int(0),
			"remotes":           int(0),
			"in_messages":       int(0),
			"out_messages":      int(0),
			"in_bytes":          int(0),
			"out_bytes":         int(0),
			"slow_consumers":    int(0),
			"subscriptions":     int(25),
		},
		map[string]string{
			"server": host,
			"port":   port,
		})
}
