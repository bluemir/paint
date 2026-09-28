// 칠하기 모드의 화면이다. 글자는 두고 전경 · 배경색만 붓 색으로 바꾼다. 다 그린 뒤 색만 고쳐 보는 일이
// 잦아서 있다. 빈 칸에 쓰면 배경만 칠해진다. 우클릭은 지운다. (ADR-0006)
//
//   - 한 칸씩: 누른 채 지나간 칸마다 칠한다.
//   - 직선 · 테두리 · 채움: 누른 곳에서 끌고 가는 동안 판의 사본에 모양을 그려 미리 보이고, 버튼을 떼면
//     판에 적는다. 끄는 중 우클릭은 끌던 것을 버린다.
//
// 칸마다 칠한다. 넓은 글자는 어느 반쪽을 칠하든 두 칸이 함께 바뀐다(Canvas.Recolor).

package sketch

import (
	"slices"

	tea "charm.land/bubbletea/v2"
)

const paintModeName = "칠하기"

type viewPaint struct {
	*sketch
}

func openPaint(s *sketch, _ tea.Model) (tea.Model, tea.Cmd) {
	s.dragging = nil
	return &viewPaint{sketch: s}, nil
}

func (v *viewPaint) Init() tea.Cmd { return nil }

func (v *viewPaint) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		// 누름부터 뗌까지가 되돌리기 한 번이다. 끌며 칠한 것이 한 번에 되돌아간다.
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

func (v *viewPaint) View() tea.View {
	canvas := v.canvas
	if v.dragging != nil {
		canvas = v.canvas.clone()
		v.drawDrag(canvas)
	}
	return v.render(canvas, paintModeName, v.label(), nil)
}

// label 은 띠에 적는 도구와 모양, 그리고 마우스가 하는 일이다. 우클릭이 지우개라는 것이 안 보이면
// 아무도 모른다.
func (v *viewPaint) label() string {
	hint := "좌 색만 바꿈·우 지움"
	if v.figure != figureDot {
		hint = "끌어 그림·우 취소"
	}
	return paintModeName + "·" + v.figure.String() + "(" + hint + ")"
}

// stroke 는 한 칸씩 모양에서 버튼이 눌린 채 지나간 칸에 손을 댄다. 좌클릭은 색만 바꾸고 우클릭은 지운다.
func (v *viewPaint) stroke(button tea.MouseButton, x, y int) {
	switch button {
	case tea.MouseLeft:
		v.canvas.Recolor(x, y, v.brush.Fg, v.brush.Bg)
	case tea.MouseRight:
		v.canvas.Erase(x, y)
	default:
		return
	}
	v.dirty = true
}

// drawDrag 는 끌고 있는 모양을 canvas 에 칠한다. 칸마다다.
func (v *viewPaint) drawDrag(canvas *Canvas) {
	for _, at := range v.figure.points(*v.dragging, 1) {
		canvas.Recolor(at.x, at.y, v.brush.Fg, v.brush.Bg)
	}
}
