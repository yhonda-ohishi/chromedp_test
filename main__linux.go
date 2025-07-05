//go:build linux
// +build linux

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

var err error
var isProcessing bool
var processingMutex sync.Mutex

type requestData struct {
	Data []struct {
		RisLoginId  string `json:"risLoginId"`
		RisPassword string `json:"risPassword"`
	} `json:"data"`
	ResUrl string `json:"resUrl"`
}

func main() {
	//http handlerを設定
	log.SetFlags(log.LstdFlags | log.Lshortfile) // ログにタイムスタンプとファイル名を表示
	http.HandleFunc("/etc-meisai", func(w http.ResponseWriter, r *http.Request) {
		// favicon.icoなどの無関係なリクエストを除外
		if r.URL.Path != "/etc-meisai" {
			http.NotFound(w, r)
			return
		}

		// 処理中かどうかをチェック
		processingMutex.Lock()
		if isProcessing {
			processingMutex.Unlock()
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status": "error", "message": "既にetc-meisaiのダウンロード処理が実行中です。"}`)
			return
		}
		isProcessing = true
		processingMutex.Unlock()
		log.Println("ETC明細のダウンロードを開始します...")

		//postリクエストであれば、ETC明細のダウンロードを開始
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status": "error", "message": "POSTリクエストで実行してください。"}`)
			return
		}
		var requestData requestData
		err = json.NewDecoder(r.Body).Decode(&requestData)

		//非同期で処理を実行
		go func() {
			defer func() {
				processingMutex.Lock()
				isProcessing = false
				processingMutex.Unlock()
			}()

			err := downloadEtcMeisai(requestData)
			handleError(err, "ETC明細のダウンロードに失敗しました")
			log.Println("ETC明細のダウンロードが完了しました。")
		}()

		// jsonでレスポンスを返す
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status": "success", "message": "ETC明細のダウンロードが開始されました。"}`)
	})
	port := setDefaultPort() // 環境変数 PORT が設定されていない場合は、デフォルトのポートを設定
	log.Printf("HTTPサーバーを :%s で起動します", port)
	err = http.ListenAndServe(":"+port, nil)
	handleErrorReturn(err, "HTTPサーバーの起動に失敗しました")
}

func setDefaultPort() string {
	// 環境変数 PORT が設定されていない場合は、デフォルトのポートを設定
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // デフォルトのポートを 8080 に設定
	}
	return port
}
