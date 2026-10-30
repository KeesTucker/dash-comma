package stats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"time"

	"github.com/KeesTucker/dash-comma/internal/config"
)

// SpeedTestResult is the part of NetworkInfo a speed test fills in.
type SpeedTestResult struct {
	SpeedDown     float64
	SpeedUp       float64
	PublicIP      string
	LastSpeedTest float64
	Runner        string
}

var errNoSpeedTest = errors.New(`there is no speed test installed: use one of
- speedtest: https://www.speedtest.net/apps/cli (with DASHDOT_ACCEPT_OOKLA_EULA=true)
- speedtest-cli: https://github.com/sivel/speedtest-cli
or give a file of results with DASHDOT_SPEED_TEST_FROM_PATH`)

// SpeedTest runs the same runners as dashdot, in the same order: a results
// file, Ookla's speedtest, then speedtest-cli. Speeds are in bits per second.
func SpeedTest(ctx context.Context, cfg *config.Config) (SpeedTestResult, error) {
	started := float64(time.Now().UnixMilli())
	wantIP := slices.Contains(cfg.NetworkLabelList, "public_ip")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var r SpeedTestResult
	switch {
	case cfg.SpeedTestFromPath != nil:
		b, err := os.ReadFile(*cfg.SpeedTestFromPath)
		if err != nil {
			return r, err
		}
		var j struct {
			SpeedDown float64 `json:"speedDown"`
			SpeedUp   float64 `json:"speedUp"`
			PublicIP  string  `json:"publicIp"`
			Unit      string  `json:"unit"`
		}
		if err := json.Unmarshal(b, &j); err != nil {
			return r, err
		}
		mult := 1.0
		switch j.Unit {
		case "", "bit":
		case "byte":
			mult = 8
		default:
			return r, errors.New("you can only use 'bit' or 'byte' as a unit for your speed test results")
		}
		r = SpeedTestResult{SpeedDown: j.SpeedDown * mult, SpeedUp: j.SpeedUp * mult, PublicIP: j.PublicIP, Runner: "file"}

	case cfg.AcceptOoklaEULA && onPath("speedtest"):
		out, err := exec.CommandContext(ctx, "speedtest", "--accept-license", "--accept-gdpr", "-f", "json").Output()
		if err != nil {
			return r, fmt.Errorf("speedtest: %w", err)
		}
		var j struct {
			Download  struct{ Bandwidth float64 }
			Upload    struct{ Bandwidth float64 }
			Interface struct {
				ExternalIP string `json:"externalIp"`
			}
		}
		if err := json.Unmarshal(out, &j); err != nil {
			return r, err
		}
		r = SpeedTestResult{SpeedDown: j.Download.Bandwidth * 8, SpeedUp: j.Upload.Bandwidth * 8, PublicIP: j.Interface.ExternalIP, Runner: "ookla"}

	case onPath("speedtest-cli"):
		out, err := exec.CommandContext(ctx, "speedtest-cli", "--json", "--secure").Output()
		if err != nil {
			return r, fmt.Errorf("speedtest-cli: %w", err)
		}
		var j struct {
			Download float64
			Upload   float64
			Client   struct{ IP string }
		}
		if err := json.Unmarshal(out, &j); err != nil {
			return r, err
		}
		r = SpeedTestResult{SpeedDown: j.Download, SpeedUp: j.Upload, PublicIP: j.Client.IP, Runner: "speedtest-cli"}

	default:
		return r, errNoSpeedTest
	}

	if !wantIP {
		r.PublicIP = ""
	}
	r.LastSpeedTest = started
	return r, nil
}

func onPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
