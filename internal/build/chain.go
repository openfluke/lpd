package build

import (
	"github.com/openfluke/lpd/internal/sweep"
	"github.com/openfluke/tide/chain"
	"github.com/openfluke/tide/permute"
	"github.com/openfluke/tide/runner"
	"github.com/openfluke/welvet/layers/parallel"
)

// Cell builds a CNN2 MNIST stack with optional CamSync on the cameral head.
func Cell(cell permute.Cell) (runner.Net, error) {
	spec := chain.DefaultMNIST()
	m, err := chain.Build(spec, cell)
	if err != nil {
		return nil, err
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
