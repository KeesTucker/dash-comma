package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/KeesTucker/dash-comma/internal/config"
	"github.com/KeesTucker/dash-comma/internal/stats"
)

func testServer(t *testing.T, env map[string]string) http.Handler {
	t.Helper()
	cfg := config.From(func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	})
	s := &Server{cfg: cfg, hw: stats.EmptyHardware(), staticSubs: map[int]func(stats.ServerInfo){}}
	s.cpu = newFeed("CPU", true, false, 3, 1000, func() (stats.CPULoad, error) { return stats.CPULoad{{Core: 0, Load: 12.5}}, nil })
	s.ram = newFeed("RAM", true, false, 3, 1000, func() (stats.RAMLoad, error) { return 42, nil })
	s.storage = newFeed("Storage", true, false, 1, 1000, func() (stats.StorageLoad, error) { return stats.StorageLoad{1, -1}, nil })
	s.network = newFeed("Network", true, false, 3, 1000, func() (stats.NetworkLoad, error) { return stats.NetworkLoad{}, errors.New("down") })
	s.gpu = newFeed("GPU", false, false, 3, 1000, func() (stats.GPULoad, error) { return stats.GPULoad{Layout: []stats.GPUUsage{}}, nil })
	web := fstest.MapFS{
		"index.html":    {Data: []byte("<html></html>")},
		"assets/app.js": {Data: []byte("x")},
	}
	return s.Handler(web)
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return w
}

func TestRoutes(t *testing.T) {
	h := testServer(t, nil)
	cases := map[string]string{
		"/load/cpu":     `[{"core":0,"load":12.5,"temp":0}]`,
		"/load/ram":     `{"load":42}`,
		"/load/storage": `[1,-1]`,
		"/load/gpu":     `{"layout":[]}`,
	}
	for path, want := range cases {
		w := get(t, h, path)
		if w.Code != http.StatusOK || w.Body.String() != want+"\n" {
			t.Errorf("%s: %d %q, want %q", path, w.Code, w.Body.String(), want)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("%s: no CORS header", path)
		}
	}
	// A failed reading is an empty 200, as dashdot's res.send(undefined).
	if w := get(t, h, "/load/network"); w.Code != http.StatusOK || w.Body.Len() != 0 {
		t.Errorf("/load/network: %d %q", w.Code, w.Body.String())
	}

	var info map[string]any
	if err := json.Unmarshal(get(t, h, "/info").Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if _, ok := info["config"]; ok {
		t.Error("/info must not carry the config")
	}
	var cfg map[string]any
	if err := json.Unmarshal(get(t, h, "/config").Body.Bytes(), &cfg); err != nil || cfg["config"] == nil {
		t.Errorf("/config: %v %v", cfg, err)
	}
}

func TestStaticFiles(t *testing.T) {
	h := testServer(t, nil)
	if w := get(t, h, "/"); w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "public, max-age=0" {
		t.Errorf("/: %d %q", w.Code, w.Header().Get("Cache-Control"))
	}
	if w := get(t, h, "/assets/app.js"); w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "public, max-age=31536000" {
		t.Errorf("asset: %d %q", w.Code, w.Header().Get("Cache-Control"))
	}
	if w := get(t, h, "/assets/"); w.Code != http.StatusNotFound {
		t.Errorf("directory listing: %d", w.Code)
	}
}

func TestRoutingPathAndIntegrations(t *testing.T) {
	h := testServer(t, map[string]string{"DASHDOT_ROUTING_PATH": "/dash/", "DASHDOT_DISABLE_INTEGRATIONS": "true"})
	if w := get(t, h, "/dash/"); w.Code != http.StatusOK {
		t.Errorf("/dash/: %d", w.Code)
	}
	if w := get(t, h, "/dash/load/cpu"); w.Code == http.StatusOK {
		t.Error("integration routes should be off")
	}
	if w := get(t, h, "/dash/"); w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS should be off with integrations disabled")
	}
	if w := get(t, h, "/dash/socket/?EIO=4&transport=polling"); w.Code != http.StatusOK {
		t.Errorf("socket handshake under the routing path: %d %s", w.Code, w.Body.String())
	}
}

func TestFeedReplaysAndStops(t *testing.T) {
	n := 0
	f := newFeed("test", true, false, 2, 100, func() (int, error) { n++; return n, nil })
	time.Sleep(10 * time.Millisecond)
	if n != 0 {
		t.Fatal("a feed that isn't in the background shouldn't poll before a subscriber")
	}
	got := make(chan int, 10)
	stop := f.subscribe(func(v int) { got <- v })
	<-got
	<-got
	<-got

	// A late subscriber gets the last two readings straight away.
	late := make(chan int, 10)
	stopLate := f.subscribe(func(v int) { late <- v })
	a, b := <-late, <-late
	if b != a+1 {
		t.Fatalf("replay out of order: %d, %d", a, b)
	}
	stop()
	stopLate()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.stop != nil || f.buf != nil {
		t.Fatal("the feed should stop and clear with no subscribers")
	}
}
