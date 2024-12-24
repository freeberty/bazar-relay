run:
    go build && ./relay

verbose:
    go build -tags debug && ./relay

build-prod:
    rm -f relay
    CC=$(which musl-gcc) go build -ldflags='-s -w -linkmode external -extldflags "-static"' -o ./relay

prettify:
    gofumpt -w .
