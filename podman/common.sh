#!/usr/bin/env bash
# Shared podman helpers (Linux + macOS Podman Machine).
set -euo pipefail

LPD_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GIT_ROOT="$(cd "${LPD_ROOT}/.." && pwd)"

IMAGE="${LPD_IMAGE:-lpd:latest}"
NAME="${LPD_CONTAINER:-lpd}"
CONTAINERFILE="${LPD_ROOT}/Containerfile"
IGNOREFILE="${LPD_ROOT}/.containerignore"

die() {
	echo "lpd podman: $*" >&2
	exit 1
}

need_podman() {
	command -v podman >/dev/null 2>&1 || die "podman not found (install Podman or start Podman Desktop)"
}

ensure_machine() {
	# macOS: Podman runs inside a VM.
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
	# :z works on Linux (SELinux) and is ignored harmlessly on many Mac setups.
	printf '%s:%s:z' "$1" "$2"
}

build_image() {
	[[ -d "${GIT_ROOT}/tide" && -d "${GIT_ROOT}/welvet" && -d "${GIT_ROOT}/lpd" ]] \
		|| die "expected ${GIT_ROOT}/{tide,welvet,lpd} — build from the git parent layout"

	echo "Building ${IMAGE} (context ${GIT_ROOT})…"
	podman build \
		-f "${CONTAINERFILE}" \
		--ignorefile "${IGNOREFILE}" \
		-t "${IMAGE}" \
		"${GIT_ROOT}"
}
