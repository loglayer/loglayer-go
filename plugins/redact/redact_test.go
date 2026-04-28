package redact_test

import (
	"regexp"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/plugins/plugintest"
	"go.loglayer.dev/plugins/redact"
)

func TestRedact_MetadataMapKey(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Keys: []string{"password"},
	}))

	log.WithMetadata(map[string]any{
		"username": "alice",
		"password": "hunter2",
	}).Info("login")

	line := lib.PopLine()
	m := line.Metadata.(map[string]any)
	if m["password"] != "[REDACTED]" {
		t.Errorf("password should be redacted: got %v", m["password"])
	}
	if m["username"] != "alice" {
		t.Errorf("non-matching keys should pass through: got %v", m["username"])
	}
}

func TestRedact_NestedMap(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Keys: []string{"secret"},
	}))

	log.WithMetadata(map[string]any{
		"user": map[string]any{
			"name":   "alice",
			"secret": "abc",
		},
	}).Info("event")

	line := lib.PopLine()
	m := line.Metadata.(map[string]any)
	user := m["user"].(map[string]any)
	if user["secret"] != "[REDACTED]" {
		t.Errorf("nested secret should be redacted: got %v", user["secret"])
	}
	if user["name"] != "alice" {
		t.Errorf("sibling key should pass through: got %v", user["name"])
	}
}

func TestRedact_FieldsKey(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Keys: []string{"apiKey"},
	}))

	log = log.WithFields(loglayer.Fields{
		"requestId": "abc",
		"apiKey":    "sk_live_xxx",
	})
	log.Info("served")

	line := lib.PopLine()
	if line.Data["apiKey"] != "[REDACTED]" {
		t.Errorf("apiKey field should be redacted: got %v", line.Data["apiKey"])
	}
	if line.Data["requestId"] != "abc" {
		t.Errorf("requestId should pass through: got %v", line.Data["requestId"])
	}
}

func TestRedact_CustomCensor(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Keys:   []string{"k"},
		Censor: "***",
	}))

	log.WithMetadata(map[string]any{"k": "v"}).Info("ok")
	line := lib.PopLine()
	m := line.Metadata.(map[string]any)
	if m["k"] != "***" {
		t.Errorf("custom censor: got %v", m["k"])
	}
}

func TestRedact_PreservesCallerMap(t *testing.T) {
	t.Parallel()
	log, _ := plugintest.Install(t, redact.New(redact.Config{
		Keys: []string{"password"},
	}))

	original := map[string]any{"password": "secret"}
	log.WithMetadata(original).Info("test")

	if original["password"] != "secret" {
		t.Errorf("caller's metadata map should not be mutated: got %v", original["password"])
	}
}

func TestRedact_NonMatchingStructPassesThrough(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Keys: []string{"password"},
	}))

	type evt struct {
		Anything string
	}
	log.WithMetadata(evt{Anything: "raw"}).Info("ok")
	line := lib.PopLine()
	if line.Metadata.(evt).Anything != "raw" {
		t.Errorf("non-matching struct should pass through unchanged: got %v", line.Metadata)
	}
}

func TestRedact_SliceOfMaps(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Keys: []string{"password"},
	}))

	log.WithMetadata(map[string]any{
		"users": []any{
			map[string]any{"name": "alice", "password": "p1"},
			map[string]any{"name": "bob", "password": "p2"},
		},
	}).Info("login")

	line := lib.PopLine()
	users := line.Metadata.(map[string]any)["users"].([]any)
	for i, u := range users {
		m := u.(map[string]any)
		if m["password"] != "[REDACTED]" {
			t.Errorf("users[%d].password should be redacted: got %v", i, m["password"])
		}
	}
}

func TestRedact_TypedSliceOfMaps(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Keys: []string{"secret"},
	}))

	log.WithMetadata(map[string]any{
		"items": []map[string]any{
			{"id": 1, "secret": "x"},
			{"id": 2, "secret": "y"},
		},
	}).Info("ok")

	line := lib.PopLine()
	items := line.Metadata.(map[string]any)["items"].([]map[string]any)
	for i, m := range items {
		if m["secret"] != "[REDACTED]" {
			t.Errorf("items[%d].secret should be redacted: got %v", i, m["secret"])
		}
	}
}

func TestRedact_StructPreservesType(t *testing.T) {
	t.Parallel()
	type user struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Keys: []string{"password"},
	}))

	log.WithMetadata(user{Name: "alice", Password: "hunter2"}).Info("login")

	line := lib.PopLine()
	got, ok := line.Metadata.(user)
	if !ok {
		t.Fatalf("struct type should be preserved: got %T", line.Metadata)
	}
	if got.Password != "[REDACTED]" {
		t.Errorf("struct password field should be redacted: got %v", got.Password)
	}
	if got.Name != "alice" {
		t.Errorf("struct name field should pass through: got %v", got.Name)
	}
}

func TestRedact_NestedStruct(t *testing.T) {
	t.Parallel()
	type creds struct {
		APIKey string `json:"apiKey"`
	}
	type req struct {
		User  string `json:"user"`
		Creds creds  `json:"creds"`
	}

	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Keys: []string{"apiKey"},
	}))

	log.WithMetadata(req{User: "alice", Creds: creds{APIKey: "sk_live"}}).Info("call")
	line := lib.PopLine()
	got := line.Metadata.(req)
	if got.Creds.APIKey != "[REDACTED]" {
		t.Errorf("nested struct field should be redacted: got %v", got.Creds.APIKey)
	}
}

func TestRedact_PointerToStructPreservesPointer(t *testing.T) {
	t.Parallel()
	type user struct {
		Password string `json:"password"`
	}
	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Keys: []string{"password"},
	}))

	in := &user{Password: "hunter2"}
	log.WithMetadata(in).Info("call")
	line := lib.PopLine()
	got, ok := line.Metadata.(*user)
	if !ok {
		t.Fatalf("pointer type should be preserved: got %T", line.Metadata)
	}
	if got.Password != "[REDACTED]" {
		t.Errorf("pointer struct field should be redacted: got %v", got.Password)
	}
	if in.Password != "hunter2" {
		t.Errorf("caller's struct should not be mutated: got %v", in.Password)
	}
}

func TestRedact_PatternsMatchValues(t *testing.T) {
	t.Parallel()
	ssn := regexp.MustCompile(`^\d{3}-\d{2}-\d{4}$`)
	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Patterns: []*regexp.Regexp{ssn},
	}))

	log.WithMetadata(map[string]any{
		"note": "user provided 123-45-6789 in a free-form field",
		"id":   "123-45-6789",
	}).Info("intake")

	line := lib.PopLine()
	m := line.Metadata.(map[string]any)
	if m["id"] != "[REDACTED]" {
		t.Errorf("ssn-shaped value should be redacted: got %v", m["id"])
	}
	if m["note"] != "user provided 123-45-6789 in a free-form field" {
		t.Errorf("partial match (full-anchor regex) should not match note: got %v", m["note"])
	}
}

func TestRedact_PatternsRecurseIntoSlices(t *testing.T) {
	t.Parallel()
	cc := regexp.MustCompile(`^\d{16}$`)
	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Patterns: []*regexp.Regexp{cc},
	}))

	log.WithMetadata(map[string]any{
		"cards": []any{"4111111111111111", "not-a-card"},
	}).Info("intake")

	line := lib.PopLine()
	cards := line.Metadata.(map[string]any)["cards"].([]any)
	if cards[0] != "[REDACTED]" {
		t.Errorf("matching slice element should be redacted: got %v", cards[0])
	}
	if cards[1] != "not-a-card" {
		t.Errorf("non-matching slice element should pass through: got %v", cards[1])
	}
}

func TestRedact_KeysAndPatternsCombined(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Keys:     []string{"password"},
		Patterns: []*regexp.Regexp{regexp.MustCompile(`^token-`)},
	}))

	log.WithMetadata(map[string]any{
		"password": "p",
		"session":  "token-abc",
		"username": "alice",
	}).Info("ok")

	line := lib.PopLine()
	m := line.Metadata.(map[string]any)
	if m["password"] != "[REDACTED]" {
		t.Errorf("key match: got %v", m["password"])
	}
	if m["session"] != "[REDACTED]" {
		t.Errorf("pattern match: got %v", m["session"])
	}
	if m["username"] != "alice" {
		t.Errorf("untouched: got %v", m["username"])
	}
}

func TestRedact_DefaultCensor(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, redact.New(redact.Config{
		Keys: []string{"k"},
	}))
	log.WithMetadata(map[string]any{"k": "v"}).Info("ok")
	line := lib.PopLine()
	m := line.Metadata.(map[string]any)
	if m["k"] != "[REDACTED]" {
		t.Errorf("default censor should be [REDACTED]: got %v", m["k"])
	}
}

func TestRedact_DefaultID(t *testing.T) {
	t.Parallel()
	p := redact.New(redact.Config{Keys: []string{"k"}})
	if p.ID != "redact" {
		t.Errorf("default ID: got %q, want \"redact\"", p.ID)
	}
}
