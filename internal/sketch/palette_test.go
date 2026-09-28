package sketch

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var openPalette = tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl}

// openCommandPalette 는 ctrl+p 로 팔레트를 연다. 뜬 화면이 팔레트가 아니면 시험을 멈춘다.
func openCommandPalette(t *testing.T, s *testSketch) *viewCommand {
	t.Helper()
	palette, ok := send(s, openPalette).(*viewCommand)
	if !ok {
		t.Fatal("ctrl+p 를 눌렀는데 팔레트가 안 떴다")
	}
	return palette
}

func TestPaletteOpensColorAndGlyph(t *testing.T) {
	for _, tc := range []struct {
		query string
		want  string
	}{
		{"", "*sketch.viewColor"}, // 걸러지기 전 첫 줄이 색표다
		{"색", "*sketch.viewColor"},
		{"글자", "*sketch.viewGlyph"},
		{"glyph", "*sketch.viewGlyph"},
	} {
		s := newTestSketch(60, 20)
		var next tea.Model = openCommandPalette(t, s)
		for _, r := range tc.query {
			next = send(next, typed(string(r)))
		}
		next = send(next, tea.KeyPressMsg{Code: tea.KeyEnter})
		if got := typeName(next); got != tc.want {
			t.Errorf("%q: 화면 = %s, %s 여야 한다", tc.query, got, tc.want)
		}
	}
}

func TestPaletteDownPicksNext(t *testing.T) {
	s := newTestSketch(60, 20)
	next := send(openCommandPalette(t, s), tea.KeyPressMsg{Code: tea.KeyDown}, tea.KeyPressMsg{Code: tea.KeyEnter})
	if _, ok := next.(*viewGlyph); !ok {
		t.Errorf("화면 = %T, 글자표여야 한다", next)
	}
}

func TestPaletteClickRunsRow(t *testing.T) {
	s := newTestSketch(60, 20)
	palette := openCommandPalette(t, s)
	originX, originY := s.popupOrigin(palette.box())
	// 테두리 한 칸, 검색어 한 줄 아래 둘째 줄이 글자표다.
	next := send(palette, tea.MouseClickMsg{X: originX + 3, Y: originY + 1 + 2, Button: tea.MouseLeft})
	if _, ok := next.(*viewGlyph); !ok {
		t.Errorf("화면 = %T, 글자표여야 한다", next)
	}
}

// 팔레트가 떠 있는 동안 키는 검색어로 간다. 판의 모드나 붓이 바뀌면 안 된다.
func TestPaletteSwallowsKeys(t *testing.T) {
	s := newTestSketch(60, 20)
	brush, figure := s.brush, s.figure
	palette := openCommandPalette(t, s)
	send(palette, tea.KeyPressMsg{Code: tea.KeyTab}, typed("x"))
	if s.mode() != modeBrush || s.brush != brush || s.figure != figure {
		t.Errorf("모드 %s, 모양 %s, 붓 %+v 가 바뀌었다", s.mode(), s.figure, s.brush)
	}
	if palette.query.Value() != "x" {
		t.Errorf("검색어 = %q", palette.query.Value())
	}
	if next := send(palette, tea.KeyPressMsg{Code: tea.KeyEscape}); !onMode(next) {
		t.Error("esc 가 팔레트를 안 닫았다")
	}
}

func TestPaletteSaves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mock.json")
	s := newTestScreen(path, NewCanvas(10, 5))
	s.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	s.Update(canvasClick(s, 0, 0, tea.MouseLeft))
	var next tea.Model = openCommandPalette(t, s)
	for _, r := range "save" {
		next = send(next, typed(string(r)))
	}
	next = send(next, tea.KeyPressMsg{Code: tea.KeyEnter})
	if s.dirty || !onMode(next) {
		t.Errorf("저장 뒤 수정됨 %v, 화면 %T", s.dirty, next)
	}
	if _, err := Load(path); err != nil {
		t.Errorf("저장한 파일이 안 열린다: %v", err)
	}
}

// 창이 떠 있어도 ctrl 키는 판이 받는다. 저장 · 되돌리기는 창을 그대로 두고, ctrl+p 는 팔레트로 바꾼다.
func TestSketchKeysPassThroughPopups(t *testing.T) {
	s := newTestSketch(10, 5)
	s.Update(canvasClick(s, 0, 0, tea.MouseLeft))
	for _, open := range []tea.Msg{typed("c"), typed("v")} {
		popup := send(s, open)
		if next := send(popup, tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}); next != popup {
			t.Errorf("%T: ctrl+z 뒤 화면 = %T, 창이 그대로여야 한다", popup, next)
		}
		if next := send(popup, openPalette); typeName(next) != "*sketch.viewCommand" {
			t.Errorf("%T: ctrl+p 뒤 화면 = %T, 팔레트여야 한다", popup, next)
		}
	}
	if s.canvas.At(0, 0) != blank {
		t.Error("창 위의 ctrl+z 가 판을 안 되돌렸다")
	}
}
