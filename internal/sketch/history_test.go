package sketch

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

var (
	undoKey = tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}
	redoKey = tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl}
)

// 누름부터 뗌까지 끌며 칠한 것이 한 번에 되돌아가고, 다시 하면 돌아온다.
func TestUndoRedoStroke(t *testing.T) {
	s := newTestSketch(10, 5)
	s.Update(canvasClick(s, 1, 1, tea.MouseLeft))
	s.Update(canvasDrag(s, 2, 1, tea.MouseLeft))
	s.Update(canvasDrag(s, 3, 1, tea.MouseLeft))
	s.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
	s.Update(undoKey)
	for x := 1; x <= 3; x++ {
		if s.canvas.At(x, 1) != blank {
			t.Errorf("되돌린 뒤 (%d,1) = %+v", x, s.canvas.At(x, 1))
		}
	}
	s.Update(redoKey)
	for x := 1; x <= 3; x++ {
		if s.canvas.At(x, 1) != s.brush {
			t.Errorf("다시 한 뒤 (%d,1) = %+v", x, s.canvas.At(x, 1))
		}
	}
}

// 글자 모드는 키 하나가 되돌리기 한 번이다. 넓은 글자는 두 칸이 함께 돌아온다.
func TestUndoTextOneKeyAtATime(t *testing.T) {
	s := newTestSketch(10, 3)
	s.setMode(modeText)
	s.Update(canvasClick(s, 0, 0, tea.MouseLeft))
	s.Update(typed("a"))
	s.Update(typed("한"))
	s.Update(undoKey)
	if s.canvas.At(0, 0).Glyph != "a" || s.canvas.At(1, 0) != blank || s.canvas.At(2, 0) != blank {
		t.Errorf("한 번 되돌린 뒤 = %+v %+v %+v", s.canvas.At(0, 0), s.canvas.At(1, 0), s.canvas.At(2, 0))
	}
	s.Update(undoKey)
	if s.canvas.At(0, 0) != blank {
		t.Errorf("두 번 되돌린 뒤 = %+v", s.canvas.At(0, 0))
	}
}

// 박스는 뗀 뒤 한 번에 되돌아간다.
func TestUndoBox(t *testing.T) {
	s := newTestSketch(10, 5)
	s.setFigure(figureBox)
	s.Update(canvasClick(s, 1, 1, tea.MouseLeft))
	s.Update(canvasDrag(s, 4, 3, tea.MouseLeft))
	s.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
	s.Update(undoKey)
	for y := range 5 {
		for x := range 10 {
			if s.canvas.At(x, y) != blank {
				t.Fatalf("되돌린 뒤 (%d,%d) = %+v", x, y, s.canvas.At(x, y))
			}
		}
	}
}

// 새 손길이 오르면 다시 하기는 버린다. 되돌릴 수 있는 것은 historyLimit 번이다.
func TestUndoLimitAndRedoCleared(t *testing.T) {
	s := newTestSketch(10, 5)
	for i := range historyLimit + 5 {
		s.Update(canvasClick(s, i%10, i/10%5, tea.MouseLeft))
		s.brush.Fg = Color(i % 200)
		s.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
	}
	if got := len(s.history.done); got != historyLimit {
		t.Errorf("되돌릴 수 있는 수 = %d, %d 여야 한다", got, historyLimit)
	}
	s.Update(undoKey)
	s.Update(canvasClick(s, 9, 4, tea.MouseRight))
	s.Update(tea.MouseReleaseMsg{Button: tea.MouseRight})
	if len(s.history.undone) != 0 {
		t.Error("새 손길 뒤에도 다시 하기가 남았다")
	}
}

// 판을 안 바꾼 누름(빈 칸 지우기)은 되돌리기에 오르지 않는다.
func TestNoopEditIsNotRecorded(t *testing.T) {
	s := newTestSketch(10, 5)
	s.Update(canvasClick(s, 1, 1, tea.MouseRight))
	s.Update(tea.MouseReleaseMsg{Button: tea.MouseRight})
	if len(s.history.done) != 0 {
		t.Errorf("바뀐 것 없는 손길이 올랐다: %d", len(s.history.done))
	}
}
