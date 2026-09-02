package catalog

import (
	"fmt"

	"github.com/openfluke/lpd/internal/sweep"
	"github.com/openfluke/tide/permute"
)

// Test is one LPD experiment (layer/dtype/dataset).
type Test struct {
	ID      int
	Slug    string // e.g. cnn2/bit/mnist
	Title   string
	OutDir  string
	Matrix  string
	Build   func(lr float64) ([]permute.Cell, map[string]float64, error)
}

// All registered tests (menu order).
var All = []Test{
	{
		ID:     1,
		Slug:   "cnn2/bit/mnist",
		Title:  "CNN2 · binary · MNIST — cam 1–6 × CamSync 1/10/50/75/100% × all train modes",
		OutDir: "cnn2/bit/mnist/results",
		Matrix: "cnn2_bit_mnist",
		Build:  sweep.CNN2BitMNIST,
	},
}

func ByID(id int) (Test, error) {
	for _, t := range All {
		if t.ID == id {
			return t, nil
		}
	}
	return Test{}, fmt.Errorf("unknown test %d", id)
}

func Menu() {
	fmt.Println("lpd — configuration lab")
	fmt.Println()
	for _, t := range All {
		cells, _, err := t.Build(0.6)
		n := 0
		if err == nil {
			n = len(cells)
		}
		fmt.Printf("  %d) %s  (~%d cells)\n", t.ID, t.Slug, n)
		fmt.Printf("     %s\n", t.Title)
	}
	fmt.Println("  9) ocean watcher (poll peer tides)")
	fmt.Println("  0) exit")
}
