package main

import "testing"

func TestParseLoose(t *testing.T) {
	cases := []struct {
		in                     string
		hint                   int
		date, kind, ref, title string
	}{
		{"20190310 주일 롬8장28-30 하나님의 영원한 작정", 0, "2019-03-10", "주일", "롬 8:28–30", "하나님의 영원한 작정"},
		{"2019.03.10 주일오후 로마서 8장 28~30절 구원의 황금사슬(최종)", 0, "2019-03-10", "주일 오후", "롬 8:28–30", "구원의 황금사슬"},
		{"190313_수요_시편23편_여호와는 나의 목자", 0, "2019-03-13", "수요", "시 23편", "여호와는 나의 목자"},
		{"0310 새벽 요 3:16 이처럼 사랑하사", 2021, "2021-03-10", "새벽", "요 3:16", "이처럼 사랑하사"},
		{"2020년 4월 12일 부활절 고전 15:1-11 부활의 첫 열매", 0, "2020-04-12", "", "고전 15:1–11", "부활의 첫 열매"},
		{"0317 주일오후 엡1장3-14절 창세 전에 택하사", 2019, "2019-03-17", "주일 오후", "엡 1:3–14", "창세 전에 택하사"},
		{"0412 부활절 고전15장1-11절 부활의 첫 열매(최종)", 2020, "2020-04-12", "", "고전 15:1–11", "부활의 첫 열매"},
		{"2018-10-28 엡 2.8-10 은혜로 말미암아", 0, "2018-10-28", "", "엡 2:8–10", "은혜로 말미암아"},
		{"20230805 여름 수련회 행 1:8 땅끝까지 이르러", 0, "2023-08-05", "수련회", "행 1:8", "땅끝까지 이르러"},
		{"2023.09.03 청년부 요 15:1-8 참포도나무", 0, "2023-09-03", "부서", "요 15:1–8", "참포도나무"},
		{"장례예배 2022.1.5 살전4장13-18절 소망 있는 자의 슬픔", 0, "2022-01-05", "그 외", "살전 4:13–18", "소망 있는 자의 슬픔"},
	}
	for _, c := range cases {
		r := parseLoose(c.in, c.hint)
		ref := refsText(parsePassages(r.PassageText))
		if r.Date != c.date || r.Kind != c.kind || ref != c.ref || r.Title != c.title {
			t.Errorf("%q → date=%q kind=%q ref=%q title=%q", c.in, r.Date, r.Kind, ref, r.Title)
		}
	}
}

func TestRowsFromCells(t *testing.T) {
	rows := rowsFromCells(splitPasted("날짜\t종류\t본문\t제목\t장소\n2019.3.10\t주일\t롬 8:28-30\t하나님의 작정\t본 교회\n2019.3.13\t수요\t시 23편\t목자\t기도원"))
	if len(rows) != 2 || rows[0].Date != "2019-03-10" || rows[1].Place != "기도원" || rows[1].Kind != "수요" {
		t.Errorf("header rows → %+v", rows)
	}
	rows = rowsFromCells(splitPasted("2019-03-10 | 롬 8:28-30 | 하나님의 작정 | 본 교회\n2019-03-17 | 주일 | 엡 1:4 | 택하심"))
	if len(rows) != 2 || rows[0].Title != "하나님의 작정" || rows[0].Place != "본 교회" || rows[1].Kind != "주일" || rows[1].PassageText != "엡 1:4" {
		t.Errorf("no header → %+v", rows)
	}
	rows = rowsFromCells(splitPasted("2019.03.10 주일 롬 8:28-30 하나님의 작정\n2019.03.17 엡 1:4 택하심"))
	if len(rows) != 2 || rows[0].Kind != "주일" || rows[1].Title != "택하심" {
		t.Errorf("loose → %+v", rows)
	}
	if d := normDate("43534"); d != "2019-03-10" {
		t.Errorf("excel serial → %s", d)
	}
}
