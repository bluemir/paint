// 도구(mode)다. 도구마다 이름 · 키 · 팔레트 이름 · 띠 안내를 들고, 판을 누르고 끌 때와 키를 칠 때
// 무엇을 하는지를 스스로 안다. 새 도구는 여기에 타입 하나를 두고 modes 에 올리면 도구 줄 · 팔레트 ·
// 단축키에 다 나온다. (ADR-0004)
//
// 브러시 · 칠하기 · 지우기는 모양(figure)을 따라 칸마다 무언가를 쓰는 도구라 figureMode 를 함께 쓰고
// 칸 하나에 하는 일(apply)과 건너 쓸 폭(step)만 다르다. 글자는 모양을 안 쓰고 키로 적는다.

package sketch

import (
	tea "charm.land/bubbletea/v2"
)

type mode interface {
	// String 은 도구 줄과 띠에 보이는 이름이다.
	String() string
	// shortcut 은 도구를 드는 한 글자 키다. 왼손이 닿는 키(qwert asdfg zxcvb)에서 고른다.
	shortcut() string
	// command 와 desc 는 명령 팔레트에 오르는 이름과 설명이다.
	command() string
	desc() string
	// label 은 띠에 적는 지금 도구와 마우스가 하는 일이다. 우클릭이 지우개라는 것이 안 보이면
	// 아무도 모른다.
	label(s *sketch) string

	// press 와 move 는 판 위의 누름과, 누른 채 움직인 것이다. 판 칸 (x, y) 를 받는다.
	press(s *sketch, button tea.MouseButton, x, y int)
	move(s *sketch, button tea.MouseButton, x, y int)
	// keyPress 는 창이 없을 때의 키다. Tab · ctrl 키 · 창을 닫는 Esc 는 여기 오기 전에 받는다.
	keyPress(s *sketch, msg tea.KeyPressMsg) tea.Cmd
	// cursor 는 터미널의 진짜 커서를 둘 곳이다. 없으면 nil 이다.
	cursor(s *sketch) *tea.Cursor

	// apply 는 canvas 의 한 칸에 도구를 쓴다. step 은 모양을 따라 가로로 몇 칸씩 건너 쓸지다.
	// 모양(shape.go)이 이 둘로 칸을 채운다.
	apply(s *sketch, canvas *Canvas, x, y int)
	step(s *sketch) int
}

var (
	modeBrush mode = brushMode{figureMode{name: "브러시", key: "b", commandName: "brush", description: "브러시 모드로", dotHint: "좌 찍음·우 지움"}}
	modeText  mode = textMode{}
	modePaint mode = paintMode{figureMode{name: "칠하기", key: "r", commandName: "paint", description: "칠하기 모드로 (글자는 두고 색만)", dotHint: "좌 색만 바꿈·우 지움"}}
	modeErase mode = eraseMode{figureMode{name: "지우기", key: "e", commandName: "erase", description: "지우기 모드로", dotHint: "좌·우 지움"}}
)

// modes 는 도구 목록이다. 목록 차례가 곧 도구 줄과 팔레트에 뜨는 차례다.
var modes = []mode{modeBrush, modeText, modePaint, modeErase}

// figureMode 는 모양을 따라 칸마다 쓰는 도구가 함께 쓰는 부분이다. 판의 누름과 끌기를 지금 모양에
// 넘기고, 키는 도구 키로 받는다. 칸 하나에 하는 일(apply)은 품은 쪽이 정한다.
type figureMode struct {
	name, key, commandName, description string
	// dotHint 는 한 칸씩 모양에서 마우스가 하는 일이다. 끄는 모양은 제 안내가 있다(figure.hint).
	dotHint string
}

func (m figureMode) String() string   { return m.name }
func (m figureMode) shortcut() string { return m.key }
func (m figureMode) command() string  { return m.commandName }
func (m figureMode) desc() string     { return m.description }

func (m figureMode) label(s *sketch) string {
	return m.name + "·" + s.figure.String() + "(" + s.figure.hint(m.dotHint) + ")"
}

// press 는 누름부터 뗌까지를 되돌리기 한 번으로 묶는다. 끌며 칠한 것이 한 번에 되돌아간다.
func (figureMode) press(s *sketch, button tea.MouseButton, x, y int) {
	s.beginEdit()
	s.figure.press(s, button, x, y)
}

func (figureMode) move(s *sketch, button tea.MouseButton, x, y int) {
	s.figure.move(s, button, x, y)
}

func (figureMode) keyPress(s *sketch, msg tea.KeyPressMsg) tea.Cmd { return s.toolKey(msg) }

func (figureMode) cursor(*sketch) *tea.Cursor { return nil }

// step 은 칸마다다. 브러시만 붓 폭으로 건넌다(brushMode.step).
func (figureMode) step(*sketch) int { return 1 }

// brushMode 는 붓(글자와 두 색)을 찍는다. 넓은 붓은 가로로 두 칸씩 건너 찍어 앞 글자의 반쪽을 덮지
// 않는다.
type brushMode struct{ figureMode }

func (brushMode) apply(s *sketch, canvas *Canvas, x, y int) { canvas.Put(x, y, s.brush) }
func (brushMode) step(s *sketch) int                        { return cellWidth(s.brush) }

// paintMode 는 글자를 두고 전경 · 배경색만 붓 색으로 바꾼다. 빈 칸에 쓰면 배경만 칠해진다.
type paintMode struct{ figureMode }

func (paintMode) apply(s *sketch, canvas *Canvas, x, y int) {
	canvas.Recolor(x, y, s.brush.Fg, s.brush.Bg)
}

// eraseMode 는 좌클릭으로도 지운다. 우클릭 지우개만으로는 그런 것이 있는 줄 몰라서 도구로도 둔다.
type eraseMode struct{ figureMode }

func (eraseMode) apply(_ *sketch, canvas *Canvas, x, y int) { canvas.Erase(x, y) }

// textMode 는 클릭으로 커서를 놓고 치는 대로 적는다. 글자 키가 전부 입력이라 도구 키가 안 먹고, Esc 가
// 직전 도구로 돌아가는 길이다. 모양은 안 쓴다.
type textMode struct{}

func (textMode) String() string   { return "글자" }
func (textMode) shortcut() string { return "t" }
func (textMode) command() string  { return "text" }
func (textMode) desc() string     { return "글자 모드로" }

func (m textMode) label(*sketch) string {
	return m.String() + "(클릭=커서·치면 적힘·Esc 나감)"
}

func (textMode) press(s *sketch, _ tea.MouseButton, x, y int) {
	s.moveCursor(x, y)
	s.lineStart = x
}

// move 는 하는 일이 없다. 커서는 누른 곳 한 칸이다.
func (textMode) move(*sketch, tea.MouseButton, int, int) {}

// keyPress 는 키 하나가 되돌리기 한 번이다.
func (textMode) keyPress(s *sketch, msg tea.KeyPressMsg) tea.Cmd {
	if msg.String() == "esc" {
		s.setMode(s.previousMode)
		return nil
	}
	s.beginEdit()
	s.textKey(msg)
	s.endEdit()
	return nil
}

// cursor 는 칸을 뒤집어 그리지 않고 터미널의 진짜 커서를 적힐 칸에 둔다.
//
// 입력기가 조합 중인 글자(ㅎ → 하 → 한)는 앱에 오지 않고 터미널이 제 커서 자리에 그린다. 커서가
// 화면 구석에 있으면 조합 중인 글자가 거기 떠서 안 보였다. 칸 위에 두면 적힐 곳에서 조립된다.
//
// 커서 칸이 굴려서 안 보이면 숨긴다.
func (textMode) cursor(s *sketch) *tea.Cursor {
	x, y := s.cursorX-s.left, s.cursorY-s.top
	if x < 0 || y < 0 || x >= s.canvasViewWidth() || y >= s.canvasViewHeight() {
		return nil
	}
	return tea.NewCursor(x+s.originX(), y+1)
}

// apply 와 step 은 쓰이지 않는다. 글자는 모양을 따라 쓰지 않고 키로 적는다(textKey).
func (textMode) apply(*sketch, *Canvas, int, int) {}
func (textMode) step(*sketch) int                 { return 1 }
