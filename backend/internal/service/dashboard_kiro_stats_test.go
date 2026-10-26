package service

import (
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestKiroRSStatsSummary(t *testing.T) {
	for _, tc := range []struct {
		name, body, status string
		count              int64
	}{
		{"zero", `{"filteredTotal":0,"page":1,"pageSize":1}`, "ok", 0},
		{"excludes cooling", `{"enabledPriorityZeroCount":42,"filteredTotal":30,"page":1,"pageSize":1}`, "ok", 30},
		{"legacy", `{"filteredTotal":5,"page":1,"pageSize":1}`, "ok", 5},
		{"missing", `{"total":500,"credentials":[]}`, "unavailable", 0},
		{"unfiltered summary", `{"enabledPriorityZeroCount":42}`, "unavailable", 0},
		{"incomplete", `{"filteredTotal":5}`, "unavailable", 0},
		{"negative", `{"filteredTotal":-1,"page":1,"pageSize":1}`, "unavailable", 0},
		{"invalid", `not json`, "unavailable", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("x-api-key") != "private-secret" || r.URL.Query().Get("priority") != "0" || r.URL.Query().Get("status") != "enabled" || r.URL.Query().Get("pageSize") != "1" || r.URL.Query().Get("hide") != "throttled" {
					t.Error("missing authentication or filters")
				}
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			s := &kiroRSStats{client: srv.Client()}
			row := s.fetch(kiroRSStatsServer{Name: "instance", URL: srv.URL, Token: "private-secret"})
			if row.Status != tc.status {
				t.Fatalf("status: %s", row.Status)
			}
			if tc.status == "ok" && (row.Count == nil || *row.Count != tc.count) {
				t.Fatalf("count: %v", row.Count)
			}
			if tc.status != "ok" && row.Count != nil {
				t.Fatal("failure must not appear as zero")
			}
			data, _ := json.Marshal(row)
			if strings.Contains(string(data), "private-secret") || strings.Contains(string(data), srv.URL) {
				t.Fatal("private configuration leaked")
			}
		})
	}
}

func TestKiroRSStatsDedupCacheAndFailure(t *testing.T) {
	requests := 0
	fail := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if fail {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"filteredTotal":12,"page":1,"pageSize":1}`))
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "stats.json")
	data, _ := json.Marshal([]kiroRSStatsServer{{Name: "first", URL: srv.URL, Token: "secret"}, {Name: "duplicate", URL: srv.URL + "/", Token: "secret"}})
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	cfg.Dashboard.KiroRSStatsFile = path
	s := newKiroRSStats(cfg)
	if len(s.servers) != 1 {
		t.Fatal("endpoint not deduplicated")
	}
	s.refresh()
	rows := s.snapshot()
	if requests != 1 || rows[0].Count == nil || *rows[0].Count != 12 {
		t.Fatal("unexpected cached result")
	}
	rows[0].Name = "mutated"
	if s.snapshot()[0].Name != "first" {
		t.Fatal("snapshot mutates cache")
	}
	fail = true
	s.refresh()
	rows = s.snapshot()
	if requests != 2 || rows[0].Count != nil || rows[0].Status != "unavailable" {
		t.Fatal("failed refresh retained old count")
	}
	if s.nextRefresh.Before(time.Now()) {
		t.Fatal("missing refresh backoff")
	}
}
