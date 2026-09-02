package env

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Config is loaded from .env then overridden by flags.
type Config struct {
	Test       int
	Autostart  bool
	Addr       string
	TrainN     int
	Workers    int
	Micro      int
	Batch      int
	LR         float64
	Shard      int
	Shards     int
	Fresh      bool
	WebOnly    bool
	OceanURL   string
	OceanOnly  bool
	OceanAddr  string
	OceanPeers string
	PeerName   string
	DataDir    string
	OutDir     string
}

func Load(path string) Config {
	c := Config{
		Test:      1,
		Autostart: true,
		Addr:      "0.0.0.0:8301",
		TrainN:    2000,
		Workers:   0,
		Micro:     8,
		Batch:     8,
		LR:        0.6,
		Shard:     0,
		Shards:    1,
		OceanAddr: "0.0.0.0:8090",
		DataDir:   "", // → downloads/mnist via mnist.CacheDir()
		OutDir:    "cnn2/bit/mnist/results",
	}
	if path == "" {
		path = ".env"
	}
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			applyEnvPair(&c, sc.Text())
		}
		f.Close()
	}
	// Podman --env-file / exported shell vars (override file defaults).
	applyEnvPair(&c, "TEST="+os.Getenv("LPD_TEST"))
	applyEnvPair(&c, "AUTOSTART="+os.Getenv("LPD_AUTOSTART"))
	applyEnvPair(&c, "ADDR="+os.Getenv("LPD_ADDR"))
	applyEnvPair(&c, "TRAIN_N="+os.Getenv("LPD_TRAIN_N"))
	applyEnvPair(&c, "WORKERS="+os.Getenv("LPD_WORKERS"))
	applyEnvPair(&c, "MICRO="+os.Getenv("LPD_MICRO"))
	applyEnvPair(&c, "BATCH="+os.Getenv("LPD_BATCH"))
	applyEnvPair(&c, "LR="+os.Getenv("LPD_LR"))
	applyEnvPair(&c, "SHARD="+os.Getenv("LPD_SHARD"))
	applyEnvPair(&c, "SHARDS="+os.Getenv("LPD_SHARDS"))
	applyEnvPair(&c, "FRESH="+os.Getenv("LPD_FRESH"))
	applyEnvPair(&c, "WEB_ONLY="+os.Getenv("LPD_WEB_ONLY"))
	applyEnvPair(&c, "OCEAN_URL="+os.Getenv("LPD_OCEAN_URL"))
	applyEnvPair(&c, "OCEAN_ONLY="+os.Getenv("LPD_OCEAN_ONLY"))
	applyEnvPair(&c, "OCEAN_ADDR="+os.Getenv("LPD_OCEAN_ADDR"))
	applyEnvPair(&c, "OCEAN_PEERS="+os.Getenv("LPD_OCEAN_PEERS"))
	applyEnvPair(&c, "PEER_NAME="+os.Getenv("LPD_PEER_NAME"))
	applyEnvPair(&c, "DATA_DIR="+os.Getenv("LPD_DATA_DIR"))
	applyEnvPair(&c, "OUT_DIR="+os.Getenv("LPD_OUT_DIR"))
	return c
}

func applyEnvPair(c *Config, line string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}
	k, v, ok := strings.Cut(line, "=")
	if !ok {
		return
	}
	k = strings.TrimSpace(k)
	v = strings.TrimSpace(strings.Trim(v, `"'`))
	if v == "" && strings.HasPrefix(strings.ToUpper(k), "LPD_") {
		return
	}
	if v == "" && k != "" {
		// os.Getenv miss — skip (keep file/default).
		switch strings.ToUpper(k) {
		case "TEST", "LPD_TEST", "TRAIN_N", "LPD_TRAIN_N", "WORKERS", "LPD_WORKERS",
			"MICRO", "LPD_MICRO", "BATCH", "LPD_BATCH", "LR", "LPD_LR",
			"SHARD", "LPD_SHARD", "SHARDS", "LPD_SHARDS", "ADDR", "LPD_ADDR",
			"OCEAN_URL", "LPD_OCEAN_URL", "OCEAN_ADDR", "LPD_OCEAN_ADDR",
			"OCEAN_PEERS", "LPD_OCEAN_PEERS", "PEER_NAME", "LPD_PEER_NAME",
			"DATA_DIR", "LPD_DATA_DIR", "OUT_DIR", "LPD_OUT_DIR":
			return
		}
	}
	switch strings.ToUpper(k) {
	case "TEST", "LPD_TEST":
		c.Test, _ = strconv.Atoi(v)
	case "AUTOSTART", "LPD_AUTOSTART":
		c.Autostart = parseBool(v, true)
	case "ADDR", "LPD_ADDR":
		c.Addr = v
	case "TRAIN_N", "LPD_TRAIN_N":
		c.TrainN, _ = strconv.Atoi(v)
	case "WORKERS", "LPD_WORKERS":
		c.Workers, _ = strconv.Atoi(v)
	case "MICRO", "LPD_MICRO":
		c.Micro, _ = strconv.Atoi(v)
	case "BATCH", "LPD_BATCH":
		c.Batch, _ = strconv.Atoi(v)
	case "LR", "LPD_LR":
		c.LR, _ = strconv.ParseFloat(v, 64)
	case "SHARD", "LPD_SHARD":
		c.Shard, _ = strconv.Atoi(v)
	case "SHARDS", "LPD_SHARDS":
		c.Shards, _ = strconv.Atoi(v)
	case "FRESH", "LPD_FRESH":
		c.Fresh = parseBool(v, false)
	case "WEB_ONLY", "LPD_WEB_ONLY":
		c.WebOnly = parseBool(v, false)
	case "OCEAN_URL", "LPD_OCEAN_URL":
		c.OceanURL = v
	case "OCEAN_ONLY", "LPD_OCEAN_ONLY":
		c.OceanOnly = parseBool(v, false)
	case "OCEAN_ADDR", "LPD_OCEAN_ADDR":
		c.OceanAddr = v
	case "OCEAN_PEERS", "LPD_OCEAN_PEERS":
		c.OceanPeers = v
	case "PEER_NAME", "LPD_PEER_NAME":
		c.PeerName = v
	case "DATA_DIR", "LPD_DATA_DIR":
		c.DataDir = v
	case "OUT_DIR", "LPD_OUT_DIR":
		c.OutDir = v
	}
}

func parseBool(s string, def bool) bool {
	switch strings.ToLower(s) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}
