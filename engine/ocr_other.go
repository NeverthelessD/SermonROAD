//go:build !windows && !darwin

package main

// 리눅스(시험용)에는 글자 인식이 없어요. 글자가 있는 PDF 만 읽어요.

import "image"

type ocrSession struct{}

func newOCRSession() *ocrSession                    { return &ocrSession{} }
func (s *ocrSession) ok() bool                      { return false }
func (s *ocrSession) recognize(img image.Image) string { return "" }
func (s *ocrSession) close()                        {}
