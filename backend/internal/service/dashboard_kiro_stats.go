package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

// The private configuration contains admin keys; only aggregate counts leave this service.
type kiroRSStatsServer struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Token string `json:"token"`
}

type kiroRSStats struct {
	mu          sync.Mutex
	servers     []kiroRSStatsServer
	rows        []usagestats.KiroRSAccountCount
	client      *http.Client
	refreshing  bool
	nextRefresh time.Time
}

func newKiroRSStats(cfg *config.Config) *kiroRSStats {
	if cfg == nil || cfg.Dashboard.KiroRSStatsFile == "" {
		return nil
	}
	data, err := os.ReadFile(cfg.Dashboard.KiroRSStatsFile)
	var servers []kiroRSStatsServer
	if err != nil || json.Unmarshal(data, &servers) != nil {
		logger.LegacyPrintf("service.dashboard", "[Dashboard] Cannot load private kiro.rs statistics configuration")
		return nil
	}
	s := &kiroRSStats{client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	seen := map[string]bool{}
	for _, server := range servers {
		server.URL = strings.TrimRight(strings.TrimSpace(server.URL), "/")
		u, err := url.Parse(server.URL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || server.Token == "" || seen[server.URL] {
			continue
		}
		seen[server.URL] = true
		if server.Name == "" {
			server.Name = u.Host
		}
		s.servers = append(s.servers, server)
		s.rows = append(s.rows, usagestats.KiroRSAccountCount{Name: server.Name, Status: "loading"})
	}
	return s
}

func (s *kiroRSStats) snapshot() []usagestats.KiroRSAccountCount {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.refreshing && time.Now().After(s.nextRefresh) {
		s.refreshing = true
		go s.refresh()
	}
	return append([]usagestats.KiroRSAccountCount(nil), s.rows...)
}

func (s *kiroRSStats) refresh() {
	rows := make([]usagestats.KiroRSAccountCount, len(s.servers))
	var wg sync.WaitGroup
	limit := make(chan struct{}, 4)
	for i, server := range s.servers {
		wg.Add(1)
		go func(i int, server kiroRSStatsServer) {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			rows[i] = s.fetch(server)
		}(i, server)
	}
	wg.Wait()
	s.mu.Lock()
	s.rows = rows
	s.nextRefresh = time.Now().Add(30 * time.Second)
	s.refreshing = false
	s.mu.Unlock()
}

func (s *kiroRSStats) fetch(server kiroRSStatsServer) usagestats.KiroRSAccountCount {
	now := time.Now().UTC()
	row := usagestats.KiroRSAccountCount{Name: server.Name, Status: "unavailable", UpdatedAt: &now}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/admin/credentials?page=1&pageSize=1&priority=0&status=enabled&hide=throttled", nil)
	if err != nil {
		return row
	}
	req.Header.Set("x-api-key", server.Token)
	resp, err := s.client.Do(req)
	if err != nil {
		return row
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return row
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024+1))
	if err != nil || len(data) > 1024*1024 {
		return row
	}
	var summary struct {
		FilteredTotal *int64 `json:"filteredTotal"`
		Page          int    `json:"page"`
		PageSize      int    `json:"pageSize"`
	}
	if json.Unmarshal(data, &summary) != nil {
		return row
	}
	// Use the filtered total: enabledPriorityZeroCount also includes cooling accounts.
	if summary.Page != 1 || summary.PageSize != 1 || summary.FilteredTotal == nil || *summary.FilteredTotal < 0 {
		return row
	}
	row.Count, row.Status = summary.FilteredTotal, "ok"
	return row
}
