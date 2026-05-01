//go:build livetest

// Live test against a real Google Cloud Logging project. Compiled only
// with `-tags=livetest` so normal `go test ./...` runs ignore it.
//
// Run:
//
//	GOOGLE_CLOUD_PLATFORM_PROJECT_ID=<project-id> \
//	  GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json \
//	  go test -tags=livetest -v -run TestLive_GCPLogging ./transports/gcplogging/
//
// Optional environment variables:
//
//	GCP_LOG_NAME      log name to write to (default: loglayer-livetest)
//
// To verify in Google Cloud Logging: open the Logs Explorer for your
// project and filter by:
//
//	logName="projects/<project-id>/logs/<log-name>"
//	jsonPayload.livetest_id="<id-from-test-output>"
package gcplogging_test

import (
	"cmp"
	"context"
	"os"
	"testing"

	"cloud.google.com/go/logging"

	"go.loglayer.dev/transport/transporttest"
	"go.loglayer.dev/transports/gcplogging"
	"go.loglayer.dev/utils/idgen"
)

func TestLive_GCPLogging_SendsLog(t *testing.T) {
	projectID := os.Getenv("GOOGLE_CLOUD_PLATFORM_PROJECT_ID")
	if projectID == "" {
		t.Skip("GOOGLE_CLOUD_PLATFORM_PROJECT_ID not set; skipping live GCP Logging test")
	}

	ctx := context.Background()
	client, err := logging.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("logging.NewClient: %v", err)
	}
	t.Cleanup(func() {
		// client.Close flushes every Logger it created, so any entries
		// still in the bundler queue land before the process exits.
		if err := client.Close(); err != nil {
			t.Logf("client.Close: %v", err)
		}
	})

	logName := cmp.Or(os.Getenv("GCP_LOG_NAME"), "loglayer-livetest")
	gcpLogger := client.Logger(logName)

	tr := gcplogging.New(gcplogging.Config{
		Logger: gcpLogger,
	})
	t.Cleanup(func() {
		// Transport.Close calls Logger.Flush. Registered after the
		// client cleanup so it runs FIRST (LIFO), draining the bundler
		// before the client tears down.
		if err := tr.Close(); err != nil {
			t.Logf("tr.Close: %v", err)
		}
	})

	ids := transporttest.SendLivetestVariants(tr, idgen.Random(""))

	t.Logf("Sent livetest entries to GCP Logging (%d variants).", len(ids))
	t.Logf("  Project: %s", projectID)
	t.Logf("  Log:     %s", logName)
	for i, v := range transporttest.LivetestVariants {
		t.Logf("  %s: filter for jsonPayload.livetest_id=%q", v.Name, ids[i])
	}
}
