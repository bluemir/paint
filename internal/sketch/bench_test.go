package sketch

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// 120x40 빈 판을 한 번 그리는 값이다. 마우스를 움직이거나 굴릴 때마다 이만큼 든다. 칸마다 스타일을
// 렌더하던 때는 2.8ms 였고 같은 색 칸을 묶어 칠한 뒤 0.3ms 다(renderArea).
func BenchmarkViewCanvas(b *testing.B) {
	s := newTestScreen("unused.json", NewCanvas(120, 40))
	s.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	for b.Loop() {
		s.View()
	}
}

// 글자표를 한 칸 굴리는 값이다(그리기는 빼고). 굴릴 때마다 글자표를 다시 만들던 때는 0.38ms 였다
// (glyphGroups). 창이 화면 80% · 60% 로 커진 뒤 마우스 자리를 재려고 창을 그리던 때는 1.9ms 였고,
// 크기로 재게 한 뒤 0.3ms 다(viewGlyph.boxSize).
func BenchmarkGlyphWheel(b *testing.B) {
	s := newTestScreen("unused.json", NewCanvas(120, 40))
	s.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	table := send(s, typed("v"))
	for b.Loop() {
		table.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	}
}
