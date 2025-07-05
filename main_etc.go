package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

func downloadEtcMeisai(requestData requestData) error {
	log.Println("ChromeDP初期化を開始します...")

	// Cloudflare Containers環境に特化したChrome設定
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromeExePath), // ここにChromeの実行パスを追加！

		// 基本的な設定
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),

		// パフォーマンス向上
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-plugins", true),
		chromedp.Flag("disable-images", true),
		chromedp.Flag("disable-javascript", false), // JavaScriptは有効にする

		// 安定性向上
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-features", "TranslateUI,BlinkGenPropertyTrees"),

		// ネットワーク関連
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("ignore-ssl-errors", true),
		chromedp.Flag("allow-running-insecure-content", true),

		// デバッグ設定（簡素化）
		chromedp.Flag("remote-debugging-port", "9222"),
		chromedp.Flag("remote-debugging-address", "127.0.0.1"),

		// ユーザーデータディレクトリ
		// chromedp.Flag("user-data-dir", "/tmp/chrome-user-data"),
		chromedp.Flag("user-data-dir", userDataDir), // または、C:\Users\[あなたのユーザー名]\AppData\Local\Temp\my-chromedp-data のように指定

		// ウィンドウサイズ
		chromedp.WindowSize(1920, 1080),
	)

	// ExecAllocatorを作成
	allocCtx, cancel1 := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel1()

	// コンテキストを作成
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// より長いタイムアウトを設定
	ctx, cancel = context.WithTimeout(ctx, 180*time.Second) // 3分に設定
	defer cancel()

	// 簡単な接続テスト
	log.Println("ChromeDP接続テストを開始します...")
	err := chromedp.Run(ctx, chromedp.Navigate("about:blank"))
	if err != nil {
		return fmt.Errorf("ChromeDP接続に失敗しました: %v", err)
	}
	log.Println("ChromeDP接続成功しました")
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
		log.Printf("ログインID: %s, パスワード: ***", request.RisLoginId)

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
