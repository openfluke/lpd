package run

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/openfluke/lpd/internal/build"
	"github.com/openfluke/lpd/internal/catalog"
	"github.com/openfluke/lpd/internal/dataset/mnist"
	"github.com/openfluke/lpd/internal/dataset/tideds"
	"github.com/openfluke/lpd/internal/env"
	"github.com/openfluke/lpd/internal/sweep"
	"github.com/openfluke/tide/checkpoint"
	"github.com/openfluke/tide/dash"
	"github.com/openfluke/tide/ocean"
	"github.com/openfluke/tide/permute"
	"github.com/openfluke/tide/pulse"
	"github.com/openfluke/tide/river"
	"github.com/openfluke/tide/runner"
	"github.com/openfluke/welvet/simd"
)

// Options is the resolved runtime for one test run.
type Options struct {
	env.Config
	Test catalog.Test
}

func OceanWatcher(cfg env.Config) error {
	peers := parsePeers(cfg.OceanPeers)
	srv := &ocean.Server{
		Addr:  cfg.OceanAddr,
		Title: "lpd ocean",
		Peers: peers,
	}
	fmt.Printf("Ocean watching %d peer(s) on %s\n", len(peers), cfg.OceanAddr)
	fmt.Println(river.DashURLs(cfg.OceanAddr))
	return srv.ListenAndServe()
}

func Test(ctx context.Context, opt Options) error {
	cfg := opt.Config
	test := opt.Test
	if cfg.OutDir == "" {
		cfg.OutDir = test.OutDir
	}

	host, _ := os.Hostname()
	if host == "" {
		host = "local"
	}
	peerName := cfg.PeerName
	if peerName == "" {
		peerName = fmt.Sprintf("%s-shard%d", strings.ReplaceAll(test.Slug, "/", "-"), cfg.Shard)
	}

	cells, cellLR, err := test.Build(cfg.LR)
	if err != nil {
		return err
	}
	full := len(cells)
	cells = sweep.Shard(cells, cfg.Shard, cfg.Shards)
	if len(cells) == 0 {
		return fmt.Errorf("shard %d/%d got 0 cells (plan has %d)", cfg.Shard, cfg.Shards, full)
	}

	resultsPath := filepath.Join(cfg.OutDir, "results.json")
	ckptDir := filepath.Join(cfg.OutDir, "checkpoint")
	_ = os.MkdirAll(cfg.OutDir, 0o755)

	st := river.NewStore(resultsPath, host, test.Matrix, cfg.TrainN, mnist.SampleSeed, []float64{cfg.LR})
	st.SetPlan(cells)

	if cfg.OceanURL != "" {
		tideURL := river.DashURLs(cfg.Addr)
		go ocean.KeepRegistered(ctx, cfg.OceanURL, ocean.RegisterRequest{
			Name:  peerName,
			URL:   tideURL,
			Layer: test.Slug,
		})
		fmt.Printf("Registering with ocean %s as %s → %s\n", cfg.OceanURL, peerName, tideURL)
	}

	srv := &dash.Server{
		Tracker: nil, // set below
		Cells:   cells,
		Addr:    cfg.Addr,
		Task:    test.Slug,
		ID:      peerName,
		Subtitle: fmt.Sprintf("shard %d/%d · %d cells (of %d) · lr=%g",
			cfg.Shard, cfg.Shards, len(cells), full, cfg.LR),
		LR: cfg.LR,
		River: st,
		RiverOpts: river.Options{
			Title:       test.Title,
			Subtitle:    fmt.Sprintf("LPD %s · shard %d/%d", test.Slug, cfg.Shard, cfg.Shards),
			PDFFilename: strings.ReplaceAll(test.Slug, "/", "_") + ".pdf",
		},
	}

	if cfg.WebOnly {
		p := st.Progress()
		fmt.Printf("web-only → %s\n", river.DashURLs(cfg.Addr))
		fmt.Printf(" progress %d/%d (%.1f%%) · left %d\n", p.Done, p.Plan, p.Pct, p.Left)
		go func() { _ = srv.ListenAndServe() }()
		<-ctx.Done()
		return nil
	}

	workers := cfg.Workers
	if workers < 1 {
		workers = runtime.NumCPU()
		if workers < 2 {
			workers = 2
		}
		if workers > 8 {
			workers = 8
		}
	}

	dataDir := cfg.DataDir
	if dataDir == "" {
		dataDir = mnist.CacheDir()
	}
	fmt.Println("Loading MNIST from", dataDir, "…")
	split, err := mnist.Load(dataDir)
	if err != nil {
		return err
	}
	split.Train = mnist.TakeBalanced(split.Train, cfg.TrainN, mnist.SampleSeed)
	st.SetTrainN(len(split.Train))

	store := checkpoint.New(ckptDir, test.Matrix)
	var resume *checkpoint.Progress
	if !cfg.Fresh {
		resume, err = store.Load()
		if err != nil {
			return fmt.Errorf("checkpoint: %w", err)
		}
		resume = mergeResultsDone(resume, st, cells)
		resume = reconcileDoneToResults(resume, st, cells)
	}
	epoch, resume, alreadyDone := pinOneEpoch(resume, cells)

	tr := pulse.New()
	srv.Tracker = tr
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			fmt.Fprintf(os.Stderr, "tide: %v\n", err)
		}
	}()

	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Printf(" lpd %s\n", test.Slug)
	fmt.Printf(" shard %d/%d · %d/%d cells · train-n=%d · workers=%d · SIMD=%v\n",
		cfg.Shard, cfg.Shards, len(cells), full, cfg.TrainN, workers, simd.Enabled())
	fmt.Printf(" CamSync: %v\n", sweep.CamSyncLabels)
	fmt.Printf(" Tide %s\n", river.DashURLs(cfg.Addr))
	fmt.Printf(" MNIST cache %s\n", dataDir)
	fmt.Printf(" Results %s\n", resultsPath)
	fmt.Println("════════════════════════════════════════════════════════════")

	cfgRun := runner.DefaultConfig(cells)
	cfgRun.BatchSize = cfg.Batch
	cfgRun.Workers = workers
	cfgRun.Build = build.Cell
	cfgRun.NewDataset = func() runner.Dataset {
		cp := *split
		cp.Train = append([]mnist.Image(nil), split.Train...)
		cp.Val = append([]mnist.Image(nil), split.Val...)
		return tideds.New(&cp, cfg.Micro, mnist.SampleSeed)
	}
	cfgRun.Epoch = epoch
	cfgRun.PulseEvery = 200 * time.Millisecond
	cfgRun.CheckpointEvery = time.Minute
	cfgRun.LR = cfg.LR
	cfgRun.CellLR = cellLR
	cfgRun.Store = store
	cfgRun.Resume = resume
	cfgRun.FlipAt = 2
	cfgRun.FlipBack = 3

	ds := tideds.New(split, cfg.Micro, mnist.SampleSeed)
	tr.ResetDoneIDs(st.ResultDoneIDs())
	if rows := st.PulseResults(); len(rows) > 0 {
		tr.SeedReportLog(rows)
	}
	runner.Hydrate(tr, cfgRun, fmt.Sprintf("paused — %d/%d done", st.Progress().Done, st.Progress().Plan))

	p0 := st.Progress()
	if alreadyDone || p0.Left == 0 {
		srv.SignalStart()
		_ = st.SyncFromTracker(tr, cellLR)
		tr.Park("already finished — dash up")
		<-ctx.Done()
		return nil
	}

	if cfg.Autostart {
		srv.SignalStart()
	} else if err := srv.AwaitStart(ctx); err != nil {
		return err
	}

	go syncLoop(ctx, st, tr, cellLR)

	for round := 1; round <= 8; round++ {
		if ctx.Err() != nil {
			break
		}
		prog := st.Progress()
		if prog.Left == 0 {
			break
		}
		resume = reconcileDoneToResults(resume, st, cells)
		cfgRun.Resume = resume
		tr.ResetDoneIDs(st.ResultDoneIDs())
		fmt.Printf("round %d — %d/%d done · %d left\n", round, prog.Done, prog.Plan, prog.Left)
		if err := runner.Run(ctx, cfgRun, ds, tr); err != nil && ctx.Err() == nil {
			return err
		}
		_ = st.SyncFromTracker(tr, cellLR)
		if st.Progress().Left == 0 {
			break
		}
	}
	_ = st.SyncFromTracker(tr, cellLR)
	fmt.Printf("Done — %s\n", river.DashURLs(cfg.Addr))
	<-ctx.Done()
	return nil
}

func syncLoop(ctx context.Context, st *river.Store, tr *pulse.Tracker, cellLR map[string]float64) {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = st.SyncFromTracker(tr, cellLR)
		}
	}
}

func parsePeers(csv string) []ocean.Peer {
	var out []ocean.Peer
	for _, p := range strings.Split(csv, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, ocean.Peer{Name: p, URL: p})
	}
	return out
}

func mergeResultsDone(resume *checkpoint.Progress, st *river.Store, cells []permute.Cell) *checkpoint.Progress {
	if st == nil {
		return resume
	}
	snap := st.Snapshot()
	if len(snap.Rows) == 0 {
		return resume
	}
	if resume == nil {
		resume = &checkpoint.Progress{Epoch: 1}
	}
	done := checkpoint.DoneSet(resume)
	for _, r := range snap.Rows {
		if r.Status != "" && r.Status != "ok" && r.Status != "gap" {
			continue
		}
		for _, a := range permute.IDAliases(r.ID) {
			done[a] = true
		}
	}
	ids := make([]string, 0, len(done))
	for _, c := range cells {
		if permute.IDDone(done, c.ID) {
			ids = append(ids, c.ID)
		}
	}
	resume.DoneIDs = ids
	return resume
}

func reconcileDoneToResults(resume *checkpoint.Progress, st *river.Store, cells []permute.Cell) *checkpoint.Progress {
	if resume == nil {
		resume = &checkpoint.Progress{Epoch: 1}
	}
	have := map[string]bool{}
	for _, id := range st.ResultDoneIDs() {
		for _, a := range permute.IDAliases(id) {
			have[a] = true
		}
	}
	ids := make([]string, 0, len(cells))
	for _, c := range cells {
		if permute.IDDone(have, c.ID) {
			ids = append(ids, c.ID)
		}
	}
	resume.DoneIDs = ids
	return resume
}

func pinOneEpoch(resume *checkpoint.Progress, cells []permute.Cell) (epoch int, out *checkpoint.Progress, done bool) {
	if checkpoint.AllDone(resume, cells) {
		return 1, resume, true
	}
	epoch, out = checkpoint.PrepareEpoch(resume, cells)
	if epoch > 1 {
		epoch = 1
		if out != nil {
			out.Epoch = 1
		}
	}
	return epoch, out, checkpoint.AllDone(out, cells)
}

// MainContext returns a context cancelled on SIGINT/SIGTERM.
func MainContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
