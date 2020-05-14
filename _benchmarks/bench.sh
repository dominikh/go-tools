#!/usr/bin/env sh
set -e
# PKG="k8s.io/kubernetes/pkg/..."
# LABEL="k8s"
PKG="std"
LABEL=$PKG
MIN_CORES=16
MAX_CORES=16
SAMPLES=5
WIPE_CACHE=1
BIN=$(realpath ./silent-staticcheck.sh)

go build ../cmd/staticcheck

export GO111MODULE=off

for cores in $(seq $MIN_CORES $MAX_CORES); do
	for i in $(seq 1 $SAMPLES); do
		procs=$((cores*2))
		if [ $WIPE_CACHE -ne 0 ]; then
			rm -rf ~/.cache/staticcheck
		fi
		
		out=$(env time -f "%e %M" taskset -c 0-$((procs-1)) $BIN $PKG 2>&1)
		t=$(echo "$out" | cut -f1 -d" ")
		m=$(echo "$out" | cut -f2 -d" ")
		ns=$(printf "%s 1000000000 * p" $t | dc)
		b=$((m * 1024))
		printf "BenchmarkStaticcheck-%s-%d  1   %.0f ns/op  %.0f B/op\n" "$LABEL" "$procs" "$ns" "$b"
	done
done
