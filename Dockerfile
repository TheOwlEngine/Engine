FROM ubuntu:latest

WORKDIR /app

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
    # ffmpeg
    ffmpeg \
    # golang
    golang \
    # tesseract
    libtesseract-dev \
    # leptonica
    libleptonica-dev \
    # tesseract english
    tesseract-ocr-eng \
    # tesseract indonesian
    tesseract-ocr-ind \
    # cleanup
    && rm -rf /var/lib/apt/lists/*

# processs reaper
ENTRYPOINT ["dumb-init", "--"]

COPY main.go go.mod go.sum ./
COPY lib ./lib
COPY types ./types

ENV GO111MODULE=on

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /app/engine
RUN go run ./lib/browser/install.go

COPY flows ./flows
COPY logs ./logs
COPY resources ./resources

CMD /app/engine