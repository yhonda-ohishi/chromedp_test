//go:build windows
// +build windows

package main

import (
	"github.com/chromedp/chromedp"
)

// getOSSpecificChromeOptions はWindows固有のchromedp.ExecPathオプションを返す
func getOSSpecificChromeOptions() []chromedp.ExecAllocatorOption {
	return []chromedp.ExecAllocatorOption{
		chromedp.ExecPath(chromeExePath),            // Windowsでのみコンパイルされる
		chromedp.Flag("user-data-dir", userDataDir), // または、C:\Users\[あなたのユーザー名]\AppData\Local\Temp\my-chromedp-data のように指定
	}
}
