package sketch

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var openPalette = tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl}

func TestPaletteOpensColorAndGlyph(t *testing.T) {
	for _, tc := range []struct {
		query string
		want  popup
	}{
		{"", popupColor}, // 걸러지기 전 첫 줄이 색표다
		{"색", popupColor},
		{"글자", popupGlyph},
		{"glyph", popupGlyph},
	} {
		s := newTestSketch(60, 20)
		s.Update(openPalette)
		for _, r := range tc.query {
			s.Update(typed(string(r)))
		}
		s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		if s.popup != tc.want {
			t.Errorf("%q: 창 = %d, %d 여야 한다", tc.query, s.popup, tc.want)
		}
	}
}

func TestPaletteDownPicksNext(t *testing.T) {
	s := newTestSketch(60, 20)
	s.Update(openPalette)
	s.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if s.popup != popupGlyph {
		t.Errorf("창 = %d, 글자표여야 한다", s.popup)
	}
}

func TestPaletteClickRunsRow(t *testing.T) {
	s := newTestSketch(60, 20)
	s.Update(openPalette)
	originX, originY := s.popupOrigin(s.commandBox())
	// 테두리 한 칸, 검색어 한 줄 아래 둘째 줄이 글자표다.
	s.Update(tea.MouseClickMsg{X: originX + 3, Y: originY + 1 + 2, Button: tea.MouseLeft})
	if s.popup != popupGlyph {
		t.Errorf("창 = %d, 글자표여야 한다", s.popup)
	}
}

// 팔레트가 떠 있는 동안 키는 검색어로 간다. 판의 모드나 붓이 바뀌면 안 된다.
func TestPaletteSwallowsKeys(t *testing.T) {
	s := newTestSketch(60, 20)
	brush := s.brush
	s.Update(openPalette)
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	s.Update(typed("x"))
	if s.mode != modePaint || s.brush != brush {
		t.Errorf("모드 %s, 붓 %+v 가 바뀌었다", s.mode, s.brush)
	}
	if s.query.Value() != "x" {
		t.Errorf("검색어 = %q", s.query.Value())
	}
	s.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if s.popup != popupNone {
		t.Error("esc 가 팔레트를 안 닫았다")
	}
}

func TestPaletteSaves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mock.json")
	s := newSketch(path, NewCanvas(10, 5))
	s.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	s.Update(canvasClick(s, 0, 0, tea.MouseLeft))
	s.Update(openPalette)
	for _, r := range "save" {
		s.Update(typed(string(r)))
	}
	s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if s.dirty || s.popup != popupNone {
		t.Errorf("저장 뒤 수정됨 %v, 창 %d", s.dirty, s.popup)
	}
	if _, err := Load(path); err != nil {
		t.Errorf("저장한 파일이 안 열린다: %v", err)
	}
}
