package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func BenchmarkWriteJSON_SmallPayload(b *testing.B) {
	status := models.Status{Code: models.StatusOK, Message: "instance is running"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		_ = WriteJSON(w, http.StatusOK, status)
	}
}

func BenchmarkWriteJSON_LargePayload(b *testing.B) {
	clones := make([]*models.Clone, 100)
	for i := range clones {
		clones[i] = &models.Clone{
			ID:        "clone-" + string(rune('a'+i%26)),
			CreatedAt: &models.LocalTime{Time: time.Now()},
			Status:    models.Status{Code: models.StatusOK, Message: "clone is running"},
			DB:        models.Database{DBName: "testdb", Username: "user"},
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		_ = WriteJSON(w, http.StatusOK, clones)
	}
}

func BenchmarkReadJSON(b *testing.B) {
	body := `{"id":"bench-clone","snapshot":{"id":"snap-1"},"db":{"username":"test","password":"test"}}`

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/clone", strings.NewReader(body))
		var result map[string]interface{}
		_ = ReadJSON(req, &result)
	}
}

func BenchmarkWriteData(b *testing.B) {
	data := []byte(`{"status":"ok","engine":{"version":"3.5.0"},"cloning":{"numClones":42}}`)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		_ = WriteData(w, http.StatusOK, data)
	}
}
