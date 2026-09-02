#!/usr/bin/env bash
# Shared podman helpers (Linux + macOS Podman Machine).
set -euo pipefail

LPD_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GIT_ROOT="$(cd "${LPD_ROOT}/.." && pwd)"

BINARY="${LPD_ROOT}/bin/lpd"
IMAGE="${LPD_IMAGE:-lpd:latest}"
NAME="${LPD_CONTAINER:-lpd}"
CONTAINERFILE="${LPD_ROOT}/Containerfile"

die() {
	echo "lpd podman: $*" >&2
	exit 1
}

need_podman() {
	command -v podman >/dev/null 2>&1 || die "podman not found (install Podman or Podman Desktop)"
}

ensure_machine() {
	if [[ "$(uname -s)" == "Darwin" ]]; then
		if ! podman machine list --format '{{.Name}}' 2>/dev/null | grep -q .; then
			die "no podman machine — run: podman machine init && podman machine start"
		fi
		if ! podman info >/dev/null 2>&1; then
			echo "Starting podman machine…"
			podman machine start
		fi
	fi
}

env_file() {
	if [[ -f "${LPD_ROOT}/.env" ]]; then
		echo "${LPD_ROOT}/.env"
	elif [[ -f "${LPD_ROOT}/.env.example" ]]; then
		echo "${LPD_ROOT}/.env.example"
	else
		die "missing .env (copy .env.example)"
	fi
}

vol_opt() {
	printf '%s:%s:z' "$1" "$2"
}

ensure_data_dirs() {
	mkdir -p "${LPD_ROOT}/downloads/mnist" "${LPD_ROOT}/cnn2/bit/mnist/results"
}

check_siblings() {
	[[ -d "${GIT_ROOT}/tide" && -d "${GIT_ROOT}/welvet" && -d "${LPD_ROOT}" ]] \
		|| die "expected ${GIT_ROOT}/{tide,welvet,lpd} for go build (go.mod replace paths)"
}

build_binary() {
	check_siblings
	ensure_data_dirs
	mkdir -p "${LPD_ROOT}/bin"
	echo "Compiling lpd → bin/lpd …"
	if [[ "$(uname -s)" == "Darwin" ]]; then
		# Mac host binary is darwin — compile a linux binary in an ephemeral builder container.
		podman run --rm \
			-v "${GIT_ROOT}:/src:Z" \
			-w /src/lpd \
			docker.io/library/golang:1.22-bookworm \
			go build -trimpath -ldflags="-s -w" -o bin/lpd .
	else
		(
			cd "${LPD_ROOT}"
			go build -trimpath -ldflags="-s -w" -o bin/lpd .
		)
	fi
	[[ -x "${BINARY}" ]] || die "compile failed — no ${BINARY}"
}

build_image() {
	[[ -x "${BINARY}" ]] || die "missing ${BINARY} — run: ./podman/build"
	echo "Building ${IMAGE} (binary only, context ${LPD_ROOT})…"
	podman build -f "${CONTAINERFILE}" -t "${IMAGE}" "${LPD_ROOT}"
}

container_running() {
	podman container exists "${NAME}" 2>/dev/null \
		&& [[ "$(podman inspect -f '{{.State.Running}}' "${NAME}" 2>/dev/null || echo false)" == "true" ]]
}

container_exists() {
	podman container exists "${NAME}" 2>/dev/null
}

run_container() {
	ensure_data_dirs
	ENV="$(env_file)"
	echo "Using env: ${ENV}"
	echo "Data on host: ${LPD_ROOT}/downloads  ${LPD_ROOT}/cnn2"

	podman run -d \
		--name "${NAME}" \
		--env-file "${ENV}" \
		-p "8301:8301" \
		-p "8090:8090" \
		-v "$(vol_opt "${LPD_ROOT}/downloads" /app/downloads)" \
		-v "$(vol_opt "${LPD_ROOT}/cnn2" /app/cnn2)" \
		"${IMAGE}" \
		"$@"
}

print_urls() {
	echo ""
	echo "lpd running (${NAME})"
	echo "  Tide   http://127.0.0.1:8301"
	echo "  Ocean  http://127.0.0.1:8090  (when LPD_OCEAN_ONLY=true)"
	echo ""
	echo "  Host data (persists when pod is stopped):"
	echo "    ${LPD_ROOT}/downloads"
	echo "    ${LPD_ROOT}/cnn2"
	echo ""
	echo "  logs:    podman logs -f ${NAME}"
	echo "  stop:    ${LPD_ROOT}/podman/stop"
	echo "  restart: ${LPD_ROOT}/podman/restart"
}
