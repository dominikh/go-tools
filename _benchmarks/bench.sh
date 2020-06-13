#!/usr/bin/env bash
set -e

declare -A PKGS=(
	["strconv"]="strconv"
	["std"]="std"
	["k8s"]="k8s.io/kubernetes/pkg/..."
)

MIN_CORES=1
MAX_CORES=16
MIN_GOGC=10
MAX_GOGC=100
SAMPLES=5
WIPE_CACHE=1
FORMAT=csv
BIN=$(realpath ./silent-staticcheck.sh)
SMT=1


runBenchmark() {
	local pkg="$1"
	local label="$2"
	local gc="$3"
	local cores="$4"
	local wipe="$5"

	if [ $wipe -ne 0 ]; then
		rm -rf ~/.cache/staticcheck
	fi

	local procs
	if [ $SMT -ne 0 ]; then
		procs=$((cores*2))
	else
		procs=$cores
	fi

	local out=$(GOGC=$gc env time -f "%e %M" taskset -c 0-$((procs-1)) $BIN $pkg 2>&1)
	local t=$(echo "$out" | cut -f1 -d" ")
	local m=$(echo "$out" | cut -f2 -d" ")
	local ns=$(printf "%s 1000000000 * p" $t | dc)
	local b=$((m * 1024))

	case $FORMAT in
		bench)
			printf "BenchmarkStaticcheck-%s-GOGC%d-wiped%d-%d  1   %.0f ns/op  %.0f B/op\n" "$label" "$gc" "$wipe" "$procs" "$ns" "$b"
			;;
		csv)
			printf "%s,%d,%d,%d,%.0f,%.0f\n" "$label" "$gc" "$procs" "$wipe" "$ns" "$b"
			;;
	esac
}

go build ../cmd/staticcheck
export GO111MODULE=off

if [ "$FORMAT" = "csv" ]; then
	printf "packages,gogc,procs,wipe-cache,time,memory\n"
fi

for label in "${!PKGS[@]}"; do
	pkg=${PKGS[$label]}
	for gc in $(seq $MIN_GOGC 10 $MAX_GOGC); do
		for cores in $(seq $MIN_CORES $MAX_CORES); do
			for i in $(seq 1 $SAMPLES); do
				runBenchmark "$pkg" "$label" "$gc" "$cores" 1
				runBenchmark "$pkg" "$label" "$gc" "$cores" 0
			done
		done
	done
done
