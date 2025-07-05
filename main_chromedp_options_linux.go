//go:build linux
// +build linux

package main

import (
	"github.com/chromedp/chromedp"
)

// getOSSpecificChromeOptions はLinux固有のchromedp.ExecPathオプションを返す
// Linuxでは chromedp.ExecPath は通常不要なため、空のオプションを返す
func getOSSpecificChromeOptions() []chromedp.ExecAllocatorOption {
	return []chromedp.ExecAllocatorOption{
		// 空のオプションを返すことで、Linuxビルド時には chromedp.ExecPath が含まれない
	}
}
