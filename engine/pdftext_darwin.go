//go:build darwin

package main

// 맥은 앱 안의 sermonroad-pdftext(PDFKit + Vision)가 PDF 를 읽어요.
func pdfTextMain(args []string) int { return 1 }
