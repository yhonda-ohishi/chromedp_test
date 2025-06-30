package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func main() {
	// コンテキストの作成: タイムアウトとデバッグ出力を設定
	// chromedp.WithDebugf(log.Printf) は、chromedpの内部ログを表示し、デバッグに役立ちます
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("headless", false), // headlessモードを無効にする
	)
	allocCtx, cancel1 := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel1()

	ctx, cancel := chromedp.NewContext(allocCtx) // chromedp.WithDebugf(log.Printf),

	defer cancel()

	// 操作全体のタイムアウトを設定
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
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
	if err := setDownloadBehavior(ctx, "download", "ohishiexp"); err != nil {
		log.Fatal(err)
		return
	}

	// ダイアログの自動受け入れを設定
	setDialogBehavior(ctx)

	targetURL := "https://www2.etc-meisai.jp/etc/R?funccode=1013000000&nextfunc=1013000000" // スクレイピングしたいウェブサイトのURLに変更してください
	log.Printf("URLにアクセス中: %s", targetURL)

	// ブラウザ操作のタスクを実行
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(targetURL), // 指定したURLにナビゲート
	}); err != nil {
		log.Fatal(err)
		return
	}
	if err := inputSelectorWithName(ctx, "risLoginId", "ohishiexp"); err != nil {
		log.Fatal(err)
		return
	}
	if err := inputSelectorWithName(ctx, "risPassword", "ohishi11"); err != nil {
		log.Fatal(err)
		return
	}
	if err := clickSelectorWithName(ctx, "focusTarget", 3); err != nil {
		log.Fatal(err)
		return
	}

	var exists bool
	var err error
	if exists, err = ExistsStringinContext(ctx, "1014000000"); err != nil {
		log.Fatal(err)
		return
	}
	if exists {
		log.Println("指定された文字列がページ内に存在します。")
		if err := ExecuteScript(ctx, `submitPage('frm','/etc/R?funccode=1014000000&nextfunc=1014000000'`); err != nil {
			log.Fatal(err)
			return
		}

	} else {
		log.Println("指定された文字列がページ内に存在しません。")

	}

	//2か月前の日付を作成
	lastmonth := time.Now().AddDate(0, -1, 0) // 2か月前の日付を取得
	//年を4桁で取得
	lastmonthYY := fmt.Sprintf("%04d", lastmonth.Year()) // 年を4桁で取得
	lastmonthMM := fmt.Sprintf("%02d", int(lastmonth.Month()))
	// last2monthDD := fmt.Sprintf("%02d", last2month.Day()) // 日を2桁で取得
	// 今日の日付を取得
	today := time.Now()
	todayYY := fmt.Sprintf("%04d", today.Year()) // 年を4桁で取得
	todayMM := fmt.Sprintf("%02d", int(today.Month()))
	todayDD := fmt.Sprintf("%02d", today.Day()) // 日を2桁で取得
	if err := selectSlectorwithName(ctx, "fromYYYY", lastmonthYY); err != nil {
		log.Fatal(err)
		return
	}
	if err := selectSlectorwithName(ctx, "fromMM", lastmonthMM); err != nil {
		log.Fatal(err)
		return
	}
	if err := selectSlectorwithName(ctx, "fromDD", "01"); err != nil {
		log.Fatal(err)
		return
	}
	if err := selectSlectorwithName(ctx, "toYYYY", todayYY); err != nil {
		log.Fatal(err)
		return
	}
	if err := selectSlectorwithName(ctx, "toMM", todayMM); err != nil {
		log.Fatal(err)
		return
	}
	if err := selectSlectorwithName(ctx, "toDD", todayDD); err != nil {
		log.Fatal(err)
		return
	}

	if err := clickRadioButtonByNameByValue(ctx, "sokoKbn", 0); err != nil {
		log.Fatal(err)
		return
	}
	if err := ExecuteScript(ctx, `allSelected('hyojiCard')`, 0); err != nil {
		log.Fatal(err)
		return
	}
	if err := clickSelectorWithName(ctx, "focusTarget_Save", 5); err != nil {
		log.Fatal(err)
		return
	}
	if err := clickButtonByNameByWaitNavigation(ctx, "focusTarget", 5); err != nil {
		log.Fatal(err)
		return
	}

	time.Sleep(2 * time.Second) // ダウンロードが完了するまで待機
	if err := clickInputByValeue(ctx, "利用明細ＣＳＶ出力"); err != nil {
		log.Fatal(err)
		return
	}

	if err := takeScreenshot(ctx, "google_chromedp_search.png"); err != nil {
		return
	}

	log.Println("スクリーンショットが保存されました: google_chromedp_search.png")
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
	if err := chromedp.Run(ctx, browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
		// ダウンロードの保存先を指定
		// ここで指定したパスにダウンロードされます
		WithDownloadPath(currentDir).
		WithEventsEnabled(true)); err != nil {
		log.Println("ダウンロードの保存先の設定に失敗:", err)
		return err
	}

	return nil
}

func clickInputByValeue(ctx context.Context, value string) error {
	// ページにアクセス
	// name=["risLoginId"] の要素を待機
	log.Printf("指定されたvalue属性を持つ要素をクリック: %s", value)
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

func selectSlectorwithName(ctx context.Context, name string, value string) error {
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

func ExistsStringinContext(ctx context.Context, str string) (bool, error) {
	var exists bool
	// 指定された文字列がページ内に存在するか確認
	if err := chromedp.Run(ctx, chromedp.Evaluate(`document.body.innerText.includes("`+str+`")`, &exists)); err != nil {
		log.Fatal(err)
		return false, err
	}
	return exists, nil
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
