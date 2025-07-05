//go:build linux
// +build linux

package main

// Chromeブラウザの実行パス (Linux用)
const chromeExePath = "/usr/bin/google-chrome" // 一般的なLinuxのChromeパス
// または "/usr/bin/chromium" など、環境に合わせて変更

// ユーザーデータディレクトリのパス (Linux用)
const userDataDir = "/tmp/chrome-user-data" // Linuxの一般的な一時ディレクトリ
