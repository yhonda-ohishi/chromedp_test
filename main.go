package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
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
			fmt.Fprintf(w, `{"status": "error", "message": "既に処理が実行中です。"}`)
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

func downloadEtcMeisai(requestData requestData) error {

	// コンテキストの作成: タイムアウトとデバッグ出力を設定
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("headless", false), // headlessモードを無効にする
	)
	allocCtx, cancel1 := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel1()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var err error

	// 操作全体のタイムアウトを設定
	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	done := make(chan string, 1)

	chromedp.ListenTarget(ctx, func(v interface{}) {
		if ev, ok := v.(*browser.EventDownloadProgress); ok {
			completed := "(unknown)"
			if ev.TotalBytes != 0 {
				completed = fmt.Sprintf("%0.2f%%", ev.ReceivedBytes/ev.TotalBytes*100.0)
			}
			log.Printf("state: %s, completed: %s\n", ev.State.String(), completed)
			if ev.State == browser.DownloadProgressStateCompleted {
				done <- ev.GUID
				close(done)
			}
		}
	})

	// ダウンロードの保存先を設定

	if err = setDownloadBehavior(ctx, "download", "ohishiexp"); err != nil {
		return handleErrorReturn(err, "ダウンロードの保存先の設定に失敗しました")
	}

	// ダイアログの自動受け入れを設定
	setDialogBehavior(ctx)

	targetURL := "https://www2.etc-meisai.jp/etc/R?funccode=1013000000&nextfunc=1013000000" // スクレイピングしたいウェブサイトのURLに変更してください
	downloadPath := "download"                                                              // ダウンロードの保存先ディレクトリを指定
	log.Printf("URLにアクセス中: %s", targetURL)

	// ダウンロードするファイルの名前を指定
	// requestData.Data[0].RisLoginId = "ohishiexp" // ダウンロードするファイルの名前を指定
	// requestData.Data[1].RisLoginId = "ohishiexp1" // ダウンロードするファイルの名前を指定
	// ファイル名の配列を作成
	filenameArray := make([]string, len(requestData.Data))
	for i, request := range requestData.Data {
		filenameArray[i] = request.RisLoginId + ".csv"
	}
	for _, request := range requestData.Data {
		log.Printf("ログインID: %s, パスワード: %s", request.RisLoginId, request.RisPassword)

		// ブラウザ操作のタスクを実行
		if err = chromedp.Run(ctx, chromedp.Navigate(targetURL)); err != nil {
			log.Printf("URLへのアクセスに失敗しました: %v", err)
			time.Sleep(30 * time.Second) // URLへのアクセスに失敗した場合は、3秒待機して再試行
			return handleErrorReturn(err, "URLへのアクセスに失敗しました")
		}

		if err = inputSelectorWithName(ctx, "risLoginId", request.RisLoginId); err != nil {
			return handleErrorReturn(err, "risLoginIdの入力に失敗しました")
		}

		if err = inputSelectorWithName(ctx, "risPassword", request.RisPassword); err != nil {
			return handleErrorReturn(err, "risPasswordの入力に失敗しました")
		}

		if err = clickSelectorWithName(ctx, "focusTarget", 3); err != nil {
			return handleErrorReturn(err, "focusTargetのクリックに失敗しました")
		}

		var exists bool
		if exists, err = ExistsStringInContext(ctx, "hyojiCard"); err != nil {
			return handleErrorReturn(err, "指定された文字列の存在確認に失敗しました")
		}
		if exists {
			log.Println("指定された文字列がページ内に存在します。")
			if request.RisLoginId == "ohishiexp1" {
				// log.Println("body:")
				// //ctxのbodyをログに出力
				// log.Println("bodyの内容を出力します。")
				// var ctxBody string
				// err := chromedp.Run(ctx, chromedp.InnerHTML("body", &ctxBody, chromedp.ByQuery))
				// if err != nil {
				// 	log.Printf("bodyの取得に失敗しました: %v", err)
				// } else {
				// 	log.Printf("body内容: %s", ctxBody)
				// }

				// continue // ログインIDがohishiexp1の場合は処理をスキップ
			}
		} else {
			log.Println("指定された文字列がページ内に存在しません。")
			if err = ExecuteScript(ctx, `submitPage('frm','/etc/R?funccode=1014000000&nextfunc=1014000000')`); err != nil {
				return handleErrorReturn(err, "submitPageの実行に失敗しました")
			}
		}

		//2か月前の日付を作成
		lastMonth := time.Now().AddDate(0, -1, 0) // 2か月前の日付を取得
		//年を4桁で取得
		lastMonthYY := fmt.Sprintf("%04d", lastMonth.Year()) // 年を4桁で取得
		lastMonthMM := fmt.Sprintf("%02d", int(lastMonth.Month()))
		// last2monthDD := fmt.Sprintf("%02d", last2month.Day()) // 日を2桁で取得
		// 今日の日付を取得
		today := time.Now()
		todayYY := fmt.Sprintf("%04d", today.Year()) // 年を4桁で取得
		todayMM := fmt.Sprintf("%02d", int(today.Month()))
		todayDD := fmt.Sprintf("%02d", today.Day()) // 日を2桁で取得
		if err = selectSelectorWithName(ctx, "fromYYYY", lastMonthYY); err != nil {
			return handleErrorReturn(err, "fromYYYYのセレクタの選択に失敗しました")
		}

		if err = selectSelectorWithName(ctx, "fromMM", lastMonthMM); err != nil {
			return handleErrorReturn(err, "fromMMのセレクタの選択に失敗しました")
		}

		if err = selectSelectorWithName(ctx, "fromDD", "01"); err != nil {
			return handleErrorReturn(err, "fromDDのセレクタの選択に失敗しました")
		}

		if err = selectSelectorWithName(ctx, "toYYYY", todayYY); err != nil {
			return handleErrorReturn(err, "toYYYYのセレクタの選択に失敗しました")
		}
		if err = selectSelectorWithName(ctx, "toMM", todayMM); err != nil {
			return handleErrorReturn(err, "toMMのセレクタの選択に失敗しました")
		}
		if err = selectSelectorWithName(ctx, "toDD", todayDD); err != nil {
			return handleErrorReturn(err, "toDDのセレクタの選択に失敗しました")
		}

		if err = clickRadioButtonByNameByValue(ctx, "sokoKbn", 0); err != nil {
			return handleErrorReturn(err, "sokoKbnのラジオボタンのクリックに失敗しました")
		}
		if err = ExecuteScript(ctx, `allSelected('hyojiCard')`, 0); err != nil {
			return handleErrorReturn(err, "hyojiCardの選択に失敗しました")
		}
		if err = clickSelectorWithName(ctx, "focusTarget_Save", 5); err != nil {
			return handleErrorReturn(err, "focusTarget_Saveのクリックに失敗しました")
		}
		if err = clickButtonByNameByWaitNavigation(ctx, "focusTarget", 5); err != nil {
			return handleErrorReturn(err, "focusTargetのクリックに失敗しました")
		}

		initialCount, err := ReadDirCount(downloadPath)
		if err != nil {
			return handleErrorReturn(err)
		}
		log.Printf("初期のファイル数: %d", initialCount)

		time.Sleep(2 * time.Second) // ダウンロードが完了するまで待機
		if err = clickInputByValue(ctx, "利用明細ＣＳＶ出力"); err != nil {
			return handleErrorReturn(err, "利用明細ＣＳＶ出力のクリックに失敗しました")
		}

		if err = takeScreenshot(ctx, "google_chromedp_search.png"); err != nil {
			return handleErrorReturn(err, "スクリーンショットの保存に失敗しました")
		}

		// fileName := "ohishiexp.csv" // ダウンロードするファイルの名前を指定
		// ダウンロードが完了するまで待機
		// ダウンロードの進行状況を監視
		//download folderの中にあるファイルを監視
		log.Println("ダウンロードを待機中...")

		if err = watchDownload("download", initialCount, 30); err != nil {
			return handleErrorReturn(err, "ダウンロードの監視中にエラーが発生しました")
		}

		if err = changeDownloadedFileName("download", filenameArray...); err != nil {
			return handleErrorReturn(err, "ダウンロードしたファイルの名前の変更に失敗しました")
		}
		log.Println("終了しました。")

		log.Printf("submitPageの実行を開始します")
		if err = ExecuteScript(ctx, `submitPage('frm','/etc/R?funccode=1021000000&nextfunc=1021000000')`); err != nil {
			return handleErrorReturn(err, "submitPageの実行に失敗しました")
		}
		time.Sleep(3 * time.Second) // 1秒待機してから
		log.Println("終了しました。")

	}
	return nil
}

func watchDownload(downloadPath string, initialCount int, timeout ...int) error {
	// ダウンロードの保存先を指定

	log.Printf("ダウンロードの保存先: %s", downloadPath)
	// ダウンロードが完了するまで待機
	timeoutDuration := 0 * time.Second // デフォルトのタイムアウト
	if len(timeout) > 0 {
		timeoutDuration = time.Duration(timeout[0]) * time.Second // タイムアウトの設定
	}

	currentDuration := 0 * time.Second // 現在の経過時間

	for {
		count, err := ReadDirCount(downloadPath)
		handleError(err)

		if initialCount < count {
			// ダウンロードが完了した場合はループを抜ける
			return nil
		}
		if currentDuration >= timeoutDuration {
			log.Println("ダウンロードがタイムアウトしました。")
			return fmt.Errorf("ダウンロードがタイムアウトしました")
		}
		time.Sleep(1 * time.Second)        // 1秒待機して再チェック
		currentDuration += 1 * time.Second // 現在の経過時間を更新
		log.Printf("現在の経過時間: %s, タイムアウト: %s count: %d", currentDuration, timeoutDuration, count)
	}

}
func changeDownloadedFileName(downloadPath string, newNames ...string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Println("現在のディレクトリの取得に失敗:", err)
		currentDir = "." // フォールバック
	}
	currentDir = filepath.Join(currentDir, downloadPath) // ダウンロードの保存先を指定

	// ダウンロードの保存先のディレクトリを確認
	if _, err := os.Stat(currentDir); os.IsNotExist(err) {
		log.Println("ダウンロードの保存先のディレクトリが存在しません:", currentDir)
		return fmt.Errorf("ダウンロードの保存先のディレクトリが存在しません: %s", currentDir)
	}
	// ダウンロードの保存先のディレクトリ内のファイルを取得
	files, err := os.ReadDir(currentDir)
	if err != nil {
		return err
	}
	files = sortFilesByModTime(files) // ファイルを更新日時でソート

	//filesの中からfileの名前がnewNamesに含まれるものを探す
	count := 0
	changed := false
	for _, file := range files {
		// if file.IsDir() {
		// 	continue // ディレクトリはスキップ
		// }
		oldName := file.Name()
		// newNamesに含まれていなければ、名前を変更
		stringIn := stringInSlice(oldName, newNames...)
		if !stringIn {
			//oldNameがnewNamesに含まれていなければ、名前を変更
			oldFilePath := filepath.Join(currentDir, oldName)
			newFilePath := filepath.Join(currentDir, newNames[count])
			err := os.Rename(oldFilePath, newFilePath)
			handleError(err, "ファイル名の変更に失敗")
			changed = true
			log.Printf("ファイル名を変更しました: %s -> %s", oldName, newNames[count])
			break // 最初のファイル名を変更したらループを抜ける

		} else {
			count++ // newNamesに含まれている場合はカウントを増やす
		}
	}
	if !changed {
		log.Println("指定されたファイル名がすでに存在します。変更は行われませんでした。")
		return fmt.Errorf("指定されたファイル名がすでに存在します: %s", newNames)
	}
	return nil
}

func sortFilesByModTime(files []os.DirEntry) []os.DirEntry {

	// Get file info with creation time and sort
	type fileWithTime struct {
		info    os.DirEntry
		modTime time.Time
	}

	var filesWithTime []fileWithTime
	for _, file := range files {
		if !file.IsDir() {
			fileInfo, err := file.Info()
			if err != nil {
				continue
			}
			filesWithTime = append(filesWithTime, fileWithTime{
				info:    file,
				modTime: fileInfo.ModTime(),
			})
		}
	}

	// Sort by modification time (ascending)
	sort.Slice(filesWithTime, func(i, j int) bool {
		return filesWithTime[i].modTime.Before(filesWithTime[j].modTime)
	})

	// Extract sorted files
	files = make([]os.DirEntry, len(filesWithTime))
	for i, f := range filesWithTime {
		files[i] = f.info
	}
	return files
}

func stringInSlice(str string, slice ...string) bool {
	if len(slice) == 0 {
		return false // スライスが空の場合はfalseを返す
	}
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func handleError(err error, message ...string) {
	if err != nil {
		if len(message) == 0 {
			message = append(message, "エラーが発生しました")
		}
		log.Printf("%s: %v", message[0], err)
		// ここで必要に応じて追加のエラー処理（例: ユーザーへの通知、リトライ、アプリケーションの終了など）
	}
}
func handleErrorReturn(err error, message ...string) error {
	if err != nil {
		if len(message) == 0 {
			message = append(message, "エラーが発生しました")
		}
		log.Printf("%s: %v", message[0], err)
		// ここで必要に応じて追加のエラー処理（例: ユーザーへの通知、リトライ、アプリケーションの終了など）
		return err
	}
	return nil
}
func ReadDirCount(downloadPath string) (int, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Println("現在のディレクトリの取得に失敗:", err)
		currentDir = "." // フォールバック
	}
	currentDir = currentDir + "\\" + downloadPath // ダウンロードの保存先を指定

	files, err := os.ReadDir(currentDir)
	if err != nil {
		log.Println("ディレクトリの読み取りに失敗:", err)
		return 0, err
	}

	count := 0
	for _, file := range files {
		if !file.IsDir() { // ディレクトリはカウントしない
			fileName := file.Name()
			log.Printf("ファイル名: %s", fileName)
			if strings.HasSuffix(strings.ToLower(fileName), ".csv") {
				count++
			}
		}
	}

	return count, nil
}

func setDialogBehavior(ctx context.Context) {

	// 現在のディレクトリを取得
	// currentDir, err := os.Getwd()
	// if err != nil {
	// 	log.Println("現在のディレクトリの取得に失敗:", err)
	// 	currentDir = "." // フォールバック
	// }
	// ダウンロードファイルの名前を指定
	// filename := "ohishiexp.csv" // ダウンロードするファイルの名前を指定
	// ダイアログが表示された場合に自動的に受け入れる
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			log.Println("Console API called:", ev)
		case *page.EventJavascriptDialogOpening:
			log.Println("ダイアログが開かれました:", ev.Message)
			// confirmダイアログやalertダイアログが開かれた場合に自動的に受け入れる
			log.Println("ダイアログの種類:", ev.Type)

			// ダイアログを自動的に受け入れる
			go func() {
				if err := chromedp.Run(ctx, page.HandleJavaScriptDialog(true)); err != nil {
					log.Println("ダイアログの受け入れに失敗:", err)
				} else {
					log.Println("ダイアログを受け入れました:", ev.Message)
				}
			}()
		}
	})

}

func setDownloadBehavior(ctx context.Context, downloadPath string, filename string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Println("現在のディレクトリの取得に失敗:", err)
		currentDir = "." // フォールバック
	}

	//mkdir
	if err := os.MkdirAll(downloadPath, 0o755); err != nil {
		log.Println("ダウンロードの保存先のディレクトリの作成に失敗:", err)
		return err
	}

	currentDir = currentDir + "\\" + downloadPath // ダウンロードの保存先を指定

	//folderの中のfileを削除
	files, err := os.ReadDir(currentDir)
	if err != nil {
		log.Println("ディレクトリの読み取りに失敗:", err)
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			continue // ディレクトリはスキップ
		}
		if err := os.Remove(currentDir + "\\" + file.Name()); err != nil {
			log.Println("ファイルの削除に失敗:", err)
			return err
		}
		log.Println("ファイルを削除しました:", file.Name())
	}

	// if err := chromedp.Run(ctx, browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).WithDownloadPath(currentDir).WithEventsEnabled(true)); err != nil {
	// 	log.Println("ダウンロードの保存先の設定に失敗:", err)
	// } else {
	// 	log.Printf("ダウンロードの保存先を設定しました: %s", currentDir)
	// }
	// ダウンロードするファイルの名前を指定
	if err := chromedp.Run(ctx, browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).
		// ダウンロードの保存先を指定
		// ここで指定したパスにダウンロードされます
		WithDownloadPath(currentDir).
		WithEventsEnabled(true)); err != nil {
		log.Println("ダウンロードの保存先の設定に失敗:", err)
		return err
	}

	return nil
}

func clickInputByValue(ctx context.Context, value string) error {
	// ページにアクセス
	// name=["risLoginId"] の要素を待機
	log.Printf("指定されたvalue属性を持つ要素をクリック: %s", value)
	time.Sleep(5 * time.Second) // 1秒待機してから要素を待機
	if err := chromedp.Run(ctx, chromedp.WaitVisible(`[value="`+value+`"]`, chromedp.ByQuery)); err != nil {
		log.Printf("指定されたvalue属性を持つ要素が見つかりません: %s", value)
		return err
	}
	// 指定されたname属性とvalue属性を持つinput要素をクリック
	if err := chromedp.Run(ctx, chromedp.Click(`[value="`+value+`"]`, chromedp.NodeVisible, chromedp.ByQuery)); err != nil {
		log.Printf("指定されたvalue属性を持つ要素のクリックに失敗しました: %s", value)
		return err
	}

	return nil
}

func clickButtonByNameByWaitNavigation(ctx context.Context, name string, value ...int) error {
	// ページにアクセス
	// name=["risLoginId"] の要素を待機
	if err := chromedp.Run(ctx, chromedp.WaitVisible(`[name="`+name+`"]`, chromedp.ByQuery)); err != nil {
		return err
	}
	// 指定されたname属性とvalue属性を持つラジオボタンをクリック
	if err := chromedp.Run(ctx, chromedp.Click(`[name="`+name+`"]`, chromedp.NodeVisible, chromedp.ByQuery)); err != nil {
		return err
	}

	time.Sleep(3 * time.Second) // 1秒待機してからナビゲーションを待機
	//timeoutを設定
	//defaltは0秒
	waitforNavigation(ctx, value...)

	return nil
}

func waitforNavigation(ctx context.Context, timeoutSeconds ...int) error {
	// ページのナビゲーションを待機
	//urlが変わるまで待機
	// 1秒ごとにURLをチェックして、変化があれば終了

	log.Println("ページのナビゲーションを待機しています...")
	timeout := 0 * time.Second // デフォルト値
	if len(timeoutSeconds) > 0 {
		timeout = time.Duration(timeoutSeconds[0]) * time.Second
	} // タイムアウトの設定

	var currentURL string
	var currentDuration time.Duration
	for {
		// timout 以上であれば、loopを抜ける
		if timeout > 0 && currentDuration >= timeout {
			log.Println("タイムアウトしました。")
			break
		}
		if timeout == 0 {
			// タイムアウトが設定されていない場合は、loopを抜ける
			log.Println("タイムアウトが設定されていません。ループを抜けます。")
			break

		}
		if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
			return err
		}
		if currentURL != "https://www2.etc-meisai.jp/etc/R?funccode=1013000000&nextfunc=1013000000" {
			// URLが変わった場合はループを抜ける
			log.Printf("URLが変わりました: %s", currentURL)
			//描画が完了するまで待機
			if err := chromedp.Run(ctx, chromedp.WaitVisible("body", chromedp.ByQuery)); err != nil {
				return err
			}
			log.Println("ページの描画が完了しました。")
			break
		}

		time.Sleep(1 * time.Second) // 1秒待機して再チェック
		currentDuration += 1 * time.Second
	}

	return nil
}

func clickRadioButtonByNameByValue(ctx context.Context, name string, value int) error {
	// name=["risLoginId"] の要素を待機
	if err := chromedp.Run(ctx, chromedp.WaitVisible(`[name="`+name+`"][value="`+strconv.Itoa(value)+`"]`, chromedp.ByQuery)); err != nil {
		return err
	}
	// 指定されたname属性とvalue属性を持つラジオボタンをクリック
	if err := chromedp.Run(ctx, chromedp.Click(`[name="`+name+`"][value="`+strconv.Itoa(value)+`"]`, chromedp.NodeVisible, chromedp.ByQuery)); err != nil {
		return err
	}

	return nil
}

func selectSelectorWithName(ctx context.Context, name string, value string) error {
	// ページにアクセス
	// name=["risLoginId"] の要素を待機
	if err := chromedp.Run(ctx, chromedp.WaitVisible(`[name="`+name+`"]`, chromedp.ByQuery)); err != nil {
		return err
	}
	// 指定されたname属性を持つselect要素を選択
	if err := chromedp.Run(ctx, chromedp.SetValue(`[name="`+name+`"]`, value, chromedp.ByQuery)); err != nil {
		return err
	}

	return nil
}

func ExecuteScript(ctx context.Context, script string, timeout ...int) error {
	// 指定されたJavaScriptを実行
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, nil)); err != nil {
		log.Fatal("fatal error:", err)
		return err
	}
	// タイムアウトを設定
	waitforNavigation(ctx, timeout...)
	return nil
}

func ExistsStringInContext(ctx context.Context, str string) (bool, error) {
	var ctxBody string
	if err := chromedp.Run(ctx, chromedp.InnerHTML("body", &ctxBody, chromedp.ByQuery)); err != nil {
		log.Printf("bodyの取得に失敗しました: %v", err)
		return false, err // bodyの取得に失敗した場合はfalseを返す
	} else {
		// log.Printf("body内容: %s", ctxBody)
	}
	return strings.Contains(ctxBody, str), nil // bodyの内容に指定された文字列が含まれているか確認
}

func takeScreenshot(ctx context.Context, filename string) error {
	var buf []byte
	// スクリーンショットを取得
	if err := chromedp.Run(ctx, chromedp.Screenshot("body", &buf, chromedp.NodeVisible, chromedp.ByQuery)); err != nil {
		return err
	}
	// スクリーンショットをファイルに保存
	if err := os.WriteFile(filename, buf, 0o644); err != nil {

		log.Fatal(err)
		return err
	}
	return nil
}

func inputSelectorWithName(ctx context.Context, name string, input string) error {
	// ページにアクセス
	// name=["risLoginId"] の要素を待機
	if err := chromedp.Run(ctx, chromedp.WaitVisible(`[name="`+name+`"]`, chromedp.ByQuery)); err != nil {
		return err
	}
	// 指定されたname属性を持つinput要素にテキストを入力
	if err := chromedp.Run(ctx, chromedp.SendKeys(`[name="`+name+`"]`, input, chromedp.ByQuery)); err != nil {
		return err
	}

	return nil
}

func clickSelectorWithName(ctx context.Context, name string, timeoutSeconds ...int) error {
	// タイムアウト設定

	// コンテキストにタイムアウトを設定

	// 指定されたname属性を持つ要素を待機
	if err := chromedp.Run(ctx, chromedp.WaitVisible(`[name="`+name+`"]`, chromedp.ByQuery)); err != nil {
		return err
	}
	// 指定されたname属性を持つ要素をクリック
	if err := chromedp.Run(ctx, chromedp.Click(`[name="`+name+`"]`, chromedp.NodeVisible, chromedp.ByQuery)); err != nil {
		return err
	}

	timeout := 0 * time.Second // デフォルト値
	if len(timeoutSeconds) > 0 {
		timeout = time.Duration(timeoutSeconds[0]) * time.Second
	}
	//timeout 秒数待機　sleep

	//3秒待機
	time.Sleep(timeout)

	return nil
}
