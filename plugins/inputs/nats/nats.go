package nginx_plus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type Nats struct {
	Urls []string

	client *http.Client

	ResponseTimeout internal.Duration
}

var sampleConfig = `
  ## An array of status URI to gather stats.
  urls = ["http://localhost:4222"]

  # HTTP response timeout (default: 5s)
  response_timeout = "5s"
`

func (n *Nats) SampleConfig() string {
	return sampleConfig
}

func (n *Nats) Description() string {
	return "Uses NATS HTTP stats server"
}

func (n *Nats) Gather(acc telegraf.Accumulator) error {
	var wg sync.WaitGroup

	// Create an HTTP client that is re-used for each
	// collection interval

	if n.client == nil {
		client, err := n.createHttpClient()
		if err != nil {
			return err
		}
		n.client = client
	}

	for _, u := range n.Urls {
		addr, err := url.Parse(u)
		if err != nil {
			acc.AddError(fmt.Errorf("Unable to parse address '%s': %s", u, err))
			continue
		}

		wg.Add(1)
		go func(addr *url.URL) {
			defer wg.Done()
			acc.AddError(n.gatherUrl(addr, acc))
		}(addr)
	}

	wg.Wait()
	return nil
}

func (n *Nats) createHttpClient() (*http.Client, error) {

	if n.ResponseTimeout.Duration < time.Second {
		n.ResponseTimeout.Duration = time.Second * 5
	}

	client := &http.Client{
		Transport: &http.Transport{},
		Timeout:   n.ResponseTimeout.Duration,
	}

	return client, nil
}

func (n *Nats) gatherUrl(addr *url.URL, acc telegraf.Accumulator) error {
	endpoint := fmt.Sprintf("%s%s", addr.String(), "/varz")
	resp, err := n.client.Get(endpoint)

	if err != nil {
		return fmt.Errorf("error making HTTP request to %s: %s", addr.String(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP status %s", addr.String(), resp.Status)
	}
	contentType := strings.Split(resp.Header.Get("Content-Type"), ";")[0]
	switch contentType {
	case "application/json":
		return gatherStatusUrl(bufio.NewReader(resp.Body), getTags(addr), acc)
	default:
		return fmt.Errorf("%s returned unexpected content type %s", addr.String(), contentType)
	}
}

func getTags(addr *url.URL) map[string]string {
	h := addr.Host
	host, port, err := net.SplitHostPort(h)
	if err != nil {
		host = addr.Host
		port = "4222"
	}
	return map[string]string{"server": host, "port": port}
}

type VarStats struct {
	Memory           int64 `json:"mem"`
	CPU              int64 `json:"cpu"`
	Connections      int64 `json:"connections"`
	TotalConnections int64 `json:"total_connections"`
	Routes           int64 `json:"routes"`
	Remotes          int64 `json:"remotes"`
	InMessages       int64 `json:"in_msgs"`
	OutMessages      int64 `json:"out_msgs"`
	InBytes          int64 `json:"in_bytes"`
	OutBytes         int64 `json:"out_bytes"`
	SlowConsumers    int64 `json:"slow_consumers"`
	Subscriptions    int64 `json:"subscritpions"`
}

func gatherStatusUrl(r *bufio.Reader, tags map[string]string, acc telegraf.Accumulator) error {
	dec := json.NewDecoder(r)
	varStats := &VarStats{}
	if err := dec.Decode(varStats); err != nil {
		return fmt.Errorf("Error while decoding JSON response")
	}
	varStats.Gather(tags, acc)
	return nil
}

func (s *VarStats) Gather(tags map[string]string, acc telegraf.Accumulator) {
	acc.AddFields(
		"nats",
		map[string]interface{}{
			"connections":       s.Connections,
			"total_connections": s.TotalConnections,
			"memory":            s.Memory,
			"used_cpu":          s.CPU,
			"routes":            s.Routes,
			"remotes":           s.Remotes,
			"in_messages":       s.InMessages,
			"out_messages":      s.OutMessages,
			"in_bytes":          s.InBytes,
			"out_bytes":         s.OutBytes,
			"slow_consumers":    s.SlowConsumers,
			"subscriptions":     s.Subscriptions,
		},
		tags,
	)
}

func init() {
	inputs.Add("nats", func() telegraf.Input {
		return &Nats{}
	})
}
