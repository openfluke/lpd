package tideds

import (
	"math/rand/v2"
	"sync"

	"github.com/openfluke/lpd/internal/dataset/mnist"
	"github.com/openfluke/tide/runner"
	"github.com/openfluke/welvet/core"
)

// DS serves random val batches and one sequential train pass per epoch.
type DS struct {
	mu     sync.Mutex
	train  []mnist.Image
	val    []mnist.Image
	batch  int
	seed   uint64
	rng    *rand.Rand
	offset int
	order  []int
}

func New(split *mnist.Split, batch int, seed uint64) *DS {
	if batch < 1 {
		batch = 8
	}
	d := &DS{
		train: split.Train,
		val:   split.Val,
		batch: batch,
		seed:  seed,
		rng:   rand.New(rand.NewPCG(seed, seed^0xC0FFEE)),
	}
	d.ResetEpoch(0)
	return d
}

func (d *DS) NextServe(phase string) runner.Sample {
	_ = phase
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.batchFrom(d.val, true)
}

func (d *DS) TrainLen() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.train)
}

func (d *DS) EpochOffset() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.offset
}

func (d *DS) ResetEpoch(offset int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.train)
	d.order = make([]int, n)
	for i := range d.order {
		d.order[i] = i
	}
	shuf := rand.New(rand.NewPCG(d.seed, d.seed^0xE90C4))
	for i := n - 1; i > 0; i-- {
		j := shuf.IntN(i + 1)
		d.order[i], d.order[j] = d.order[j], d.order[i]
	}
	if offset < 0 {
		offset = 0
	}
	if offset > n {
		offset = n
	}
	d.offset = offset
}

func (d *DS) NextTrain() (runner.Sample, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.train)
	if d.offset >= n {
		return runner.Sample{}, false
	}
	remain := n - d.offset
	bs := d.batch
	if bs > remain {
		bs = remain
	}
	x := core.NewTensor[float32](bs, 1, 28, 28)
	target := core.NewTensor[float32](bs, 10)
	labels := make([]int, bs)
	for i := 0; i < bs; i++ {
		im := d.train[d.order[d.offset]]
		d.offset++
		labels[i] = im.Label
		copy(x.Data[i*784:(i+1)*784], im.Pixels)
		if im.Label >= 0 && im.Label < 10 {
			target.Data[i*10+im.Label] = 1
		}
	}
	return runner.Sample{X: x, Target: target, Labels: labels}, true
}

func (d *DS) batchFrom(pool []mnist.Image, random bool) runner.Sample {
	n := d.batch
	if n > len(pool) {
		n = len(pool)
	}
	x := core.NewTensor[float32](n, 1, 28, 28)
	target := core.NewTensor[float32](n, 10)
	labels := make([]int, n)
	for i := 0; i < n; i++ {
		var im mnist.Image
		if random {
			im = pool[d.rng.IntN(len(pool))]
		} else {
			im = pool[i]
		}
		labels[i] = im.Label
		copy(x.Data[i*784:(i+1)*784], im.Pixels)
		if im.Label >= 0 && im.Label < 10 {
			target.Data[i*10+im.Label] = 1
		}
	}
	return runner.Sample{X: x, Target: target, Labels: labels}
}
