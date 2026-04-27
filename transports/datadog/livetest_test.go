//go:build livetest

// Live test against the real Datadog Logs intake. Compiled only with
// `-tags=livetest` so normal `go test ./...` runs ignore it.
//
// Run:
//
//	DD_API_KEY=<key> go test -tags=livetest -v -run TestLive_Datadog ./transports/datadog/
//
// Optional environment variables:
//
//	DD_SITE      one of us1 (default), us3, us5, eu1, ap1
//	DD_SERVICE   service tag attached to the test entries (default: loglayer-go-livetest)
//	DD_HOSTNAME  hostname tag (default: empty)
//	DD_TAGS      comma-separated ddtags (default: env:livetest)
//	DD_SOURCE    ddsource (default: go-loglayer-livetest)
//
// To verify in Datadog: open the Logs Explorer and search for
//
//	source:go-loglayer-livetest @livetest_id:<id-from-test-output>
//
// Indexing typically takes 5-60 seconds.

package datadog_test

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"go.loglayer.dev"
	"go.loglayer.dev/transports/datadog"
	httptr "go.loglayer.dev/transports/http"
)

func TestLive_Datadog_SendsLog(t *testing.T) {
	apiKey := os.Getenv("DD_API_KEY")
	if apiKey == "" {
		t.Skip("DD_API_KEY not set; skipping live Datadog test")
	}

	site := datadog.Site(os.Getenv("DD_SITE"))
	source := envOr("DD_SOURCE", "go-loglayer-livetest")
	service := envOr("DD_SERVICE", "loglayer-go-livetest")
	hostname := os.Getenv("DD_HOSTNAME")
	tags := envOr("DD_TAGS", "env:livetest")
	livetestID := randomID()

	var (
		errMu    sync.Mutex
		sendErrs []error
		errCount int
	)
	tr := datadog.New(datadog.Config{
		APIKey:   apiKey,
		Site:     site,
		Source:   source,
		Service:  service,
		Hostname: hostname,
		Tags:     tags,
		HTTP: httptr.Config{
			// Flush immediately so any error from the API surfaces during
			// this test instead of after Close() in the worker shutdown path.
			BatchSize:     1,
			BatchInterval: 500 * time.Millisecond,
			OnError: func(err error, entries []httptr.Entry) {
				errMu.Lock()
				defer errMu.Unlock()
				errCount++
				sendErrs = append(sendErrs, err)
			},
		},
	})

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log = log.WithFields(loglayer.Fields{
		"livetest_id": livetestID,
		"sent_at":     time.Now().UTC().Format(time.RFC3339Nano),
	})

	log.Info("loglayer-go datadog livetest")
	log.WithMetadata(loglayer.Metadata{
		"step":     "with-metadata",
		"runner":   "go test",
		"checksum": livetestID,
	}).Warn("loglayer-go datadog livetest with metadata")

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	errMu.Lock()
	defer errMu.Unlock()
	if errCount > 0 {
		for _, e := range sendErrs {
			t.Logf("send error: %v", e)
			var httpErr *httptr.HTTPError
			if errors.As(e, &httpErr) {
				switch httpErr.StatusCode {
				case 401, 403:
					t.Errorf("authentication failed (status %d) — check DD_API_KEY and DD_SITE", httpErr.StatusCode)
				}
			}
		}
		t.Fatalf("Datadog intake reported %d error(s); see logs above", errCount)
	}

	t.Logf("Sent livetest entries to Datadog (%s).", site.IntakeURL())
	t.Logf("Verify in Datadog Logs Explorer: source:%s @livetest_id:%s", source, livetestID)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func randomID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
