package main

import "testing"

func TestParsePassages(t *testing.T) {
	cases := []struct{ in, want string }{
		{"롬 8:28-30", "롬 8:28–30"},
		{"로마서 8:28–30", "롬 8:28–30"},
		{"롬8:28~30", "롬 8:28–30"},
		{"로마서 8장 28-30절", "롬 8:28–30"},
		{"로마서 8장 28절", "롬 8:28"},
		{"로마서 8장 28절-9장 3절", "롬 8:28–9:3"},
		{"롬 8:28-9:3", "롬 8:28–9:3"},
		{"로마서 8장", "롬 8장"},
		{"롬 8-9장", "롬 8–9장"},
		{"시편 23편", "시 23편"},
		{"시 23", "시 23편"},
		{"요한일서 4:7-12", "요일 4:7–12"},
		{"요 3:16", "요 3:16"},
		{"Romans 8:28", "롬 8:28"},
		{"1 Cor 13:4-7", "고전 13:4–7"},
		{"유 3", "유 1:3"},
		{"계 21:1-5", "계 21:1–5"},
		{"마태복음 5장 1~12절", "마 5:1–12"},
		{"창 1:1-2:3", "창 1:1–2:3"},
		{"아가 2:1", "아 2:1"},
	}
	for _, c := range cases {
		ps := parsePassages(c.in)
		if len(ps) != 1 || ps[0].Ref != c.want {
			t.Errorf("%q → %+v, want %q", c.in, ps, c.want)
		}
	}
	ps := parsePassages("롬 8:28, 32; 엡 1:3-14")
	if len(ps) != 3 || ps[1].Ref != "롬 8:32" || ps[2].Ref != "엡 1:3–14" {
		t.Errorf("multi → %+v", ps)
	}
	if ps := parsePassages("아무 말"); len(ps) != 1 || ps[0].valid() {
		t.Errorf("garbage → %+v", ps)
	}
}

func TestOverlap(t *testing.T) {
	a := parsePassages("롬 8:28-30")[0]
	for in, want := range map[string]bool{"롬 8:29": true, "롬 8": true, "롬 8:31-39": false, "롬 7-9장": true, "엡 1:4": false, "롬 8:1-28": true} {
		if got := overlaps(a, parsePassages(in)[0]); got != want {
			t.Errorf("overlap(%s) = %v", in, got)
		}
	}
}

func TestFindRefs(t *testing.T) {
	text := "제8장\n바울은 로마서 8:28에서 말한다. 28절의 '모든 것'은 고난을 포함한다. 엡 1:4도 보라. 회의는 11:00에"
	refs, ch := findRefs(text, 45, 0)
	if ch != 8 {
		t.Errorf("chapter = %d", ch)
	}
	has := func(b, c, v int) bool {
		for _, r := range refs {
			if r.Book == b && r.C == c && r.V1 == v {
				return true
			}
		}
		return false
	}
	if !has(45, 8, 28) || !has(49, 1, 4) || !has(45, 8, 0) {
		t.Errorf("refs = %+v", refs)
	}
	if b := guessBook("칼빈 로마서 주석", nil); b != 45 {
		t.Errorf("guess = %d", b)
	}
	if b := guessBook("아가페 출판사 설교집", nil); b == 22 {
		t.Errorf("아가페 를 아가로 봤어요")
	}
}
