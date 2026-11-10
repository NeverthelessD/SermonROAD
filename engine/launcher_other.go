//go:build !windows

package main

// 맥 · 리눅스에는 실행기가 없어요 (맥은 Dock 앱 껍데기가 엔진을 켜요).
func launcherMode() bool { return false }
func runLauncher()       {}

// chooseFolderNative: 맥은 앱 껍데기가 폴더 고르기 창을 띄워요.
func chooseFolderNative(prompt, start string) (string, bool, error) { return "", false, nil }
