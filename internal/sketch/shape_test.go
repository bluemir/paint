package sketch

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func rows(canvas *Canvas) []string {
	return strings.Split(ansi.Strip(canvas.renderArea(0, 0, canvas.Width, canvas.Height)), "\n")
}

// stamp 은 브러시 도구로 모양을 찍는다. 붓 폭만큼 건넌다(toolStep).
func stamp(canvas *Canvas, f figure, d drag, brush Cell) {
	for _, at := range f.points(d, cellWidth(brush)) {
		canvas.Put(at.x, at.y, brush)
	}
}

func wantRows(t *testing.T, canvas *Canvas, want []string) {
	t.Helper()
	for y, row := range rows(canvas) {
		if row != want[y] {
			t.Errorf("%d 줄 = %q, %q 여야 한다", y, row, want[y])
		}
	}
}

func TestBoxStampsOutline(t *testing.T) {
	canvas := NewCanvas(6, 4)
	stamp(canvas, figureBox, drag{fromX: 4, fromY: 3, toX: 1, toY: 0}, Cell{Glyph: "#", Fg: NoColor, Bg: NoColor})
	wantRows(t, canvas, []string{" #### ", " #  # ", " #  # ", " #### "})
}

// 넓은 붓은 가로로 두 칸씩 건너 찍고, 오른쪽 끝을 넘지 않는다.
func TestBoxWithWideBrush(t *testing.T) {
	canvas := NewCanvas(8, 3)
	stamp(canvas, figureBox, drag{fromX: 0, fromY: 0, toX: 7, toY: 2}, Cell{Glyph: "한", Fg: NoColor, Bg: NoColor})
	wantRows(t, canvas, []string{"한한한한", "한    한", "한한한한"})
}

func TestFillFillsWhole(t *testing.T) {
	canvas := NewCanvas(5, 4)
	stamp(canvas, figureFill, drag{fromX: 3, fromY: 2, toX: 1, toY: 0}, Cell{Glyph: "#", Fg: NoColor, Bg: NoColor})
	wantRows(t, canvas, []string{" ### ", " ### ", " ### ", "     "})
}

// 직선은 끌어 간 거리가 긴 축을 따른다.
func TestLineFollowsLongerAxis(t *testing.T) {
	brush := Cell{Glyph: "-", Fg: NoColor, Bg: NoColor}
	canvas := NewCanvas(5, 3)
	stamp(canvas, figureLine, drag{fromX: 0, fromY: 1, toX: 4, toY: 2}, brush)
	if got := rows(canvas); got[1] != "-----" || got[2] != "     " {
		t.Errorf("가로선 = %q", got)
	}
	canvas = NewCanvas(5, 3)
	stamp(canvas, figureLine, drag{fromX: 2, fromY: 0, toX: 3, toY: 2}, brush)
	if got := rows(canvas); got[0] != "  -  " || got[2] != "  -  " {
		t.Errorf("세로선 = %q", got)
	}
}

// 끄는 동안은 판에 안 적히고 미리 보이기만 한다. 떼면 적힌다. 우클릭은 버린다.
func TestFigureDragPreviewsThenCommits(t *testing.T) {
	s := newTestSketch(10, 5)
	s.setFigure(figureBox)
	s.Update(canvasClick(s, 1, 1, tea.MouseLeft))
	s.Update(canvasDrag(s, 4, 3, tea.MouseLeft))
	if s.canvas.At(4, 3) != blank || s.dirty {
		t.Fatal("끄는 중에 판에 적혔다")
	}
	if !strings.Contains(ansi.Strip(s.View().Content), s.brush.Glyph+s.brush.Glyph) {
		t.Error("미리 보기가 안 보인다")
	}
	s.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
	if s.canvas.At(4, 3) != s.brush || !s.dirty {
		t.Errorf("뗀 뒤 (4,3) = %+v", s.canvas.At(4, 3))
	}

	s.Update(canvasClick(s, 6, 0, tea.MouseLeft))
	s.Update(canvasDrag(s, 8, 2, tea.MouseLeft))
	s.Update(canvasClick(s, 8, 2, tea.MouseRight))
	s.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
	if s.canvas.At(8, 2) != blank {
		t.Error("우클릭으로 버린 모양이 적혔다")
	}
}

// 모양은 도구와 따로다. 칠하기 · 지우기도 직선 · 채움을 쓴다. 칠하기는 글자를 두고 색만 바꾼다.
func TestFigureWorksWithEveryTool(t *testing.T) {
	s := newTestSketch(10, 5)
	s.setFigure(figureFill)
	s.Update(canvasClick(s, 0, 0, tea.MouseLeft))
	s.Update(canvasDrag(s, 5, 2, tea.MouseLeft))
	s.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})

	s.setMode(modePaint)
	s.brush.Fg = 196
	s.setFigure(figureLine)
	s.Update(canvasClick(s, 0, 1, tea.MouseLeft))
	s.Update(canvasDrag(s, 5, 1, tea.MouseLeft))
	s.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
	for x := range 6 {
		if got := s.canvas.At(x, 1); got.Glyph != "#" || got.Fg != 196 {
			t.Errorf("칠하기 직선 (%d,1) = %+v", x, got)
		}
	}

	s.setMode(modeErase)
	s.setFigure(figureBox)
	s.Update(canvasClick(s, 0, 0, tea.MouseLeft))
	s.Update(canvasDrag(s, 5, 2, tea.MouseLeft))
	s.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
	if s.canvas.At(0, 0) != blank || s.canvas.At(3, 1) == blank {
		t.Errorf("지우기 테두리: 모서리 %+v, 안쪽 %+v", s.canvas.At(0, 0), s.canvas.At(3, 1))
	}
}

// Tab 은 모양을 돈다. 도구는 그대로다.
func TestTabCyclesFigures(t *testing.T) {
	s := newTestSketch(10, 5)
	s.setMode(modePaint)
	for _, want := range []figure{figureLine, figureBox, figureFill, figureDot} {
		s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		if s.figure != want || s.mode != modePaint {
			t.Errorf("모양 %s, 도구 %s", s.figure, s.mode)
		}
	}
}
