// 글자 모드의 화면이다. 클릭으로 커서를 놓고 치는 대로 적는다. 글자 키가 전부 입력이라 도구 키가 안
// 먹고, Esc 가 들어오기 전 모드로 돌아가는 길이다. 모양은 안 쓴다. (ADR-0006)

package sketch

import (
	"slices"

	tea "charm.land/bubbletea/v2"
)

const textModeName = "글자"

// viewText 는 글자 모드가 들린 화면이다. back 은 Esc 로 돌아갈 모드 화면이다.
//
// 커서(cursorX · cursorY · lineStart)는 판(sketch)에 둔다. 글자 모드를 나갔다 들어와도 치던 곳에서
// 잇는다.
type viewText struct {
	*sketch
	back tea.Model
}

// openText 는 글자 모드로 들어간다. 이미 글자 모드면(팔레트로 다시 고른 것) 그대로 둔다. 새로 세우면
// 돌아갈 곳이 글자 모드 자신이 된다.
func openText(s *sketch, from tea.Model) (tea.Model, tea.Cmd) {
	if text, ok := from.(*viewText); ok {
		return text, nil
	}
	return &viewText{sketch: s, back: from}, nil
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
			// 모양은 글자 모드에서 쓰지 않지만 Tab 은 어느 모드에서나 모양을 돈다. 나가서 쓸 모양을 미리
			// 고를 수 있다.
			v.nextFigure()
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
