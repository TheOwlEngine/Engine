FROM golang AS builder

WORKDIR /app

COPY main.go go.mod go.sum ./
COPY lib ./lib
COPY types ./types

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /app/engine
RUN go run ./lib/browser/install.go

FROM ubuntu:latest

ARG apt_sources="http://archive.ubuntu.com"

RUN sed -i "s|http://archive.ubuntu.com|$apt_sources|g" /etc/apt/sources.list && \
    apt-get update && \
    apt-get install --no-install-recommends -y \
    # chromium dependencies
    libnss3 \
    libxss1 \
    libasound2 \
    libxtst6 \
    libgtk-3-0 \
    libgbm1 \
    ca-certificates \
    # fonts
    fonts-liberation fonts-noto-color-emoji fonts-noto-cjk \
    # timezone
    tzdata \
    # processs reaper
    dumb-init \
    # headful mode support, for example: $ xvfb-run chromium-browser --remote-debugging-port=9222
    xvfb \
    # cleanup
    && rm -rf /var/lib/apt/lists/*

# processs reaper
ENTRYPOINT ["dumb-init", "--"]

COPY --from=builder /root/.cache/rod /root/.cache/rod
COPY --from=builder /app/engine /usr/bin/

COPY flows ./flows
COPY logs ./logs
COPY resources ./resources

CMD engine