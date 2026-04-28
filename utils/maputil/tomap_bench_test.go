package maputil

import "testing"

type benchUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var benchTestUser = benchUser{ID: 42, Name: "Alice", Email: "alice@example.com"}

func BenchmarkToMap_Struct(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ToMap(benchTestUser)
	}
}

func BenchmarkToMap_Map(b *testing.B) {
	m := map[string]any{"id": 42, "name": "Alice", "email": "alice@example.com"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ToMap(m)
	}
}
