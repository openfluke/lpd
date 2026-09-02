# lpd

Centralized **LPD configuration lab** — sweep train modes, dtypes, layers, and CamSync
using [tide](https://github.com/openfluke/tide). Results live under
`layer/dtype/dataset/` (e.g. `cnn2/bit/mnist/results`).

## Layout

```
lpd/
  downloads/mnist/     ← shared MNIST cache (auto-downloaded)
  cnn2/bit/mnist/      ← test #1 results + checkpoint
  internal/            ← dataset, sweep, runner
  main.go              ← menu or .env auto-start
```

## Quick start

```bash
cp .env.example .env
go run .
```

### Podman (Linux / macOS)

Compiles **`bin/lpd`** (on Mac, via a one-off Linux builder container), then packs **only that binary** into the runtime image. The running pod has no source code. Data lives on the host under `downloads/` and `cnn2/` — stop/start does not delete it.

```bash
cp .env.example .env
chmod +x podman/*
./podman/build        # compile + pack image (binary only in pod)
./podman/start        # run (reuses stopped container)
./podman/stop         # stop — container kept; data stays in downloads/ cnn2/
./podman/restart      # stop then start
```

After code changes: `./podman/build && ./podman/start --recreate`

Drop container only (data untouched): `./podman/stop --rm`

Or pick from the menu:

```bash
go run . -menu
```

Test **1** — `cnn2/bit/mnist` (~211k cells before sharding):

- dtype: **binary** (FormatNone)
- layer: **CNN2** stem + cameral head
- cams: **1–6**
- CamSync: **1%, 10%, 50%, 75%, 100%** (cam ≥ 2)
- modes: **31 uniform** stack modes + **cam-mix** (distinct mode pairs × alt/block/roundrobin)
- lr: **0.005, 0.05, 0.5**
- data: class-balanced MNIST subset (`downloads/mnist`)

## Two Macs

**Mac A** (train shard 0):

```env
LPD_SHARD=0
LPD_SHARDS=2
LPD_PEER_NAME=mac-a
LPD_OCEAN_URL=http://<watcher-ip>:8090
```

**Mac B** (train shard 1):

```env
LPD_SHARD=1
LPD_SHARDS=2
LPD_PEER_NAME=mac-b
LPD_OCEAN_URL=http://<watcher-ip>:8090
```

**Watcher** (ocean only):

```env
LPD_OCEAN_ONLY=true
LPD_OCEAN_PEERS=http://<mac-a-ip>:8301,http://<mac-b-ip>:8301
```

Open ocean at `:8090`, each worker tide at `:8301` (compare/LPD pages on same port).

## Siblings

```bash
# go.mod replace paths assume:
../tide
../welvet
```
