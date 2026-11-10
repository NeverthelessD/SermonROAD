//go:build windows

package main

// 폴더 고르기 (Windows): 화면에서 /api/choose-folder 를 부르면 Windows 의 폴더 선택 창을 띄워요.
// 화면(Edge · Chrome 앱 창)이 맨 앞에 있을 때 부르니까, 그 창을 주인으로 삼아 선택 창이 앞에 뜨게 해요.

import (
	"errors"
	"runtime"
	"sync"
	"syscall"

	"github.com/harry1453/go-common-file-dialog/cfd"
)

var (
	procGetForegroundWindow = syscall.NewLazyDLL("user32.dll").NewProc("GetForegroundWindow")
	folderMu                sync.Mutex
)

func chooseFolderNative(prompt, start string) (string, bool, error) {
	if !folderMu.TryLock() {
		return "", false, errors.New("폴더 고르기 창이 이미 열려 있어요")
	}
	defer folderMu.Unlock()
	owner, _, _ := procGetForegroundWindow.Call()
	type res struct {
		path string
		err  error
	}
	ch := make(chan res, 1)
	go func() {
		runtime.LockOSThread() // COM 창은 한 스레드에서만
		defer runtime.UnlockOSThread()
		if prompt == "" {
			prompt = "폴더 고르기"
		}
		cfg := cfd.DialogConfig{Title: prompt, Role: "SermonROADFolder", ParentWindowHandle: owner}
		if dirExists(start) {
			cfg.Folder = start
		}
		dlg, err := cfd.NewSelectFolderDialog(cfg)
		if err != nil {
			ch <- res{"", err}
			return
		}
		defer dlg.Release()
		p, err := dlg.ShowAndGetResult()
		ch <- res{p, err}
	}()
	r := <-ch
	if errors.Is(r.err, cfd.ErrorCancelled) {
		return "", true, nil
	}
	if r.err != nil {
		return "", false, r.err
	}
	return r.path, true, nil
}
