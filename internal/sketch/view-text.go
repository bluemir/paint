// 글자 모드의 화면이다. 클릭으로 커서를 놓고 치는 대로 적는다. 글자 키가 전부 입력이라 도구 키가 안
// 먹고, Esc 가 들어오기 전 모드로 돌아가는 길이다. 모양은 안 쓴다. (ADR-0006)

package sketch

import (
	"slices"

	tea "charm.land/bubbletea/v2"
)

const textModeName = "글자"

// viewText 는 글자 모드가 들린 화면이다. back 은 Esc 로 돌아갈 모드 화면이다.
type viewText struct {
	*sketch
	back tea.Model

	// 커서와, Enter 가 돌아갈 열이다. 글자 모드 안에서만 사는 상태라 판이 아니라 여기 둔다. 들어올 때마다
	// 마우스가 짚은 칸에서 시작한다(openText).
	cursorX, cursorY, lineStart int
}

// openText 는 글자 모드로 들어간다. 커서는 마우스가 짚은 칸이고, 판 밖이면 맨 앞(0,0)이다. 키로 들어온
// 뒤 바로 치면 마우스 밑에 적힌다.
//
// 이미 글자 모드면(팔레트로 다시 고른 것) 그대로 둔다. 새로 세우면 돌아갈 곳이 글자 모드 자신이 된다.
func openText(s *sketch, from tea.Model) (tea.Model, tea.Cmd) {
	if text, ok := from.(*viewText); ok {
		return text, nil
	}
	text := &viewText{sketch: s, back: from}
	if s.hoverX >= 0 {
		text.moveCursor(s.hoverX, s.hoverY)
		text.lineStart = s.hoverX
	}
	return text, nil
}

func (v *viewText) Init() tea.Cmd { return nil }

func (v *viewText) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.resize(msg)
	case tea.KeyPressMsg:
		v.forgetQuit(msg)
		switch {
		case slices.Contains(sketchKeys, msg.String()):
			return v.sketchKey(v, msg)
		case msg.String() == "tab":
			// 모양을 안 쓰므로 Tab 은 하는 일이 없다. 칸에 둘 수 없는 제어 문자라 적지도 않는다.
			return v, nil
		case msg.String() == "esc":
			return v.back, nil
		}
		// 키 하나가 되돌리기 한 번이다.
		v.beginEdit()
		v.textKey(msg)
		v.endEdit()
	case tea.MouseClickMsg:
		mouse := tea.Mouse(msg)
		if v.onChrome(mouse) {
			return v.clickChrome(v, v.label(), mouse)
		}
		if x, y, ok := v.canvasAt(mouse); ok {
			v.moveCursor(x, y)
			v.lineStart = x
		}
	case tea.MouseMotionMsg:
		// 커서는 누른 곳 한 칸이다. 움직임은 짚은 칸만 적는다.
		v.hover(tea.Mouse(msg))
	case tea.MouseWheelMsg:
		v.wheel(tea.Mouse(msg))
	}
	return v, nil
}

func (v *viewText) View() tea.View {
	return v.render(v.canvas, textModeName, v.label(), v.cursor())
}

func (v *viewText) label() string { return textModeName + "(클릭=커서·치면 적힘·Esc 나감)" }

// cursor 는 칸을 뒤집어 그리지 않고 터미널의 진짜 커서를 적힐 칸에 둔다.
//
// 입력기가 조합 중인 글자(ㅎ → 하 → 한)는 앱에 오지 않고 터미널이 제 커서 자리에 그린다. 커서가
// 화면 구석에 있으면 조합 중인 글자가 거기 떠서 안 보였다. 칸 위에 두면 적힐 곳에서 조립된다.
//
// 커서 칸이 굴려서 안 보이면 숨긴다.
func (v *viewText) cursor() *tea.Cursor {
	x, y := v.cursorX-v.left, v.cursorY-v.top
	if x < 0 || y < 0 || x >= v.canvasViewWidth() || y >= v.canvasViewHeight() {
		return nil
	}
	return tea.NewCursor(x+v.originX(), y+1)
}

// textKey 는 글자 모드의 키다. 방향키는 커서를 옮기고, 글자 키는 커서 칸에 적는다.
func (v *viewText) textKey(msg tea.KeyPressMsg) {
	switch msg.String() {
	case "up":
		v.moveCursor(v.cursorX, v.cursorY-1)
	case "down":
		v.moveCursor(v.cursorX, v.cursorY+1)
	case "left":
		v.moveCursor(v.cursorX-1, v.cursorY)
	case "right":
		v.moveCursor(v.cursorX+1, v.cursorY)
	case "enter":
		v.moveCursor(v.lineStart, v.cursorY+1)
	case "backspace":
		if v.cursorX == 0 {
			return
		}
		x := v.cursorX - 1
		// 왼쪽이 넓은 글자의 오른쪽 반쪽이면 글자 전체 앞으로 물러선다.
		if v.canvas.At(x, v.cursorY).continuation() && x > 0 {
			x--
		}
		v.canvas.Erase(x, v.cursorY)
		v.dirty = true
		v.moveCursor(x, v.cursorY)
	default:
		for _, r := range typedGlyph(msg) {
			width := v.canvas.Put(v.cursorX, v.cursorY, Cell{Glyph: string(r), Fg: v.brush.Fg, Bg: v.brush.Bg})
			if width == 0 {
				return
			}
			v.dirty = true
			v.moveCursor(v.cursorX+width, v.cursorY)
		}
	}
}

// moveCursor 는 커서를 판 안으로 옮기고 그 칸이 보이게 판을 굴린다.
func (v *viewText) moveCursor(x, y int) {
	v.cursorX = min(max(x, 0), v.canvas.Width-1)
	v.cursorY = min(max(y, 0), v.canvas.Height-1)
	switch {
	case v.cursorX < v.left:
		v.left = v.cursorX
	case v.cursorX >= v.left+v.canvasViewWidth():
		v.left = v.cursorX - v.canvasViewWidth() + 1
	}
	switch {
	case v.cursorY < v.top:
		v.top = v.cursorY
	case v.cursorY >= v.top+v.canvasViewHeight():
		v.top = v.cursorY - v.canvasViewHeight() + 1
	}
}
