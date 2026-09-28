package sketch

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/bluemir/paint/internal/tui/components"
)

var openGlyphs = typed("v")

// openGlyphTable 은 v 키로 글자표를 연다. 뜬 화면이 글자표가 아니면 시험을 멈춘다.
func openGlyphTable(t *testing.T, s *testSketch) *viewGlyph {
	t.Helper()
	table, ok := send(s, openGlyphs).(*viewGlyph)
	if !ok {
		t.Fatal("v 를 눌렀는데 글자표가 안 떴다")
	}
	return table
}

// 글자표의 첫 묶음은 빈칸이다. 첫 글자 줄의 첫 칸을 누르면 반각 빈칸이 붓이 된다.
func TestGlyphPopupPicksBlank(t *testing.T) {
	s := newTestSketch(80, 30)
	table := openGlyphTable(t, s)
	originX, originY := s.popupOrigin(table.box())
	// 테두리, 안내 · 검색 두 줄, 묶음 제목 한 줄 아래가 빈칸 줄이다.
	next := send(table, tea.MouseClickMsg{X: originX + 1 + glyphCellWidth, Y: originY + 1 + glyphListTop + 1, Button: tea.MouseLeft})
	if s.brush.Glyph != "　" {
		t.Errorf("붓 = %q, 전각 빈칸이어야 한다", s.brush.Glyph)
	}
	if !onMode(next) {
		t.Error("고른 뒤에도 창이 떠 있다")
	}
}

// 굴린 만큼 밀린 줄에서 고른다.
func TestGlyphPopupPicksAfterScroll(t *testing.T) {
	s := newTestSketch(80, 20)
	table := openGlyphTable(t, s)
	send(table, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if table.top != 3 {
		t.Fatalf("내려 본 줄 = %d, 3 이어야 한다", table.top)
	}
	originX, originY := s.popupOrigin(table.box())
	row := table.rows()[3]
	send(table, tea.MouseClickMsg{X: originX + 1, Y: originY + 1 + glyphListTop, Button: tea.MouseLeft})
	if row.glyphs == nil {
		t.Fatalf("넷째 줄이 제목이다: %q", row.title)
	}
	if s.brush.Glyph != row.glyphs[0] {
		t.Errorf("붓 = %q, %q 여야 한다", s.brush.Glyph, row.glyphs[0])
	}
}

// 글자표는 지금 붓 글자가 있는 줄을 맨 위에 두고 열린다. 표에 없는 글자면 맨 위다.
func TestGlyphPopupOpensAtBrush(t *testing.T) {
	s := newTestSketch(80, 30)
	s.brush.Glyph = "∞"
	table := openGlyphTable(t, s)
	if table.top == 0 || !slices.Contains(table.rows()[table.top].glyphs, "∞") {
		t.Errorf("맨 위 줄 %d = %+v, ∞ 가 있는 줄이어야 한다", table.top, table.rows()[table.top])
	}
	send(s, escape)
	s.brush.Glyph = "한"
	if table := openGlyphTable(t, s); table.top != 0 {
		t.Errorf("표에 없는 글자인데 맨 위 줄 = %d", table.top)
	}
}

// 굴려도 창 크기가 그대로다. 크기가 바뀌면 가운데 놓인 창이 흔들린다.
func TestGlyphBoxKeepsSize(t *testing.T) {
	s := newTestSketch(80, 20)
	table := openGlyphTable(t, s)
	first := table.box()
	for range 100 {
		send(table, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	last := table.box()
	if ansi.StringWidth(first) != ansi.StringWidth(last) || strings.Count(first, "\n") != strings.Count(last, "\n") {
		t.Errorf("창 크기가 바뀌었다:\n%s\n%s", first, last)
	}
}

// 글자는 한 칸이나 두 칸이다. 한 글자가 두 묶음에 나오지 않는다.
func TestGlyphGroupsAreDrawable(t *testing.T) {
	seen := map[string]string{}
	for _, group := range glyphGroups() {
		for _, glyph := range group.glyphs {
			if other, ok := seen[glyph]; ok {
				t.Errorf("%q 가 %s 와 %s 에 다 있다", glyph, other, group.title)
			}
			seen[glyph] = group.title
			if got := ansi.StringWidth(glyph); got < 1 || got > 2 {
				t.Errorf("%s: %q 의 폭 = %d", group.title, glyph, got)
			}
		}
	}
}

// 폭 0 인 글자는 글자표에 오르지 않는다. 폭이 애매한 글자는 오른다.
func TestDrawable(t *testing.T) {
	for _, r := range []rune{'\u200D', '\uFE0F', '\u0301'} {
		if drawable(r) {
			t.Errorf("%U 가 걸러지지 않았다", r)
		}
	}
	for _, r := range []rune{'☀', '▲', '█', 'α', '✓', '😀', '䷀'} {
		if !drawable(r) {
			t.Errorf("%q (%U) 가 걸러졌다", r, r)
		}
	}
}

// 폭이 애매한 글자는 까닭이 붙는다. 확실한 글자는 "" 다.
func TestWidthDoubt(t *testing.T) {
	for glyph, want := range map[string]string{
		"▲": "Ambiguous", "█": "Ambiguous", "α": "Ambiguous", "·": "Ambiguous", "★": "Ambiguous",
		"☀": "이모지 모양", "⚠": "이모지 모양", "❤": "이모지 모양", "✌": "이모지 모양",
		"☰": "라이브러리 불일치", "䷀": "라이브러리 불일치",
		"✓": "", "⚡": "", "😀": "", "⇄": "", "₩": "", "＠": "", "한": "", "⇗▴": "",
	} {
		if got := widthDoubt(glyph); got != want {
			t.Errorf("%q: %q, %q 여야 한다", glyph, got, want)
		}
	}
}

// 전에는 폭이 애매해서 빠졌던 글자와 새로 훑는 구간의 글자가 글자표에 있다.
func TestGlyphGroupsHaveDoubtfulGlyphs(t *testing.T) {
	all := map[string]bool{}
	for _, group := range glyphGroups() {
		for _, glyph := range group.glyphs {
			all[glyph] = true
		}
	}
	for _, glyph := range []string{"█", "▲", "●", "←", "★", "♥", "∞", "☀", "⚠", "α", "Ω", "①", "Ⅳ", "¤", "×"} {
		if !all[glyph] {
			t.Errorf("%q 가 글자표에 없다", glyph)
		}
	}
}

func TestEraseModeErasesWithLeftClick(t *testing.T) {
	s := newTestSketch(10, 5)
	s.Update(canvasClick(s, 1, 1, tea.MouseLeft))
	s.setMode(modeErase)
	if s.mode() != modeErase {
		t.Fatalf("모드 = %s", s.mode())
	}
	s.Update(canvasClick(s, 1, 1, tea.MouseLeft))
	if s.canvas.At(1, 1) != blank {
		t.Errorf("지우기 모드 좌클릭이 안 지웠다: %+v", s.canvas.At(1, 1))
	}

}

// 글자표 창은 터미널 높이의 60%, 폭의 80% 안이다. 폭은 칸 단위로 내림한다.
func TestGlyphBoxSize(t *testing.T) {
	s := newTestScreen("unused.json", NewCanvas(80, 40))
	s.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	box := openGlyphTable(t, s).box()
	if got, want := lipgloss.Height(box), 30; got != want {
		t.Errorf("창 높이 = %d, %d 여야 한다", got, want)
	}
	// 200 의 80% 는 160 이고, 테두리 둘을 빼면 158 칸이라 3 칸짜리가 52 개 든다.
	if got, want := lipgloss.Width(box), 52*glyphCellWidth+2; got != want {
		t.Errorf("창 폭 = %d, %d 여야 한다", got, want)
	}
}

// 그리지 않고 잰 크기(boxSize)가 그린 창과 같다. 어긋나면 마우스가 짚는 칸과 고르는 칸이 밀린다.
// 걸러서 줄이 줄거나 하나도 안 걸려도 창 크기는 그대로다.
func TestGlyphBoxSizeMatchesRender(t *testing.T) {
	for _, size := range [][2]int{{200, 50}, {80, 24}, {40, 30}} {
		for _, query := range []string{"", "삼각형", "없는말없는말"} {
			s := newTestScreen("unused.json", NewCanvas(40, 20))
			s.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			table := openGlyphTable(t, s)
			table.query = components.NewText(query)
			box := table.box()
			width, height := table.boxSize()
			if lipgloss.Width(box) != width || lipgloss.Height(box) != height {
				t.Errorf("%v %q: 그린 창 %dx%d, 잰 크기 %dx%d", size, query, lipgloss.Width(box), lipgloss.Height(box), width, height)
			}
		}
	}
}

// 좁은 터미널에서도 한 줄에 minGlyphColumns 자는 든다.
func TestGlyphBoxKeepsMinColumns(t *testing.T) {
	s := newTestScreen("unused.json", NewCanvas(40, 20))
	s.Update(tea.WindowSizeMsg{Width: 40, Height: 30})
	if got, want := lipgloss.Width(openGlyphTable(t, s).box()), minGlyphColumns*glyphCellWidth+2; got != want {
		t.Errorf("창 폭 = %d, %d 여야 한다", got, want)
	}
}

// 안내 줄이 창 폭 안에 든다. 넘치면 두 줄로 접혀 클릭한 줄과 고른 글자가 한 줄 어긋난다.
func TestGlyphTitleFits(t *testing.T) {
	if got := ansi.StringWidth(glyphTitle); got > minGlyphColumns*glyphCellWidth {
		t.Errorf("안내 폭 = %d, %d 안이어야 한다", got, minGlyphColumns*glyphCellWidth)
	}
}

// 검색 줄을 누르면 검색이 시작되고, 치는 글자는 붓이 아니라 검색어로 간다. Enter 는 걸러진 첫 글자를
// 붓으로 삼는다.
func TestGlyphSearchFiltersAndPicks(t *testing.T) {
	s := newTestSketch(80, 30)
	brush := s.brush.Glyph
	table := openGlyphTable(t, s)
	originX, originY := s.popupOrigin(table.box())
	send(table, tea.MouseClickMsg{X: originX + 3, Y: originY + 1 + glyphListTop - 1, Button: tea.MouseLeft})
	if !table.searching {
		t.Fatal("검색 줄을 눌렀는데 검색이 안 시작됐다")
	}
	next := send(table, typed("세"), typed("모"))
	if s.brush.Glyph != brush || next != tea.Model(table) {
		t.Fatalf("검색 중 글자 키가 붓을 바꿨다: %q", s.brush.Glyph)
	}
	// 세모는 zn 의 찾는 말이다. 걸러진 것이 삼각형이다.
	rows := table.rows()
	if len(rows) < 2 || !strings.Contains(glyphNames[rows[1].glyphs[0]], "삼각형") {
		t.Fatalf("걸러진 줄 = %+v", rows)
	}
	next = send(table, tea.KeyPressMsg{Code: tea.KeyEnter})
	if s.brush.Glyph != rows[1].glyphs[0] || !onMode(next) {
		t.Errorf("붓 = %q, %q 여야 한다", s.brush.Glyph, rows[1].glyphs[0])
	}
}

// 검색어는 한국어 이름, 영문 유니코드 이름, 묶음 이름으로 걸린다. 말이 여럿이면 다 들어야 한다.
func TestGlyphMatches(t *testing.T) {
	for _, tc := range []struct {
		title, glyph, query string
		want                bool
	}{
		{"화살표", "←", "왼쪽", true},
		{"선", "┌", "down and right", true},
		{"선", "┌", "선", true},
		{"화살표", "←", "right", false},
		{"화살표", "←", "arrow left", true},
		{"화살표", "←", "arrow 위쪽", false},
	} {
		if got := glyphMatches(tc.title, tc.glyph, tc.query); got != tc.want {
			t.Errorf("%q 가 %q 에 %v, %v 여야 한다", tc.glyph, tc.query, got, tc.want)
		}
	}
}

// 검색 중 Esc 는 창을 닫지 않고 검색에서 손을 뗀다. 그다음 글자 키는 다시 붓이다.
func TestGlyphSearchEscStopsSearching(t *testing.T) {
	s := newTestSketch(80, 30)
	table := openGlyphTable(t, s)
	table.startSearch()
	if next := send(table, tea.KeyPressMsg{Code: tea.KeyEscape}); table.searching || next != tea.Model(table) {
		t.Fatalf("검색 %v, 화면 %T", table.searching, next)
	}
	send(table, typed("x"))
	if s.brush.Glyph != "x" {
		t.Errorf("붓 = %q", s.brush.Glyph)
	}
}

// 걸러져 줄이 줄어도 창 높이는 그대로다.
func TestGlyphBoxKeepsHeightWhenFiltered(t *testing.T) {
	s := newTestSketch(80, 40)
	table := openGlyphTable(t, s)
	before := strings.Count(table.box(), "\n")
	table.startSearch()
	for _, query := range []string{"z", "zzzzqqq"} {
		table.query = components.NewText(query)
		if got := strings.Count(table.box(), "\n"); got != before {
			t.Errorf("%q: 창 높이 %d, %d 여야 한다", query, got, before)
		}
	}
}

// 8점 점자 256 자(빈 점자 U+2800 은 빈칸처럼 보여 뺀다)와 팔괘 · 64괘가 글자표에 있다.
func TestGlyphGroupsHaveBrailleAndTrigrams(t *testing.T) {
	all := map[string]bool{}
	for _, group := range glyphGroups() {
		for _, glyph := range group.glyphs {
			all[glyph] = true
		}
	}
	for _, r := range []rune{'⠁', '⣿', '⡀', '☰', '☷', '⚊', '䷀', '䷿'} {
		if !all[string(r)] {
			t.Errorf("%q (%U) 가 글자표에 없다", r, r)
		}
	}
}

// 마우스가 올라간 칸이 표시되고, 그 칸을 누르면 그 글자가 붓이 된다. 굴리면 칸 밑의 글자가 바뀐 대로
// 따라간다.
func TestGlyphHoverMatchesPick(t *testing.T) {
	s := newTestSketch(80, 30)
	table := openGlyphTable(t, s)
	originX, originY := s.popupOrigin(table.box())
	// 목록 여덟째 줄(선 묶음 안)의 셋째 칸이다.
	at := tea.Mouse{X: originX + 1 + 2*glyphCellWidth, Y: originY + 1 + glyphListTop + 7}
	send(table, tea.MouseMotionMsg(at))
	if table.hoverRow < 0 {
		t.Fatal("글자 칸 위인데 표시가 없다")
	}
	glyph := table.rows()[table.hoverRow].glyphs[table.hoverColumn]
	if !strings.Contains(table.box(), glyphHoverStyle.Render(glyph)) {
		t.Errorf("%q 칸에 표시가 그려지지 않았다", glyph)
	}
	before := table.hoverRow
	send(table, tea.MouseWheelMsg{X: at.X, Y: at.Y, Button: tea.MouseWheelDown})
	if table.hoverRow != before+3 {
		t.Errorf("굴린 뒤 줄 = %d, %d 여야 한다", table.hoverRow, before+3)
	}
	want := table.rows()[table.hoverRow].glyphs[table.hoverColumn]
	at.Button = tea.MouseLeft
	send(table, tea.MouseClickMsg(at))
	if s.brush.Glyph != want {
		t.Errorf("붓 = %q, 올라가 있던 %q 여야 한다", s.brush.Glyph, want)
	}
}

// 묶음 제목이나 창 밖에서는 표시가 없다.
func TestGlyphHoverOffCell(t *testing.T) {
	s := newTestSketch(80, 30)
	table := openGlyphTable(t, s)
	originX, originY := s.popupOrigin(table.box())
	for _, at := range []tea.Mouse{{X: originX + 2, Y: originY + 1 + glyphListTop}, {X: 0, Y: 0}} {
		send(table, tea.MouseMotionMsg(at))
		if table.hoverRow >= 0 {
			t.Errorf("%+v: 표시가 있다 (%d,%d)", at, table.hoverRow, table.hoverColumn)
		}
	}
}

// 글자표 바깥을 누르면 창이 닫히고 판에는 안 칠한다. 테두리를 누르면 떠 있다.
func TestGlyphPopupClosesOnOutsideClick(t *testing.T) {
	s := newTestSketch(80, 30)
	table := openGlyphTable(t, s)
	originX, originY := s.popupOrigin(table.box())
	if next := send(table, tea.MouseClickMsg{X: originX, Y: originY, Button: tea.MouseLeft}); next != tea.Model(table) {
		t.Fatal("테두리를 눌렀는데 창이 닫혔다")
	}
	if next := send(table, tea.MouseClickMsg{X: originX - 1, Y: originY + 3, Button: tea.MouseLeft}); !onMode(next) || s.dirty {
		t.Errorf("화면 %T, 칠함 %v", next, s.dirty)
	}
}

// 글자표에서 Tab 은 검색 줄에 손을 대고, 한 번 더 누르면 뗀다. 모양은 그대로다.
func TestGlyphTabTogglesSearch(t *testing.T) {
	s := newTestSketch(80, 30)
	table := openGlyphTable(t, s)
	figure := s.figure
	send(table, tea.KeyPressMsg{Code: tea.KeyTab})
	if !table.searching || s.figure != figure {
		t.Fatalf("검색 %v, 모양 %s", table.searching, s.figure)
	}
	send(table, typed("x"))
	if table.query.Value() != "x" {
		t.Errorf("검색어 = %q", table.query.Value())
	}
	send(table, tea.KeyPressMsg{Code: tea.KeyTab})
	if table.searching || s.figure != figure {
		t.Errorf("검색 %v, 모양 %s", table.searching, s.figure)
	}
}
