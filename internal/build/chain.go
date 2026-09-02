package build

import (
	"fmt"

	"github.com/openfluke/lpd/internal/sweep"
	"github.com/openfluke/tide/chain"
	"github.com/openfluke/tide/permute"
	"github.com/openfluke/tide/runner"
	"github.com/openfluke/welvet/layers/parallel"
)

// Cell builds a CNN2 MNIST stack with optional CamSync and per-cam BranchModes (bm=).
func Cell(cell permute.Cell) (runner.Net, error) {
	spec := chain.DefaultMNIST()
	m, err := chain.Build(spec, cell)
	if err != nil {
		return nil, err
	}
	if bm, ok := sweep.ParseBM(cell.ID); ok && m.Para != nil {
		modes := make([]parallel.TrainMode, 0, len(bm))
		for _, tok := range bm {
			tm := permute.TrainMode(tok)
			wv, err := tm.Welvet()
			if err != nil {
				return nil, fmt.Errorf("bm token %q: %w", tok, err)
			}
			modes = append(modes, wv)
		}
		m.Para.SetBranchModes(modes...)
	}
	if a, ok := sweep.ParseCS(cell.ID); ok {
		n := cell.Cams
		if n <= 0 {
			n = permute.CamsOf(cell.Arch)
		}
		if n >= 2 && a > 0 {
			_ = m.SetCamSync(parallel.CamSyncConfig{
				Enabled: true,
				Alpha:   a,
				When:    parallel.SyncAfterSample,
			})
		}
	}
	return m, nil
}
