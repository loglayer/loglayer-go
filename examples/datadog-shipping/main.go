// Datadog Logs intake setup with batching tuned for production. Falls back
// to a console transport when DD_API_KEY is not set so the example is
// runnable without credentials.
//
// Run (without Datadog):
//
//	go run ./examples/datadog-shipping
//
// Run (with real Datadog shipping):
//
//	DD_API_KEY=<your-key> DD_SITE=us1 go run ./examples/datadog-shipping
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"go.loglayer.dev/transports/datadog/v2"
	httptr "go.loglayer.dev/transports/http/v2"
	"go.loglayer.dev/transports/pretty/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

func main() {
	apiKey := os.Getenv("DD_API_KEY")

	tr := buildTransport(apiKey)

	log := loglayer.New(loglayer.Config{Transport: tr})
	log = log.WithFields(loglayer.Fields{
		"service": "checkout-api",
		"version": "1.2.3",
	})

	log.Info("startup complete")
	log.WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served request")
	log.WithMetadata(loglayer.Metadata{"retry": 2}).Warn("upstream slow")

	// Always close on shutdown so the in-flight batch flushes.
	if closer, ok := tr.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}

func buildTransport(apiKey string) loglayer.Transport {
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "DD_API_KEY not set; using pretty transport for demo")
		return pretty.New(pretty.Config{
			BaseConfig: transport.BaseConfig{ID: "pretty"},
		})
	}

	return datadog.New(datadog.Config{
		BaseConfig: transport.BaseConfig{ID: "datadog"},
		APIKey:     apiKey,
		Site:       datadog.Site(envOr("DD_SITE", "us1")),
		Source:     "go",
		Service:    "checkout-api",
		Hostname:   hostnameOrEmpty(),
		Tags:       envOr("DD_TAGS", "env:demo"),
		HTTP: httptr.Config{
			BatchSize:     500,
			BatchInterval: 2 * time.Second,
			Client:        &http.Client{Timeout: 10 * time.Second},
			OnError: func(err error, entries []httptr.Entry) {
				fmt.Fprintf(os.Stderr, "datadog send failed (%d entries): %v\n", len(entries), err)
			},
		},
	})
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func hostnameOrEmpty() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}
