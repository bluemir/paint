// 지우기 모드의 화면이다. 좌클릭으로도 지운다. 우클릭 지우개만으로는 그런 것이 있는 줄 몰라서 도구로도
// 둔다. (ADR-0006)
//
//   - 한 칸씩: 누른 채 지나간 칸마다 지운다.
//   - 직선 · 테두리 · 채움: 누른 곳에서 끌고 가는 동안 판의 사본에 모양을 그려 미리 보이고, 버튼을 떼면
//     판에 적는다. 끄는 중 우클릭은 끌던 것을 버린다. 채움이 곧 영역 지우기다.

package sketch

import (
	"slices"

	tea "charm.land/bubbletea/v2"
)

const eraseModeName = "지우기"

type viewErase struct {
	*sketch
}

func openErase(s *sketch, _ tea.Model) (tea.Model, tea.Cmd) {
	s.dragging = nil
	return &viewErase{sketch: s}, nil
}

func (v *viewErase) Init() tea.Cmd { return nil }

func (v *viewErase) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.resize(msg)
	case tea.KeyPressMsg:
		v.forgetQuit(msg)
		switch {
		case slices.Contains(sketchKeys, msg.String()):
			return v.sketchKey(v, msg)
		case msg.String() == "tab":
			v.nextFigure()
			return v, nil
		}
		return v.toolKey(v, msg)
	case tea.MouseClickMsg:
		mouse := tea.Mouse(msg)
		if v.onChrome(mouse) {
			return v.clickChrome(v, v.label(), mouse)
		}
		x, y, ok := v.canvasAt(mouse)
		if !ok {
			return v, nil
		}
		// 누름부터 뗌까지가 되돌리기 한 번이다. 끌며 지운 것이 한 번에 되돌아간다.
		v.beginEdit()
		switch {
		case v.figure == figureDot:
			v.stroke(msg.Button, x, y)
		case msg.Button == tea.MouseLeft:
			v.dragging = &drag{fromX: x, fromY: y, toX: x, toY: y}
		case msg.Button == tea.MouseRight && v.dragging != nil:
			v.dragging = nil
		default:
			v.stroke(msg.Button, x, y)
		}
	case tea.MouseMotionMsg:
		x, y, ok := v.hover(tea.Mouse(msg))
		switch {
		case !ok:
		case v.figure == figureDot:
			v.stroke(msg.Button, x, y)
		case v.dragging != nil:
			// 끄는 동안 끝만 옮긴다. 판에는 버튼을 뗄 때 적는다.
			v.dragging.toX, v.dragging.toY = x, y
		}
	case tea.MouseReleaseMsg:
		// 끌던 모양을 판에 적는다. 판 밖에서 떼도 마지막으로 짚은 칸까지다.
		if v.dragging != nil {
			v.drawDrag(v.canvas)
			v.dragging = nil
			v.dirty = true
		}
		v.endEdit()
	case tea.MouseWheelMsg:
		v.wheel(tea.Mouse(msg))
	}
	return v, nil
}

func (v *viewErase) View() tea.View {
	canvas := v.canvas
	if v.dragging != nil {
		canvas = v.canvas.clone()
		v.drawDrag(canvas)
	}
	return v.render(canvas, eraseModeName, v.label(), nil)
}

// label 은 띠에 적는 도구와 모양, 그리고 마우스가 하는 일이다. 우클릭이 지우개라는 것이 안 보이면
// 아무도 모른다.
func (v *viewErase) label() string {
	hint := "좌·우 지움"
	if v.figure != figureDot {
		hint = "끌어 그림·우 취소"
	}
	return eraseModeName + "·" + v.figure.String() + "(" + hint + ")"
}

// stroke 는 한 칸씩 모양에서 버튼이 눌린 채 지나간 칸에 손을 댄다. 좌 · 우 모두 지운다.
func (v *viewErase) stroke(button tea.MouseButton, x, y int) {
	switch button {
	case tea.MouseLeft, tea.MouseRight:
		v.canvas.Erase(x, y)
	default:
		return
	}
	v.dirty = true
}

// drawDrag 는 끌고 있는 모양을 canvas 에 지운다. 칸마다다.
func (v *viewErase) drawDrag(canvas *Canvas) {
	for _, at := range v.figure.points(*v.dragging, 1) {
		canvas.Erase(at.x, at.y)
	}
}
