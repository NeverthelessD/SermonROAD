// SermonROAD 서재 PDF 글자 읽기 도구 (앱 안 Contents/Resources/sermonroad-pdftext)
// made by. Nevertheless_D
//
//   sermonroad-pdftext <PDF> <결과.json> [--no-ocr]
//   PDFKit 으로 쪽마다 글자를 읽고, 글자가 없는 쪽(스캔한 책)은 맥의 글자 인식(Vision, 한국어 · 영어)으로 읽어요.
//   앱이 아니라 작은 명령 도구라서 Dock 에 아이콘이 생기지 않아요. 엔진이 서재를 읽을 때 불러요.

import CoreGraphics
import Foundation
import PDFKit
import Vision


enum PDFText {
    /// 쪽마다 글자를 읽어 JSON({"pages": [...], "ocr": [쪽 번호…]})으로 저장해요.
    /// 진행 상황은 표준 출력에 "PAGE 3/120" 처럼 한 줄씩 알려요 (엔진이 읽어서 화면에 보여 줘요).
    static func run(_ src: String, _ out: String, ocr allowOCR: Bool) -> Int32 {
        guard let doc = PDFDocument(url: URL(fileURLWithPath: src)) else {
            FileHandle.standardError.write("PDF를 열지 못했어요\n".data(using: .utf8)!)
            return 2
        }
        if doc.isLocked {
            FileHandle.standardError.write("암호가 걸린 PDF예요\n".data(using: .utf8)!)
            return 3
        }
        var pages: [String] = []
        var ocrPages: [Int] = []
        let n = doc.pageCount
        for i in 0..<n {
            autoreleasepool {
                var text = ""
                if let page = doc.page(at: i) {
                    text = page.string ?? ""
                    let meaningful = text.unicodeScalars.filter { CharacterSet.letters.contains($0) }.count
                    if allowOCR && meaningful < 12, let t = recognize(page), !t.isEmpty {
                        text = t
                        ocrPages.append(i + 1)
                    }
                }
                pages.append(text)
            }
            print("PAGE \(i + 1)/\(n)")
            fflush(stdout)
        }
        let obj: [String: Any] = ["pages": pages, "ocr": ocrPages, "title": (doc.documentAttributes?[PDFDocumentAttribute.titleAttribute] as? String) ?? ""]
        guard let data = try? JSONSerialization.data(withJSONObject: obj, options: []) else { return 4 }
        do {
            try data.write(to: URL(fileURLWithPath: out), options: .atomic)
        } catch {
            return 5
        }
        return 0
    }

    /// 글자가 없는 쪽(스캔한 책)을 그림으로 그려서 맥의 글자 인식으로 읽어요 (한국어 · 영어)
    static func recognize(_ page: PDFPage) -> String? {
        // 그리기는 AppKit 을 거치지 않는 CoreGraphics 로 (창 서버에 붙지 않아서 Dock 에 아이콘이 생기지 않아요)
        guard let cgPage = page.pageRef else { return nil }
        let box = cgPage.getBoxRect(.mediaBox)
        let scale: CGFloat = 2.2
        let w = Int(box.width * scale), h = Int(box.height * scale)
        guard w > 10, h > 10, w < 12000, h < 12000,
              let cs = CGColorSpace(name: CGColorSpace.sRGB),
              let ctx = CGContext(data: nil, width: w, height: h, bitsPerComponent: 8, bytesPerRow: 0, space: cs,
                                  bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue) else { return nil }
        ctx.setFillColor(CGColor(gray: 1, alpha: 1))
        ctx.fill(CGRect(x: 0, y: 0, width: w, height: h))
        ctx.scaleBy(x: scale, y: scale)
        ctx.translateBy(x: -box.origin.x, y: -box.origin.y)
        ctx.drawPDFPage(cgPage)
        guard let img = ctx.makeImage() else { return nil }
        let req = VNRecognizeTextRequest()
        req.recognitionLevel = .accurate
        req.usesLanguageCorrection = true
        if #available(macOS 13.0, *) {
            req.revision = VNRecognizeTextRequestRevision3
        }
        let langs = ["ko-KR", "en-US"]
        if let ok = try? req.supportedRecognitionLanguages() {
            req.recognitionLanguages = langs.filter { ok.contains($0) }
        }
        let handler = VNImageRequestHandler(cgImage: img, options: [:])
        do {
            try handler.perform([req])
        } catch {
            return nil
        }
        let obs = req.results ?? []
        // 위에서 아래로, 왼쪽에서 오른쪽으로 (Vision 좌표는 아래가 0)
        let lines = obs.sorted { a, b in
            if abs(a.boundingBox.midY - b.boundingBox.midY) > 0.008 { return a.boundingBox.midY > b.boundingBox.midY }
            return a.boundingBox.minX < b.boundingBox.minX
        }.compactMap { $0.topCandidates(1).first?.string }
        return lines.joined(separator: "\n")
    }
}


let args = CommandLine.arguments
guard args.count >= 3 else {
    FileHandle.standardError.write("사용법: sermonroad-pdftext <PDF> <결과.json> [--no-ocr]\n".data(using: .utf8)!)
    exit(64)
}
exit(PDFText.run(args[1], args[2], ocr: !args.contains("--no-ocr")))
