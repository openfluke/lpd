#!/usr/bin/env bash
# Native macOS/Linux runner — no Podman. Data stays in downloads/ and cnn2/.
set -euo pipefail

LPD_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GIT_ROOT="$(cd "${LPD_ROOT}/.." && pwd)"

BINARY="${LPD_ROOT}/bin/lpd"
PIDFILE="${LPD_ROOT}/.lpd.pid"
LOGFILE="${LPD_ROOT}/lpd.log"
NAME="lpd"

die() {
	echo "lpd mac: $*" >&2
	exit 1
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

ensure_data_dirs() {
	mkdir -p "${LPD_ROOT}/downloads/mnist" "${LPD_ROOT}/cnn2/bit/mnist/results"
}

check_siblings() {
	[[ -d "${GIT_ROOT}/tide" && -d "${GIT_ROOT}/welvet" && -d "${LPD_ROOT}" ]] \
		|| die "expected ${GIT_ROOT}/{tide,welvet,lpd}"
}

build_binary() {
	check_siblings
	ensure_data_dirs
	echo "Compiling lpd → bin/lpd …"
	mkdir -p "${LPD_ROOT}/bin"
	(
		cd "${LPD_ROOT}"
		go build -trimpath -ldflags="-s -w" -o bin/lpd .
	)
	[[ -x "${BINARY}" ]] || die "compile failed"
}

read_pid() {
	if [[ -f "${PIDFILE}" ]]; then
		cat "${PIDFILE}"
	fi
}

is_running() {
	local pid
	pid="$(read_pid || true)"
	[[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null
}

print_urls() {
	echo ""
	echo "lpd running natively (pid $(read_pid))"
	echo "  Tide   http://127.0.0.1:8301"
	echo "  log    ${LOGFILE}"
	echo ""
	echo "  status: ${LPD_ROOT}/mac/status"
	echo "  logs:   ${LPD_ROOT}/mac/logs"
	echo "  stop:   ${LPD_ROOT}/mac/stop"
}
