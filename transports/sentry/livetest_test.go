//go:build livetest

// Live test against a real Sentry project. Compiled only with
// `-tags=livetest` so normal `go test ./...` runs ignore it.
//
// Run:
//
//	SENTRY_DSN=<dsn> go test -tags=livetest -v -run TestLive_Sentry ./transports/sentry/
//
// Optional environment variables:
//
//	SENTRY_ENVIRONMENT  attached to all events (default: livetest)
//	SENTRY_RELEASE      attached as the release tag (default: empty)
//
// To verify in Sentry: open the Logs section of your project and search for
//
//	livetest_id:<id-from-test-output>
package sentrytransport_test

import (
	"cmp"
	"context"
	"os"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"

	"go.loglayer.dev/transport/transporttest"
	sentrytransport "go.loglayer.dev/transports/sentry"
	"go.loglayer.dev/utils/idgen"
)

func TestLive_Sentry_SendsLog(t *testing.T) {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		t.Skip("SENTRY_DSN not set; skipping live Sentry test")
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		EnableLogs:  true,
		Environment: cmp.Or(os.Getenv("SENTRY_ENVIRONMENT"), "livetest"),
		Release:     os.Getenv("SENTRY_RELEASE"),
	}); err != nil {
		t.Fatalf("sentry.Init: %v", err)
	}
	t.Cleanup(func() {
		// Block up to a few seconds so any in-flight events reach Sentry
		// before the test process exits.
		if !sentry.Flush(5 * time.Second) {
			t.Logf("sentry.Flush timed out; some events may not have been sent")
		}
	})

	ctx := context.Background()
	tr := sentrytransport.New(sentrytransport.Config{
		Logger: sentry.NewLogger(ctx),
	})

	ids := transporttest.SendLivetestVariants(tr, idgen.Random(""))

	t.Logf("Sent livetest entries to Sentry (%d variants).", len(ids))
	for i, v := range transporttest.LivetestVariants {
		t.Logf("  %s: filter for livetest_id:%s", v.Name, ids[i])
	}
}
