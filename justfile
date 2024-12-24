run:
    go build -o relay && godotenv ./relay

verbose:
    go build -o relay -tags debug && godotenv ./relay

build-prod:
    rm -f relay
    CC=$(which musl-gcc) go build -ldflags='-s -w -linkmode external -extldflags "-static"' -o ./relay

prettify:
    gofumpt -w .
