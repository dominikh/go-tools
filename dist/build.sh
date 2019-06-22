#!/bin/sh -e

SYSTEMS=(windows linux freebsd darwin)
ARCHS=(amd64 386)

rev="$1"
if [ -z "$rev" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi


mkdir "$rev"
d=$(realpath "$rev")

wrk=$(mktemp -d)
trap "{ rm -rf \"$wrk\"; }" EXIT
cd "$wrk"

go mod init foo
GO111MODULE=on go get -d honnef.co/go/tools/cmd/staticcheck@"$rev"

for os in ${SYSTEMS[@]}; do
    for arch in ${ARCHS[@]}; do
        echo "Building GOOS=$os GOARCH=$arch..."
        out="staticcheck_${os}_${arch}"
        if [ $os = "windows" ]; then
            out="${out}.exe"
        fi

        CGO_ENABLED=0 GOOS=$os GOARCH=$arch GO111MODULE=on go build -o "$d/$out" honnef.co/go/tools/cmd/staticcheck
        (
            cd "$d"
            sha256sum "$out" > "$out".sha256
        )
    done
done

(
    cd "$d"
    sha256sum -c --strict *.sha256
)
