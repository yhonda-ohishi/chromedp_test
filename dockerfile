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
# 必要なパッケージをインストール
RUN apk add --no-cache ca-certificates
# chromedpを使用するための依存関係をインストール
RUN apk add --no-cache chromium

# ワーキングディレクトリを設定
WORKDIR /app
# ビルドしたバイナリをコピー
COPY --from=go_builder /app/server .
# ポートを公開
EXPOSE 8080
# コンテナ起動時に実行するコマンドを指定
CMD ["./server"]
