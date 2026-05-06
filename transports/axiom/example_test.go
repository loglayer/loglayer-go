package axiom

import (
	"go.loglayer.dev/v2"
)

// ExampleNew shows a basic usage pattern with the Axiom transport. The client
// is constructed via axiom.NewClient() which reads configuration from
// environment variables (AXIOM_TOKEN, AXIOM_ORG_ID).
func ExampleNew() {
	// axiomClient would be created here in real code:
	//   client, err := axiom.NewClient()
	var axiomClient any
	_ = axiomClient

	t, err := Build(Config{
		Client:      nil,
		DatasetName: "my-logs",
	})
	if err != nil {
		return
	}

	_ = loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
}

// ExampleConfig_MessageField shows how to customize the message field.
func ExampleConfig_MessageField() {
	t, err := Build(Config{
		Client:       nil,
		DatasetName:  "my-logs",
		MessageField: "message",
	})
	if err != nil {
		return
	}

	_ = loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
}
