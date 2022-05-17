#!/usr/bin/env bash
platforms=("linux/amd64" "linux/386" "darwin/amd64")

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}

    owl_name='owl-'$GOOS'-'$GOARCH

    if [ $GOOS = "windows" ]; then
        owl_name+='.exe'
    fi

    env GOOS=$GOOS GOARCH=$GOARCH go build -o build/$owl_name bin/cli.go

    engine_name='engine-'$GOOS'-'$GOARCH

    if [ $GOOS = "windows" ]; then
        engine_name+='.exe'
    fi

    env GOOS=$GOOS GOARCH=$GOARCH go build -o build/$engine_name main.go

    if [ $? -ne 0 ]; then
        echo 'An error has occurred! Aborting the script execution...'
        exit 1
    fi
done
