//go:build windows

package main

// 설교 도우미 AI(Claude · ChatGPT · 제미나이)의 API 키는 Windows 자격 증명 관리자(제어판 › 자격 증명 관리자 › Windows 자격 증명)에 저장해요.

import (
	"strings"

	"github.com/danieljoos/wincred"
)

func winCredGet(name string) string {
	c, err := wincred.GetGenericCredential(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(c.CredentialBlob))
}

func winCredSet(name, key string) error {
	c := wincred.NewGenericCredential(name)
	c.UserName = "SermonROAD"
	c.CredentialBlob = []byte(key)
	c.Persist = wincred.PersistLocalMachine
	return c.Write()
}

func winCredDelete(name string) {
	if c, err := wincred.GetGenericCredential(name); err == nil {
		c.Delete()
	}
}
