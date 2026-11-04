package main

// 성경 66권 표 · 본문 표기 읽기
//   "롬 8:28-30", "로마서 8장 28~30절", "시편 23편", "롬 8:28-9:3", "Romans 8:28" … 을 (책, 장, 절) 범위로 바꿔요.
//   절이 0 이면 장 전체예요.

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Book struct {
	N        int      `json:"n"`     // 1~66
	Code     string   `json:"code"`  // HOLYBIBLE 주소용 (OLD01 … NEW27)
	Name     string   `json:"name"`  // 로마서
	Abbr     string   `json:"abbr"`  // 롬
	Chapters int      `json:"ch"`    // 장 수
	Group    string   `json:"group"` // 율법서 …
	OT       bool     `json:"ot"`
	alias    []string // 다른 이름 (한국어 · 영어)
}

var books = func() []*Book {
	raw := []struct {
		name, abbr string
		ch         int
		alias      string
	}{
		{"창세기", "창", 50, "Genesis|Gen|Gn"}, {"출애굽기", "출", 40, "Exodus|Exod|Ex"}, {"레위기", "레", 27, "Leviticus|Lev"},
		{"민수기", "민", 36, "Numbers|Num"}, {"신명기", "신", 34, "Deuteronomy|Deut|Dt"},
		{"여호수아", "수", 24, "Joshua|Josh|여호수아서"}, {"사사기", "삿", 21, "Judges|Judg"}, {"룻기", "룻", 4, "Ruth"},
		{"사무엘상", "삼상", 31, "1 Samuel|1Samuel|1 Sam|1Sam|I Samuel"}, {"사무엘하", "삼하", 24, "2 Samuel|2Samuel|2 Sam|2Sam|II Samuel"},
		{"열왕기상", "왕상", 22, "1 Kings|1Kings|1 Kgs|1Kgs|I Kings"}, {"열왕기하", "왕하", 25, "2 Kings|2Kings|2 Kgs|2Kgs|II Kings"},
		{"역대상", "대상", 29, "1 Chronicles|1Chronicles|1 Chr|1Chr|I Chronicles"}, {"역대하", "대하", 36, "2 Chronicles|2Chronicles|2 Chr|2Chr|II Chronicles"},
		{"에스라", "스", 10, "Ezra|에스라서"}, {"느헤미야", "느", 13, "Nehemiah|Neh|느헤미야서"}, {"에스더", "에", 10, "Esther|Esth|에스더서"},
		{"욥기", "욥", 42, "Job"}, {"시편", "시", 150, "Psalms|Psalm|Pss|Ps|Psa"}, {"잠언", "잠", 31, "Proverbs|Prov|Pr"},
		{"전도서", "전", 12, "Ecclesiastes|Eccl|Ecc|Qoh"}, {"아가", "아", 8, "Song of Songs|Song of Solomon|Song|아가서"},
		{"이사야", "사", 66, "Isaiah|Isa|이사야서"}, {"예레미야", "렘", 52, "Jeremiah|Jer|예레미야서"}, {"예레미야애가", "애", 5, "Lamentations|Lam|애가"},
		{"에스겔", "겔", 48, "Ezekiel|Ezek|Ezk|에스겔서"}, {"다니엘", "단", 12, "Daniel|Dan|다니엘서"},
		{"호세아", "호", 14, "Hosea|Hos|호세아서"}, {"요엘", "욜", 3, "Joel|요엘서"}, {"아모스", "암", 9, "Amos|아모스서"},
		{"오바댜", "옵", 1, "Obadiah|Obad|오바댜서"}, {"요나", "욘", 4, "Jonah|Jon|요나서"}, {"미가", "미", 7, "Micah|Mic|미가서"},
		{"나훔", "나", 3, "Nahum|Nah|나훔서"}, {"하박국", "합", 3, "Habakkuk|Hab|하박국서"}, {"스바냐", "습", 3, "Zephaniah|Zeph|스바냐서"},
		{"학개", "학", 2, "Haggai|Hag|학개서"}, {"스가랴", "슥", 14, "Zechariah|Zech|스가랴서"}, {"말라기", "말", 4, "Malachi|Mal|말라기서"},
		{"마태복음", "마", 28, "Matthew|Matt|Mt|마태"}, {"마가복음", "막", 16, "Mark|Mk|Mar|마가"}, {"누가복음", "눅", 24, "Luke|Lk|누가"},
		{"요한복음", "요", 21, "John|Jn|Joh"}, {"사도행전", "행", 28, "Acts|Act"}, {"로마서", "롬", 16, "Romans|Rom|Ro"},
		{"고린도전서", "고전", 16, "1 Corinthians|1Corinthians|1 Cor|1Cor|I Corinthians"}, {"고린도후서", "고후", 13, "2 Corinthians|2Corinthians|2 Cor|2Cor|II Corinthians"},
		{"갈라디아서", "갈", 6, "Galatians|Gal"}, {"에베소서", "엡", 6, "Ephesians|Eph"}, {"빌립보서", "빌", 4, "Philippians|Phil|Php"},
		{"골로새서", "골", 4, "Colossians|Col"}, {"데살로니가전서", "살전", 5, "1 Thessalonians|1Thessalonians|1 Thess|1Thess|1 Th|I Thessalonians"},
		{"데살로니가후서", "살후", 3, "2 Thessalonians|2Thessalonians|2 Thess|2Thess|2 Th|II Thessalonians"},
		{"디모데전서", "딤전", 6, "1 Timothy|1Timothy|1 Tim|1Tim|I Timothy"}, {"디모데후서", "딤후", 4, "2 Timothy|2Timothy|2 Tim|2Tim|II Timothy"},
		{"디도서", "딛", 3, "Titus|Tit"}, {"빌레몬서", "몬", 1, "Philemon|Phlm|Philem"}, {"히브리서", "히", 13, "Hebrews|Heb"},
		{"야고보서", "약", 5, "James|Jas|Jam"}, {"베드로전서", "벧전", 5, "1 Peter|1Peter|1 Pet|1Pet|I Peter"}, {"베드로후서", "벧후", 3, "2 Peter|2Peter|2 Pet|2Pet|II Peter"},
		{"요한일서", "요일", 5, "1 John|1John|1 Jn|1Jn|I John"}, {"요한이서", "요이", 1, "2 John|2John|2 Jn|2Jn|II John"}, {"요한삼서", "요삼", 1, "3 John|3John|3 Jn|3Jn|III John"},
		{"유다서", "유", 1, "Jude"}, {"요한계시록", "계", 22, "Revelation|Rev|Apocalypse|계시록"},
	}
	groups := []struct {
		to   int
		name string
	}{{5, "율법서"}, {17, "역사서"}, {22, "시가서"}, {27, "대선지서"}, {39, "소선지서"}, {43, "복음서"}, {44, "역사서"}, {57, "바울서신"}, {65, "일반서신"}, {66, "예언서"}}
	out := make([]*Book, 0, 66)
	for i, r := range raw {
		n := i + 1
		code := fmt.Sprintf("OLD%02d", n)
		if n > 39 {
			code = fmt.Sprintf("NEW%02d", n-39)
		}
		g := ""
		for _, x := range groups {
			if n <= x.to {
				g = x.name
				break
			}
		}
		out = append(out, &Book{N: n, Code: code, Name: r.name, Abbr: r.abbr, Chapters: r.ch, Group: g, OT: n <= 39, alias: strings.Split(r.alias, "|")})
	}
	return out
}()

func bookByN(n int) *Book {
	if n < 1 || n > 66 {
		return nil
	}
	return books[n-1]
}

// 책 이름 사전: 긴 이름부터 맞춰 봐요 ("요한일서"가 "요한"보다, "고전"이 "고"보다 먼저)
type nameEntry struct {
	key  string // 비교용 (소문자 · 빈칸 없음)
	book int
	kind int // 0 = 한국어 전체 이름, 1 = 한국어 약어, 2 = 영어
}

var bookNames = func() []nameEntry {
	var out []nameEntry
	for _, b := range books {
		out = append(out, nameEntry{normName(b.Name), b.N, 0}, nameEntry{normName(b.Abbr), b.N, 1})
		for _, a := range b.alias {
			k := 2
			if hasHangul(a) {
				k = 0
			}
			out = append(out, nameEntry{normName(a), b.N, k})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return len([]rune(out[i].key)) > len([]rune(out[j].key)) })
	return out
}()

func normName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ".", "")
	return s
}

func hasHangul(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Hangul, r) {
			return true
		}
	}
	return false
}

// lookupBook: "로마서", "롬", "Romans", "1 Cor" … → 책 번호
func lookupBook(s string) int {
	k := normName(strings.TrimSpace(s))
	if k == "" {
		return 0
	}
	for _, e := range bookNames {
		if e.key == k {
			return e.book
		}
	}
	return 0
}

type Passage struct {
	Ref  string `json:"ref"`  // 보기 좋게 다시 쓴 표기 (롬 8:28–30)
	Book int    `json:"book"` // 0 = 알 수 없음
	C1   int    `json:"c1"`
	V1   int    `json:"v1"` // 0 = 장 전체
	C2   int    `json:"c2"`
	V2   int    `json:"v2"`
	Raw  string `json:"raw,omitempty"` // 알아보지 못한 표기는 그대로 남겨요
}

func (p Passage) valid() bool { return p.Book > 0 && p.C1 > 0 }

// lo/hi: 절을 (장*1000+절) 하나의 수로 — 장 전체는 1~999절
func (p Passage) lo() int {
	if p.V1 == 0 {
		return p.C1*1000 + 1
	}
	return p.C1*1000 + p.V1
}

func (p Passage) hi() int {
	c2 := p.C2
	if c2 == 0 {
		c2 = p.C1
	}
	if p.V2 == 0 {
		if p.V1 != 0 && c2 == p.C1 {
			return p.C1*1000 + p.V1
		}
		return c2*1000 + 999
	}
	return c2*1000 + p.V2
}

func overlaps(a, b Passage) bool {
	return a.valid() && b.valid() && a.Book == b.Book && a.lo() <= b.hi() && b.lo() <= a.hi()
}

// format: 롬 8:28–30 · 롬 8장 · 시 23편 · 롬 8:28–9:3 · 롬 8–9장
func (p Passage) format() string {
	b := bookByN(p.Book)
	if b == nil {
		return p.Raw
	}
	unit := "장"
	if b.N == 19 {
		unit = "편"
	}
	c2 := p.C2
	if c2 == 0 {
		c2 = p.C1
	}
	switch {
	case p.V1 == 0 && c2 == p.C1:
		return fmt.Sprintf("%s %d%s", b.Abbr, p.C1, unit)
	case p.V1 == 0:
		return fmt.Sprintf("%s %d–%d%s", b.Abbr, p.C1, c2, unit)
	case p.V2 == 0 || (c2 == p.C1 && p.V2 == p.V1):
		return fmt.Sprintf("%s %d:%d", b.Abbr, p.C1, p.V1)
	case c2 == p.C1:
		return fmt.Sprintf("%s %d:%d–%d", b.Abbr, p.C1, p.V1, p.V2)
	default:
		return fmt.Sprintf("%s %d:%d–%d:%d", b.Abbr, p.C1, p.V1, c2, p.V2)
	}
}

var (
	reDash = regexp.MustCompile(`\s*[-–—~〜∼]\s*`)
	// 장:절 꼴 — 8:28, 8:28-30, 8:28-9:3, 8
	reCV = regexp.MustCompile(`^(\d+)(?::(\d+))?(?:-(\d+)(?::(\d+))?)?$`)
	// 한국어 꼴 — 8장 28절, 8장 28-30절, 8장 28절-9장 3절, 8장, 8-9장, 23편
	reKo = regexp.MustCompile(`^(\d+)\s*(?:장|편)\s*(?:(\d+)\s*절?)?(?:-(?:(\d+)\s*(?:장|편)\s*)?(\d+)\s*(?:절|장|편)?)?$`)
)

// parsePassages: 사용자가 적은 본문 (여러 곳은 ; 또는 , 로) → 범위들
func parsePassages(s string) []Passage {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	s = strings.NewReplacer("：", ":", "，", ",", "；", ";", "․", ".", "·", ",").Replace(s)
	var out []Passage
	last := Passage{}
	for _, part := range splitRefs(s) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		p := parseOne(part, last)
		out = append(out, p)
		if p.valid() {
			last = p
		}
	}
	return out
}

// splitRefs: ";" 는 늘 나누고, "," 는 뒤에 책 이름이나 장:절이 오면 나눠요 ("롬 8:28, 32" 는 32절도 함께)
func splitRefs(s string) []string {
	var out []string
	for _, a := range strings.Split(s, ";") {
		out = append(out, strings.Split(a, ",")...)
	}
	return out
}

func parseOne(part string, prev Passage) Passage {
	raw := part
	t := reDash.ReplaceAllString(part, "-")
	t = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(t, "절"), "."))
	// 앞쪽의 책 이름 떼기: 숫자로 시작하는 영어 책(1 Cor)도 있어서 '책 이름 + 빈칸/숫자' 꼴을 찾아요
	book, rest := splitBook(t)
	if book == 0 {
		if prev.valid() {
			book = prev.Book
			rest = t
			// "32" 처럼 숫자 하나만 있으면 앞 본문과 같은 장의 절이에요
			if n, err := strconv.Atoi(strings.TrimSpace(rest)); err == nil && prev.V1 > 0 {
				c := prev.C2
				if c == 0 {
					c = prev.C1
				}
				p := Passage{Book: book, C1: c, V1: n, C2: c, V2: n}
				p.Ref = p.format()
				return p
			}
		} else {
			return Passage{Raw: raw, Ref: raw}
		}
	}
	rest = strings.TrimSpace(rest)
	b := bookByN(book)
	if rest == "" {
		if b != nil && b.Chapters == 1 {
			p := Passage{Book: book, C1: 1, C2: 1}
			p.Ref = p.format()
			return p
		}
		return Passage{Raw: raw, Ref: raw}
	}
	rest = strings.ReplaceAll(rest, " ", "")
	var p Passage
	if m := reCV.FindStringSubmatch(rest); m != nil {
		a, b1, c, d := atoi(m[1]), atoi(m[2]), atoi(m[3]), atoi(m[4])
		p = Passage{Book: book, C1: a}
		switch {
		case b1 == 0 && c == 0: // 8
			p.C2 = a
		case b1 == 0 && c > 0 && d == 0: // 8-9 (장 범위)
			p.C2 = c
		case b1 > 0 && c == 0: // 8:28
			p.V1, p.C2, p.V2 = b1, a, b1
		case b1 > 0 && d == 0: // 8:28-30
			p.V1, p.C2, p.V2 = b1, a, c
		default: // 8:28-9:3
			p.V1, p.C2, p.V2 = b1, c, d
		}
	} else if m := reKo.FindStringSubmatch(rest); m != nil {
		a, b1, c, d := atoi(m[1]), atoi(m[2]), atoi(m[3]), atoi(m[4])
		p = Passage{Book: book, C1: a, C2: a}
		switch {
		case b1 == 0 && d > 0 && c == 0: // 8-9장 (장 범위) — reKo 에서는 "8장-9장" 꼴
			p.C2 = d
		case b1 > 0 && c == 0 && d == 0:
			p.V1, p.V2 = b1, b1
		case b1 > 0 && c == 0:
			p.V1, p.V2 = b1, d
		case b1 > 0:
			p.V1, p.C2, p.V2 = b1, c, d
		}
	} else if m := regexp.MustCompile(`^(\d+)-(\d+)(?:장|편)$`).FindStringSubmatch(rest); m != nil {
		p = Passage{Book: book, C1: atoi(m[1]), C2: atoi(m[2])}
	} else {
		return Passage{Raw: raw, Ref: raw}
	}
	// 장 수가 1인 책(유다서 · 빌레몬서 …)은 "유 3" 을 3절로 봐요
	if b != nil && b.Chapters == 1 && p.V1 == 0 && p.C1 > 1 {
		p = Passage{Book: book, C1: 1, V1: p.C1, C2: 1, V2: p.C2}
	}
	if b != nil && (p.C1 > b.Chapters || p.C1 < 1) {
		return Passage{Raw: raw, Ref: raw}
	}
	if p.C2 < p.C1 || (p.C2 == p.C1 && p.V2 > 0 && p.V2 < p.V1) {
		p.C2, p.V2 = p.C1, p.V1
	}
	p.Ref = p.format()
	return p
}

func splitBook(t string) (int, string) {
	low := strings.ToLower(t)
	// 숫자로 시작하는 영어 책 이름(1 Cor, 2Tim …)을 위해 "숫자 + 글자" 꼴은 책 이름으로 먼저 봐요
	for _, e := range bookNames {
		k := e.key
		// 비교: 빈칸 · 마침표를 빼고 앞부분이 같은지
		i, j := 0, 0
		rs := []rune(low)
		ks := []rune(k)
		for i < len(rs) && j < len(ks) {
			if rs[i] == ' ' || rs[i] == '.' {
				i++
				continue
			}
			if rs[i] != ks[j] {
				break
			}
			i++
			j++
		}
		if j < len(ks) {
			continue
		}
		rest := string(rs[i:])
		// 책 이름 바로 뒤에는 빈칸 · 숫자 · 마침표 · 끝만 와야 해요 ("요한"이 "요한일서"를 먹지 않게)
		if rest != "" {
			r0 := []rune(rest)[0]
			if !(r0 == ' ' || r0 == '.' || unicode.IsDigit(r0)) {
				continue
			}
		}
		return e.book, strings.TrimLeft(rest, " .")
	}
	return 0, t
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// ---------------------------------------------------------------- 긴 글(PDF) 안의 본문 표기 찾기

type VerseRef struct {
	Book, C, V1, V2 int // V1 == 0 → 장만 (장 제목 · "8장")
}

// 책 이름 + 장:절 — "롬 8:28", "로마서 8:28-30", "Rom. 8:28", "1 Cor 13:4"
// 책 이름 + 장 절 — "로마서 8장 28절", "로마서 8장 28-30절"
var reTextRef = func() *regexp.Regexp {
	var names []string
	seen := map[string]bool{}
	for _, b := range books {
		for _, n := range append([]string{b.Name, b.Abbr}, b.alias...) {
			if !seen[n] {
				seen[n] = true
				q := regexp.QuoteMeta(n)
				q = strings.ReplaceAll(q, " ", `\s?`)
				names = append(names, q)
			}
		}
	}
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	pat := `(?i)(` + strings.Join(names, "|") + `)\.?\s?(\d{1,3})\s?(?::\s?(\d{1,3})(?:\s?[-–~]\s?(\d{1,3}))?|장(?:\s?(\d{1,3})\s?(?:[-–~]\s?(\d{1,3}))?\s?절)?|편)`
	return regexp.MustCompile(pat)
}()

// 책 이름 없는 장:절 — "8:28", "8:28-30" (문서의 주된 책 또는 바로 앞에 나온 책)
var reBareCV = regexp.MustCompile(`(?:^|[^\d:])(\d{1,3}):(\d{1,3})(?:\s?[-–~]\s?(\d{1,3}))?`)

// 장 제목 — 줄 처음의 "제8장", "8장", "CHAPTER 8", "Chapter 8"
var reChapterHead = regexp.MustCompile(`(?im)^\s*(?:제\s?(\d{1,3})\s?장|(\d{1,3})\s?장(?:\s|$)|chapter\s+(\d{1,3})\b)`)

// "28절", "28-30절"
var reBareVerse = regexp.MustCompile(`(?:^|[^\d])(\d{1,3})(?:\s?[-–~]\s?(\d{1,3}))?\s?절`)

// isLetterBefore: 한글 약어(요, 사, 아…)가 다른 낱말의 끝이 아닌지 ("필요 3:1" 같은 것을 거르기)
func isWordStart(s string, i int) bool {
	if i == 0 {
		return true
	}
	r := []rune(s[:i])
	p := r[len(r)-1]
	return !(unicode.IsLetter(p) || unicode.IsDigit(p))
}

// findRefs: 쪽 글자 안의 본문 표기들. primary 는 이 문서의 주된 책(없으면 0), curCh 는 앞쪽에서 이어진 장.
func findRefs(text string, primary int, curCh int) (refs []VerseRef, lastCh int) {
	lastCh = curCh
	lastBook := primary
	for _, m := range reTextRef.FindAllStringSubmatchIndex(text, -1) {
		if !isWordStart(text, m[0]) {
			continue
		}
		name := text[m[2]:m[3]]
		b := lookupBook(name)
		if b == 0 {
			continue
		}
		g := func(k int) int {
			if m[2*k] < 0 {
				return 0
			}
			return atoi(text[m[2*k]:m[2*k+1]])
		}
		c := g(2)
		bk := bookByN(b)
		if c < 1 || c > bk.Chapters {
			continue
		}
		v1, v2 := g(3), g(4)
		if v1 == 0 {
			v1, v2 = g(5), g(6)
		}
		if v2 == 0 {
			v2 = v1
		}
		refs = append(refs, VerseRef{b, c, v1, v2})
		lastBook = b
	}
	if lastBook > 0 {
		for _, m := range reBareCV.FindAllStringSubmatch(text, -1) {
			c, v1, v2 := atoi(m[1]), atoi(m[2]), atoi(m[3])
			if bk := bookByN(lastBook); bk == nil || c < 1 || c > bk.Chapters || v1 < 1 || v1 > 176 {
				continue
			}
			if v2 == 0 {
				v2 = v1
			}
			refs = append(refs, VerseRef{lastBook, c, v1, v2})
		}
	}
	if primary > 0 {
		for _, m := range reChapterHead.FindAllStringSubmatch(text, -1) {
			c := atoi(m[1]) + atoi(m[2]) + atoi(m[3])
			if bk := bookByN(primary); bk != nil && c >= 1 && c <= bk.Chapters {
				lastCh = c
				refs = append(refs, VerseRef{primary, c, 0, 0})
			}
		}
		if lastCh > 0 {
			for _, m := range reBareVerse.FindAllStringSubmatch(text, -1) {
				v1, v2 := atoi(m[1]), atoi(m[2])
				if v1 < 1 || v1 > 176 {
					continue
				}
				if v2 == 0 {
					v2 = v1
				}
				refs = append(refs, VerseRef{primary, lastCh, v1, v2})
			}
		}
	}
	return refs, lastCh
}

// guessBook: 파일 이름 · 제목 · 첫 쪽들에서 이 문서가 주로 다루는 책을 짐작해요
func guessBook(title string, pages []string) int {
	t := normName(title)
	best, bestLen := 0, 0
	for _, e := range bookNames {
		if e.kind == 1 && len([]rune(e.key)) < 2 { // 한 글자 약어는 파일 이름에서 믿지 않아요
			continue
		}
		if len([]rune(e.key)) < 3 && e.kind == 2 {
			continue
		}
		if i := strings.Index(t, e.key); i >= 0 && len([]rune(e.key)) > bestLen {
			// 두 글자 한국어 이름(아가 · 나훔 …)은 다른 낱말의 일부가 아닐 때만 ("아가페" 는 아가가 아니에요)
			if e.kind == 0 && len([]rune(e.key)) <= 2 {
				after := []rune(t[i+len(e.key):])
				if len(after) > 0 && unicode.Is(unicode.Hangul, after[0]) && after[0] != '서' {
					continue
				}
			}
			best, bestLen = e.book, len([]rune(e.key))
		}
	}
	if best > 0 {
		return best
	}
	// 파일 이름에 없으면: 앞쪽 40쪽에서 가장 많이 나온 책 (전체의 절반이 넘을 때만)
	count := map[int]int{}
	total := 0
	for i, p := range pages {
		if i >= 40 {
			break
		}
		r, _ := findRefs(p, 0, 0)
		for _, x := range r {
			count[x.Book]++
			total++
		}
	}
	for b, n := range count {
		if total >= 8 && n*2 > total && n > count[best] {
			best = b
		}
	}
	return best
}
