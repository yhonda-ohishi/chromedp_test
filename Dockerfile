# --- ステージ1: Goアプリケーションのビルド ---
FROM golang:1.24.4-alpine AS go_builder
# ワーキングディレクトリを設定
WORKDIR /app
# 必要なパッケージをインストール
RUN apk add --no-cache git
# Goモジュールの初期化
COPY go.mod go.sum ./
RUN go mod download
# アプリケーションのソースコードをコピー
COPY . .
# アプリケーションをビルド
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o server

FROM alpine:latest AS final
# Cloudflare Containers環境に最適化されたパッケージのインストール
RUN apk add --no-cache \
    ca-certificates \
    chromium \
    chromium-chromedriver \
    font-noto \
    font-noto-cjk \
    && rm -rf /var/cache/apk/* /tmp/*

# Chromiumの設定
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROME_PATH=/usr/bin/chromium-browser
ENV CHROMIUM_FLAGS="--no-sandbox --headless --disable-gpu --disable-dev-shm-usage"

# 非rootユーザーの作成（セキュリティ向上）
RUN addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup

# ワーキングディレクトリを設定
WORKDIR /app

# 必要なディレクトリを作成し、権限を設定
RUN mkdir -p /app/download /tmp/chrome-user-data && \
    chown -R appuser:appgroup /app /tmp/chrome-user-data && \
    chmod 755 /app/download

# ビルドしたバイナリをコピー
COPY --from=go_builder /app/server .
RUN chown appuser:appgroup /app/server

# 非rootユーザーに切り替え
USER appuser

# ポートを公開
EXPOSE 8080

# コンテナ起動時に実行するコマンドを指定
CMD ["./server"]
