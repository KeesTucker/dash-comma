// Package server serves dashdot's API (the /config, /info and /load/* routes
// and the socket.io events) and the web UI.
package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/KeesTucker/dash-comma/internal/config"
	"github.com/KeesTucker/dash-comma/internal/stats"
	"github.com/klauspost/compress/gzhttp"
	"github.com/robfig/cron/v3"
	"github.com/zishang520/socket.io/servers/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
)

type Server struct {
	cfg *config.Config

	mu         sync.Mutex
	hw         stats.HardwareInfo
	staticSubs map[int]func(stats.ServerInfo)
	nextID     int

	cpu     *feed[stats.CPULoad]
	ram     *feed[stats.RAMLoad]
	storage *feed[stats.StorageLoad]
	network *feed[stats.NetworkLoad]
	gpu     *feed[stats.GPULoad]
}

// New reads the static hardware info for each enabled widget and starts the
// feeds.
func New(cfg *config.Config) *Server {
	s := &Server{cfg: cfg, hw: stats.EmptyHardware(), staticSubs: map[int]func(stats.ServerInfo){}}
	bg := !cfg.DisableBackgroundStatsCollection

	if cfg.Widget("os") {
		s.hw.OS, _ = stats.OSStatic()
	}
	if cfg.Widget("cpu") {
		if info, err := stats.CPUStatic(); err == nil {
			s.hw.CPU = info
		} else {
			log.Printf("cpu: %v", err)
		}
	}
	if cfg.Widget("ram") {
		if info, err := stats.RAMStatic(); err == nil {
			s.hw.RAM = info
		} else {
			log.Printf("ram: %v", err)
		}
	}
	if cfg.Widget("storage") {
		s.hw.Storage, _ = stats.StorageStatic(cfg)
	}
	net := stats.NewNetwork(cfg.UseNetworkInterface)
	if cfg.Widget("network") {
		s.hw.Network = net.Static()
	}
	gpus := &stats.GPUs{}
	if cfg.Widget("gpu") {
		gpus = stats.NewGPUs(cfg)
		s.hw.GPU = gpus.Static()
	}
	b, _ := json.MarshalIndent(s.hw, "", "  ")
	log.Printf("Static Server Info: %s", b)

	cpu := stats.NewCPUSampler(cfg.EnableCPUTemps)
	layout := s.hw.Storage
	s.cpu = newFeed("CPU", cfg.Widget("cpu"), bg, cfg.CPUShownDatapoints, cfg.CPUPollInterval, cpu.Sample)
	s.ram = newFeed("RAM", cfg.Widget("ram"), bg, cfg.RAMShownDatapoints, cfg.RAMPollInterval, stats.RAMLoadNow)
	s.storage = newFeed("Storage", cfg.Widget("storage"), bg, 1, cfg.StoragePollInterval, func() (stats.StorageLoad, error) {
		return stats.StorageLoadNow(cfg, layout)
	})
	s.network = newFeed("Network", cfg.Widget("network"), bg, cfg.NetworkShownDatapoints, cfg.NetworkPollInterval, net.Sample)
	s.gpu = newFeed("GPU", cfg.Widget("gpu"), bg, cfg.GPUShownDatapoints, cfg.GPUPollInterval, gpus.Sample)
	return s
}

func (s *Server) hardware() stats.HardwareInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	hw := s.hw
	hw.OS.Uptime = stats.Uptime()
	return hw
}

func (s *Server) serverInfo() stats.ServerInfo {
	return stats.ServerInfo{HardwareInfo: s.hardware(), Config: s.cfg}
}

func (s *Server) subscribeStatic(fn func(stats.ServerInfo)) func() {
	s.mu.Lock()
	id := s.nextID
	s.nextID++
	s.staticSubs[id] = fn
	s.mu.Unlock()
	fn(s.serverInfo())
	return func() {
		s.mu.Lock()
		delete(s.staticSubs, id)
		s.mu.Unlock()
	}
}

func (s *Server) updateNetwork(r stats.SpeedTestResult) {
	s.mu.Lock()
	n := &s.hw.Network
	n.SpeedDown, n.SpeedUp, n.LastSpeedTest, n.PublicIP = r.SpeedDown, r.SpeedUp, r.LastSpeedTest, r.PublicIP
	subs := make([]func(stats.ServerInfo), 0, len(s.staticSubs))
	for _, fn := range s.staticSubs {
		subs = append(subs, fn)
	}
	s.mu.Unlock()
	info := s.serverInfo()
	for _, fn := range subs {
		fn(info)
	}
}

// RunSpeedTests runs a speed test now, then on DASHDOT_SPEED_TEST_INTERVAL
// (minutes) or DASHDOT_SPEED_TEST_INTERVAL_CRON, until ctx ends.
func (s *Server) RunSpeedTests(ctx context.Context) {
	if !s.cfg.Widget("network") {
		return
	}
	run := func() {
		r, err := stats.SpeedTest(ctx, s.cfg)
		if err != nil {
			log.Printf("Speed test failed: %v", err)
			return
		}
		log.Printf("Speed test completed [%s]: down %.0f bit/s, up %.0f bit/s", r.Runner, r.SpeedDown, r.SpeedUp)
		s.updateNetwork(r)
	}

	if expr := s.cfg.SpeedTestIntervalCron; expr != nil {
		parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		if _, err := parser.Parse(*expr); err != nil {
			log.Printf("Invalid cron expression %q: %v", *expr, err)
		} else {
			log.Printf("Speed test cron expression: %s", *expr)
			c := cron.New(cron.WithParser(parser))
			_, _ = c.AddFunc(*expr, run)
			c.Start()
			defer c.Stop()
		}
	}

	log.Print("Running speed test (this may take a few minutes)...")
	run()

	if s.cfg.SpeedTestIntervalCron == nil {
		t := time.NewTicker(time.Duration(max(1, s.cfg.SpeedTestInterval)) * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				run()
			}
		}
	}
	<-ctx.Done()
}

// urlJoin joins path parts with single slashes, dropping empty ones.
func urlJoin(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p = strings.Trim(p, "/"); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "/")
}

// Handler serves everything under DASHDOT_ROUTING_PATH: the socket at
// <path>/socket, the integration routes and the UI.
func (s *Server) Handler(web fs.FS) http.Handler {
	base := "/" + urlJoin(s.cfg.RoutingPath)
	socketPath := "/" + urlJoin(s.cfg.RoutingPath, "socket")

	opts := socket.DefaultServerOptions()
	opts.SetPath(socketPath)
	opts.SetServeClient(false)
	if !s.cfg.DisableIntegrations {
		opts.SetCors(&types.Cors{Origin: "*"})
	}
	io := socket.NewServer(nil, opts)
	_ = io.On("connection", func(args ...any) {
		c := args[0].(*socket.Socket)
		unsubs := []func(){
			s.subscribeStatic(emitter[stats.ServerInfo](c, "static-info")),
			s.cpu.subscribe(emitter[stats.CPULoad](c, "cpu-load")),
			s.ram.subscribe(emitter[stats.RAMLoad](c, "ram-load")),
			s.storage.subscribe(emitter[stats.StorageLoad](c, "storage-load")),
			s.network.subscribe(emitter[stats.NetworkLoad](c, "network-load")),
			s.gpu.subscribe(emitter[stats.GPULoad](c, "gpu-load")),
		}
		_ = c.On("disconnect", func(...any) {
			for _, u := range unsubs {
				u()
			}
		})
	})

	routes := http.NewServeMux()
	if !s.cfg.DisableIntegrations {
		routes.HandleFunc("GET /config", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, struct {
				Config    *config.Config `json:"config"`
				Version   string         `json:"version,omitempty"`
				Buildhash string         `json:"buildhash,omitempty"`
			}{s.cfg, stats.Version, stats.Buildhash})
		})
		routes.HandleFunc("GET /info", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, s.hardware()) })
		routes.HandleFunc("GET /load/cpu", loadRoute(s.cpu, nil))
		// dashdot wrapped RAM, alone among the load routes.
		routes.HandleFunc("GET /load/ram", loadRoute(s.ram, func(v stats.RAMLoad) any { return map[string]stats.RAMLoad{"load": v} }))
		routes.HandleFunc("GET /load/storage", loadRoute(s.storage, nil))
		routes.HandleFunc("GET /load/network", loadRoute(s.network, nil))
		routes.HandleFunc("GET /load/gpu", loadRoute(s.gpu, nil))
	}
	routes.Handle("GET /", staticFiles(web))

	var app http.Handler = gzhttp.GzipHandler(routes)
	if !s.cfg.DisableIntegrations {
		app = cors(app)
	}

	mux := http.NewServeMux()
	mux.Handle(socketPath+"/", io.ServeHandler(opts))
	if base == "/" {
		mux.Handle("/", app)
	} else {
		mux.Handle(base+"/", http.StripPrefix(base, app))
		mux.Handle(base, http.StripPrefix(base, app))
	}
	return mux
}

func emitter[T any](c *socket.Socket, event string) func(T) {
	return func(v T) { _ = c.Emit(event, v) }
}

// loadRoute answers with a feed's latest reading, through wrap if given.
// Like dashdot, a failed reading is an empty 200.
func loadRoute[T any](f *feed[T], wrap func(T) any) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		v, err := f.current()
		if err != nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		if wrap == nil {
			writeJSON(w, v)
			return
		}
		writeJSON(w, wrap(v))
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

// cors is the cors package's default: any origin, and a 204 for preflight.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,PUT,PATCH,POST,DELETE")
			if h := r.Header.Get("Access-Control-Request-Headers"); h != "" {
				w.Header().Set("Access-Control-Allow-Headers", h)
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// staticFiles serves the UI. index.html must revalidate, so a new release
// shows on reload; the hashed assets can be cached for a year.
func staticFiles(web fs.FS) http.Handler {
	files := http.FileServerFS(web)
	if _, err := fs.Stat(web, "index.html"); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "The UI isn't built into this binary; the API still works.", http.StatusNotFound)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No directory listings: only the page and the files it loads.
		if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, ".html") {
			w.Header().Set("Cache-Control", "public, max-age=0")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000")
		}
		files.ServeHTTP(w, r)
	})
}
