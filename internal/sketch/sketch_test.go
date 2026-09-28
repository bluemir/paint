package sketch

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// newTestSketch 는 width x height 판이 눈금 · 줄 번호 · 띠를 두르고 꼭 맞게 보이는 화면이다.
func newTestSketch(width, height int) *sketch {
	s := newSketch("unused.json", NewCanvas(width, height))
	s.Update(tea.WindowSizeMsg{Width: width + s.originX(), Height: height + 2})
	return s
}

// canvasClick 과 canvasDrag 는 판 칸 (x, y) 를 짚는 마우스다. 굴린 만큼과 눈금 여백만큼 화면 칸을 옮긴다.
func canvasClick(s *sketch, x, y int, button tea.MouseButton) tea.MouseClickMsg {
	return tea.MouseClickMsg{X: x - s.left + s.originX(), Y: y - s.top + 1, Button: button}
}

func canvasDrag(s *sketch, x, y int, button tea.MouseButton) tea.MouseMotionMsg {
	return tea.MouseMotionMsg{X: x - s.left + s.originX(), Y: y - s.top + 1, Button: button}
}

func typed(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: []rune(text)[0], Text: text}
}

func TestPaintAndEraseWithMouse(t *testing.T) {
	s := newTestSketch(10, 5)
	s.brush.Glyph = "@"
	s.Update(canvasClick(s, 1, 1, tea.MouseLeft))
	s.Update(canvasDrag(s, 2, 1, tea.MouseLeft))
	s.Update(canvasDrag(s, 3, 1, 0)) // 버튼을 뗀 채 지나가면 칠하지 않는다
	for x, want := range []string{" ", "@", "@", " "} {
		if got := s.canvas.At(x, 1).Glyph; got != want {
			t.Errorf("(%d,1) = %q, %q 여야 한다", x, got, want)
		}
	}
	if !s.dirty {
		t.Error("칠했는데 수정됨이 아니다")
	}

	s.Update(canvasClick(s, 2, 1, tea.MouseRight))
	if s.canvas.At(2, 1) != blank {
		t.Errorf("우클릭이 안 지웠다: %+v", s.canvas.At(2, 1))
	}
}

// 판이 화면보다 크면 굴린 만큼 밀린 칸에 칠한다.
func TestPaintFollowsScroll(t *testing.T) {
	s := newSketch("unused.json", NewCanvas(20, 20))
	s.Update(tea.WindowSizeMsg{Width: toolbarWidth + 10, Height: 6})
	s.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	s.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	// 화면에서 판이 시작하는 칸(눈금 여백 바로 오른쪽 아래)을 누른다.
	s.Update(tea.MouseClickMsg{X: s.originX(), Y: 1, Button: tea.MouseLeft})
	if s.canvas.At(1, 1) != s.brush {
		t.Errorf("(1,1) = %+v", s.canvas.At(1, 1))
	}
}

// 띠 위를 누르면 판에 안 칠한다.
func TestClickOnStripDoesNotPaint(t *testing.T) {
	s := newTestSketch(10, 5)
	s.Update(tea.MouseClickMsg{X: s.originX(), Y: s.height - 1, Button: tea.MouseLeft})
	if s.dirty {
		t.Error("띠를 눌렀는데 칠했다")
	}
}

func TestTextModeTypesAndBacksUp(t *testing.T) {
	s := newTestSketch(10, 5)
	s.Update(typed("t"))
	s.Update(canvasClick(s, 2, 1, tea.MouseLeft))
	s.Update(typed("a"))
	s.Update(typed("한"))
	if s.cursorX != 5 {
		t.Errorf("커서 = %d, 5 여야 한다", s.cursorX)
	}
	if s.canvas.At(2, 1).Glyph != "a" || s.canvas.At(3, 1).Glyph != "한" {
		t.Errorf("적힌 것 = %q %q", s.canvas.At(2, 1).Glyph, s.canvas.At(3, 1).Glyph)
	}

	s.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if s.cursorX != 3 || s.canvas.At(3, 1) != blank || s.canvas.At(4, 1) != blank {
		t.Errorf("Backspace 뒤 커서 %d, 칸 %+v %+v", s.cursorX, s.canvas.At(3, 1), s.canvas.At(4, 1))
	}

	s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if s.cursorX != 2 || s.cursorY != 2 {
		t.Errorf("Enter 뒤 커서 (%d,%d), (2,2) 여야 한다", s.cursorX, s.cursorY)
	}
}

func TestColorPopupPicksForegroundAndBackground(t *testing.T) {
	s := newTestSketch(60, 30)
	s.Update(typed("c"))
	originX, originY := s.popupOrigin(colorBox())
	// 테두리 한 칸, 제목 한 줄 아래가 격자다. 둘째 줄 셋째 칸이 16+2 = 18 번이다.
	s.Update(tea.MouseClickMsg{X: originX + 1 + 2*swatchWidth, Y: originY + 1 + 2, Button: tea.MouseLeft})
	s.Update(tea.MouseClickMsg{X: originX + 1, Y: originY + 1 + colorRows + 1, Button: tea.MouseRight})
	if s.brush.Fg != 18 || s.brush.Bg != NoColor {
		t.Errorf("전경 %d 배경 %d", s.brush.Fg, s.brush.Bg)
	}
	if s.dirty {
		t.Error("색표를 눌렀는데 판에 칠했다")
	}
}

func TestQuitAsksAgainWhenDirty(t *testing.T) {
	s := newTestSketch(10, 5)
	s.Update(canvasClick(s, 0, 0, tea.MouseLeft))
	quit := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	if _, cmd := s.Update(quit); cmd != nil {
		t.Fatal("저장 안 했는데 한 번에 끝났다")
	}
	if _, cmd := s.Update(quit); cmd == nil {
		t.Fatal("두 번째 ctrl+c 가 안 끝냈다")
	}
}

// 화면이 판보다 작든 크든, 창이 떴든 글자 모드든 띠까지 화면 높이에 맞는다.
func TestViewFitsScreen(t *testing.T) {
	for _, size := range [][2]int{{40, 10}, {120, 40}} {
		s := newSketch("unused.json", NewCanvas(80, 24))
		s.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, key := range []tea.KeyPressMsg{{Code: tea.KeyTab}, {Code: tea.KeyEscape}, typed("c"), {Code: tea.KeyEscape}, typed("v")} {
			s.Update(key)
			content := s.View().Content
			if got := strings.Count(content, "\n") + 1; got != size[1] {
				t.Errorf("%dx%d %s: 줄 수 %d", size[0], size[1], key, got)
			}
		}
	}
}

// 왼쪽 상태가 길어지고 짧아져도 키 안내는 오른쪽 끝 같은 곳에 있다.
func TestStripHintsStayRight(t *testing.T) {
	s := newTestSketch(100, 5)
	short := ansi.Strip(s.strip())
	s.Update(canvasDrag(s, 42, 3, 0))
	s.message = "저장했다: unused.json"
	long := ansi.Strip(s.strip())
	for _, line := range []string{short, long} {
		if ansi.StringWidth(line) != s.width || !strings.HasSuffix(line, stripHints) {
			t.Errorf("띠 = %q (폭 %d)", line, ansi.StringWidth(line))
		}
	}
}

// 판이 화면보다 작으면 판 밖이 점으로 차고 판 안은 비어 있다. 눈금은 0 부터, 줄 번호는 굴린 만큼이다.
func TestBoardShowsCanvasEdge(t *testing.T) {
	s := newSketch("unused.json", NewCanvas(12, 30))
	s.Update(tea.WindowSizeMsg{Width: 30, Height: 12})
	lines := strings.Split(ansi.Strip(s.board(s.canvas)), "\n")
	if len(lines) != s.viewHeight() {
		t.Fatalf("줄 수 = %d, %d 여야 한다", len(lines), s.viewHeight())
	}
	if want := "   0....|....1"; !strings.HasPrefix(lines[0], want) {
		t.Errorf("눈금 = %q, %q 로 시작해야 한다", lines[0], want)
	}
	row := lines[1]
	if !strings.HasPrefix(row, " 0 "+strings.Repeat(" ", 12)) || !strings.HasSuffix(row, strings.Repeat(offCanvasGlyph, 30-toolbarWidth-3-12)) {
		t.Errorf("첫 줄 = %q", row)
	}

	s.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	lines = strings.Split(ansi.Strip(s.board(s.canvas)), "\n")
	if !strings.HasPrefix(lines[1], " 1 ") {
		t.Errorf("굴린 뒤 첫 줄 번호 = %q", lines[1])
	}
}

// 판이 화면보다 짧으면 판 아래 줄은 전부 점이다.
func TestBoardFillsBelowCanvas(t *testing.T) {
	s := newSketch("unused.json", NewCanvas(5, 2))
	s.Update(tea.WindowSizeMsg{Width: 20, Height: 6})
	lines := strings.Split(ansi.Strip(s.board(s.canvas)), "\n")
	if last := lines[len(lines)-1]; strings.TrimLeft(last, " ") != strings.Repeat(offCanvasGlyph, 20-toolbarWidth-s.gutter()) {
		t.Errorf("판 아래 줄 = %q", last)
	}
}

// 눈금 위와 줄 번호 위를 눌러도 판에 안 칠한다.
func TestClickOnRulerDoesNotPaint(t *testing.T) {
	s := newTestSketch(10, 5)
	s.Update(tea.MouseClickMsg{X: s.originX() + 1, Y: 0, Button: tea.MouseLeft})
	s.Update(tea.MouseClickMsg{X: toolbarWidth, Y: 2, Button: tea.MouseLeft})
	if s.dirty {
		t.Error("눈금을 눌렀는데 칠했다")
	}
}

func TestRecolorModeChangesOnlyColor(t *testing.T) {
	s := newTestSketch(10, 5)
	s.brush.Glyph = "@"
	s.Update(canvasClick(s, 1, 1, tea.MouseLeft))
	s.brush.Glyph = "#"
	s.brush.Fg, s.brush.Bg = 196, 17
	s.switchMode(modeRecolor)
	s.Update(canvasClick(s, 1, 1, tea.MouseLeft))
	s.Update(canvasDrag(s, 2, 1, tea.MouseLeft))
	if got := s.canvas.At(1, 1); got != (Cell{Glyph: "@", Fg: 196, Bg: 17}) {
		t.Errorf("(1,1) = %+v, 글자는 @ 그대로여야 한다", got)
	}
	// 빈 칸은 빈칸 그대로 색만 입는다. 배경만 칠하는 길이다.
	if got := s.canvas.At(2, 1); got != (Cell{Glyph: " ", Fg: 196, Bg: 17}) {
		t.Errorf("(2,1) = %+v", got)
	}
}

// 도구 줄은 판 옆에 늘 같은 폭으로 붙고, 이름이 잘리거나 단축키에 붙지 않는다.
func TestToolbarKeepsWidth(t *testing.T) {
	s := newTestSketch(10, 20)
	for row := range s.viewHeight() {
		line := ansi.Strip(s.toolbarLine(row))
		if got := ansi.StringWidth(line); got != toolbarWidth {
			t.Errorf("%d 번째 줄 폭 = %d", row, got)
		}
		if row < len(tools) && tools[row].label != nil && !strings.Contains(line, tools[row].label(s)+" ") {
			t.Errorf("%d 번째 줄 %q 에 이름 뒤 빈칸이 없다", row, line)
		}
	}
}

// 글자 모드에서는 터미널 커서가 적힐 칸에 있다. 입력기가 조합 중인 글자를 거기 그린다.
func TestTextModePlacesTerminalCursor(t *testing.T) {
	s := newTestSketch(10, 5)
	if s.View().Cursor != nil {
		t.Error("칠하기 모드인데 커서가 보인다")
	}
	s.switchMode(modeText)
	s.Update(canvasClick(s, 3, 2, tea.MouseLeft))
	cursor := s.View().Cursor
	if cursor == nil || cursor.X != 3+s.originX() || cursor.Y != 2+1 {
		t.Errorf("커서 = %+v", cursor)
	}
	s.openPopup(popupColor)
	if s.View().Cursor != nil {
		t.Error("색표가 떴는데 커서가 보인다")
	}
}

// 글자 모드 Backspace 는 왼쪽 칸을 지우고 물러설 뿐 뒤 글자를 당기지 않는다.
func TestTextBackspaceLeavesFollowingText(t *testing.T) {
	s := newTestSketch(10, 3)
	s.switchMode(modeText)
	s.Update(canvasClick(s, 0, 0, tea.MouseLeft))
	for _, r := range "abc" {
		s.Update(typed(string(r)))
	}
	s.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	s.Update(tea.KeyPressMsg{Code: tea.KeyBackspace}) // b 를 지운다
	if got := ansi.Strip(s.canvas.renderArea(0, 0, 4, 1)); got != "a c " || s.cursorX != 1 {
		t.Errorf("%q, 커서 %d", got, s.cursorX)
	}
}

// 칠하기 모드에서 글자 키는 붓을 안 바꾼다. 글자표가 떠 있을 때만 바꾸고, 바꾸면 창이 닫힌다.
func TestBrushChangesOnlyInGlyphPopup(t *testing.T) {
	s := newTestSketch(10, 5)
	before := s.brush.Glyph
	s.Update(typed("x"))
	if s.brush.Glyph != before {
		t.Errorf("칠하기 모드 글자 키가 붓을 바꿨다: %q", s.brush.Glyph)
	}
	s.Update(typed("v"))
	s.Update(typed("한"))
	if s.brush.Glyph != "한" || s.popup != popupNone {
		t.Errorf("붓 %q, 창 %d", s.brush.Glyph, s.popup)
	}
}

// 한 글자 키가 도구를 든다. 도구 줄에 적힌 키와 같다.
func TestToolKeys(t *testing.T) {
	for key, want := range map[string]mode{"b": modePaint, "t": modeText, "f": modeRecolor, "e": modeErase} {
		s := newTestSketch(10, 5)
		start := modeErase // 키가 고르는 도구와 다른 곳에서 시작한다
		if key == "e" {
			start = modePaint
		}
		s.setMode(start)
		s.Update(typed(key))
		if s.mode != want {
			t.Errorf("%s: 모드 = %s, %s 여야 한다", key, s.mode, want)
		}
	}
	for key, want := range map[string]popup{"c": popupColor, "v": popupGlyph} {
		s := newTestSketch(10, 5)
		s.Update(typed(key))
		if s.popup != want {
			t.Errorf("%s: 창 = %d, %d 여야 한다", key, s.popup, want)
		}
	}
}

func TestWASDScrolls(t *testing.T) {
	s := newSketch("unused.json", NewCanvas(40, 40))
	s.Update(tea.WindowSizeMsg{Width: toolbarWidth + 13, Height: 12})
	s.Update(typed("s"))
	s.Update(typed("d"))
	if s.left != 1 || s.top != 1 {
		t.Errorf("굴린 곳 = (%d,%d), (1,1) 이어야 한다", s.left, s.top)
	}
	s.Update(typed("w"))
	s.Update(typed("a"))
	if s.left != 0 || s.top != 0 {
		t.Errorf("되돌린 곳 = (%d,%d)", s.left, s.top)
	}
}

// 글자 모드에서는 도구 키도 입력이다. Esc 가 들어오기 전 모드로 돌아간다.
func TestTextModeTypesToolKeysAndEscReturns(t *testing.T) {
	s := newTestSketch(10, 5)
	s.Update(typed("f"))
	s.Update(typed("t"))
	s.Update(canvasClick(s, 0, 0, tea.MouseLeft))
	s.Update(typed("q"))
	if s.mode != modeText || s.canvas.At(0, 0).Glyph != "q" {
		t.Errorf("모드 %s, 칸 %q", s.mode, s.canvas.At(0, 0).Glyph)
	}
	s.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if s.mode != modeRecolor {
		t.Errorf("Esc 뒤 모드 = %s, 색칠로 돌아가야 한다", s.mode)
	}
}

// 도구 키가 조합 글자로 오면 띠 맨 앞에 입력기 안내가 뜨고, 라틴 글자가 오면 사라진다. 글자 모드의
// 한글은 뜻이라 안내하지 않는다.
func TestIMEWarningOnStrip(t *testing.T) {
	const warning = "입력기가 켜져 있다"
	s := newTestSketch(60, 5)
	s.Update(typed("t"))
	s.Update(typed("한"))
	s.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if strings.Contains(ansi.Strip(s.strip()), warning) {
		t.Error("글자 모드에서 친 한글에 안내가 떴다")
	}
	s.Update(typed("ㅠ"))
	if !strings.HasPrefix(ansi.Strip(s.strip()), warning) {
		t.Errorf("띠 = %q", ansi.Strip(s.strip()))
	}
	s.Update(typed("b"))
	if strings.Contains(ansi.Strip(s.strip()), warning) {
		t.Error("입력기를 끈 뒤에도 안내가 남았다")
	}
}

// 색표 바깥을 누르면 창이 닫히고 판에는 안 칠한다. 창 안(테두리 포함)을 누르면 떠 있다.
func TestColorPopupClosesOnOutsideClick(t *testing.T) {
	s := newTestSketch(60, 30)
	s.Update(typed("c"))
	originX, originY := s.popupOrigin(colorBox())
	s.Update(tea.MouseClickMsg{X: originX, Y: originY, Button: tea.MouseLeft}) // 테두리 모서리
	if s.popup != popupColor {
		t.Fatal("테두리를 눌렀는데 창이 닫혔다")
	}
	s.Update(tea.MouseClickMsg{X: originX - 1, Y: originY + 3, Button: tea.MouseLeft})
	if s.popup != popupNone {
		t.Error("바깥을 눌렀는데 창이 떠 있다")
	}
	if s.dirty {
		t.Error("창을 닫는 누름이 판에 칠했다")
	}
}

// q 는 모드를 안 바꾸고 마우스가 짚은 칸을 바로 붓에 담는다. 판 밖이면 띠에 적는다.
func TestQPicksHoveredCell(t *testing.T) {
	s := newTestSketch(10, 5)
	s.setMode(modeRecolor)
	s.canvas.Put(3, 2, Cell{Glyph: "한", Fg: 40, Bg: 17})
	s.Update(typed("q"))
	if s.message == "" {
		t.Error("판 밖에서 q 를 눌렀는데 알림이 없다")
	}
	s.Update(canvasDrag(s, 4, 2, 0)) // 넓은 글자의 오른쪽 반쪽 위
	s.Update(typed("q"))
	if s.brush != (Cell{Glyph: "한", Fg: 40, Bg: 17}) {
		t.Errorf("붓 = %+v", s.brush)
	}
	if s.mode != modeRecolor {
		t.Errorf("모드 = %s, 그대로여야 한다", s.mode)
	}
}

// 띠의 전경 · 배경을 누르면 색표가, 붓을 누르면 글자표가 뜬다. 다른 토막은 아무것도 안 연다.
func TestStripColorOpensColorPopup(t *testing.T) {
	s := newTestSketch(80, 5)
	line := ansi.Strip(s.strip())
	for _, label := range []string{"전경", "배경"} {
		s.openPopup(popupNone)
		x := ansi.StringWidth(line[:strings.Index(line, label)]) + 1
		s.Update(tea.MouseClickMsg{X: x, Y: s.height - 1, Button: tea.MouseLeft})
		if s.popup != popupColor {
			t.Errorf("%s 을 눌렀는데 창 = %d", label, s.popup)
		}
	}
	s.openPopup(popupNone)
	s.Update(tea.MouseClickMsg{X: 0, Y: s.height - 1, Button: tea.MouseLeft})
	if s.popup != popupGlyph {
		t.Errorf("붓 토막을 눌렀는데 창 = %d", s.popup)
	}
	s.openPopup(popupNone)
	x := ansi.StringWidth(line[:strings.Index(line, s.mode.String())]) + 1
	s.Update(tea.MouseClickMsg{X: x, Y: s.height - 1, Button: tea.MouseLeft})
	if s.popup != popupNone {
		t.Errorf("모드 토막을 눌렀는데 창 = %d", s.popup)
	}
}

// 띠의 붓 뒤에 코드 포인트가 적힌다. 여러 rune 으로 된 글자는 rune 마다다. 폭이 애매한 글자는 그
// 까닭이 붙는다.
func TestStripShowsBrushCodePoint(t *testing.T) {
	for glyph, want := range map[string]string{"▲": "U+25B2 폭 애매(Ambiguous)", "☀": "U+2600 폭 애매(이모지 모양)", "✓": "U+2713", "한": "U+D55C", "⇗▴": "U+21D7 U+25B4", "　": "(전각 빈칸) U+3000"} {
		got := ansi.Strip(brushLabel(Cell{Glyph: glyph, Fg: NoColor, Bg: NoColor}))
		if !strings.HasSuffix(got, want) {
			t.Errorf("%q: %q, %q 로 끝나야 한다", glyph, got, want)
		}
	}
}
