//go:build livetest

// Live test against the real Axiom API. Compiled only with `-tags=livetest`
// so normal `go test ./...` runs ignore it.
//
// Run:
//
//	AXIOM_TOKEN=<token> AXIOM_DATASET=<dataset> go test -tags=livetest -v -run TestLive_Axiom ./transports/axiom/
//
// The test ingests a variety of log entries to the specified dataset. Check
// Axiom's Logs Explorer to verify they arrived (indexing typically takes
// 5-60 seconds).

package axiom

import (
	"os"
	"testing"

	"github.com/axiomhq/axiom-go/axiom"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/transporttest"
)

func TestLive_Axiom_SendsLog(t *testing.T) {
	token := os.Getenv("AXIOM_TOKEN")
	if token == "" {
		t.Skip("AXIOM_TOKEN not set; skipping live Axiom test")
	}

	dataset := os.Getenv("AXIOM_DATASET")
	if dataset == "" {
		t.Skip("AXIOM_DATASET not set; skipping live Axiom test")
	}

	client, err := axiom.NewClient(
		axiom.SetAPITokenConfig(token),
	)
	if err != nil {
		t.Fatalf("Failed to create Axiom client: %v", err)
	}

	tr, err := Build(Config{
		BaseConfig:  transport.BaseConfig{ID: "axiom"},
		Client:      client,
		DatasetName: dataset,
	})
	if err != nil {
		t.Fatalf("Failed to build transport: %v", err)
	}

	baseID := "livetest-" + t.Name()
	sentIDs := transporttest.SendLivetestVariants(tr, baseID)

	for i, v := range transporttest.LivetestVariants {
		t.Logf("  %s: livetest_id:%s", v.Name, sentIDs[i])
	}

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	t.Logf("Sent livetest entries to Axiom dataset %q.", dataset)
}
