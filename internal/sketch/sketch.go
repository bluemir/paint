// sketch 는 터미널 그림판이다. 칸 단위로 글자와 256색을 칠한다.
//
// 텍스트 목업으로는 터미널 색을 입힐 수 없어서 있다. 마우스로 글자와 256색을 칠해 파일로 남기고,
// 그 파일을 다시 열어 고친다. (ADR-0001)
//
// 도구(mode.go)는 넷이고 키로 고르며, Tab 은 모양(shape.go)을 돈다. 브러시에서는 좌클릭이 붓을 찍고 우클릭이 지운다. 글자에서는
// 클릭으로 커서를 놓고 치는 대로 적힌다. 칠하기에서는 글자를 두고 색만 붓 색으로 바꾼다. 지우기에서는
// 좌클릭도 지운다. 붓 글자는 글자표(v)에서 고른다. 색표(c)와 글자표는 판 위에 뜨는 창이고, 명령
// 팔레트(ctrl+p)와 도구 줄로도 열린다. 도구 키는 toolKey 에 있다.

package sketch

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/cockroachdb/errors"

	"github.com/bluemir/paint/internal/tui/components"
)

func Run(ctx context.Context, path string, canvas *Canvas) error {
	_, err := tea.NewProgram(newSketch(path, canvas), tea.WithContext(ctx)).Run()
	return errors.WithStack(err)
}

type popup int

const (
	popupNone popup = iota
	popupColor
	popupGlyph
	popupCommand
)

type sketch struct {
	path   string
	canvas *Canvas

	width, height int // 터미널 크기. 맨 아래 한 줄은 띠다.
	left, top     int // 판의 어디부터 보이는지

	mode    mode
	figure  figure
	history history

	// dragging 은 직선 · 테두리 · 채움에서 끌고 있는 모양이다. 끌고 있지 않으면 nil 이다. (shape.go)
	dragging *drag

	// previousMode 는 지금 도구 직전에 들었던 도구다. 글자 모드에서 Esc 로 여기로 돌아간다.
	previousMode mode
	popup        popup
	brush        Cell

	// 명령 팔레트의 검색어와 걸러진 목록에서 짚은 줄이다. 팔레트가 열릴 때마다 비운다.
	query         components.Text
	commandCursor int

	// glyphTop 은 글자표를 몇 줄 내려 보는지다. 창을 닫았다 열어도 보던 곳에 있다.
	glyphTop int
	// glyphQuery 는 글자표 검색어다. 글자표를 열 때마다 비운다. glyphSearching 은 검색 줄에 손이 가
	// 있는지다. 가 있는 동안은 글자 키가 붓이 아니라 검색어로 간다. (glyphs.go)
	glyphQuery     components.Text
	glyphSearching bool
	// glyphHoverRow 와 glyphHoverColumn 은 글자표에서 마우스가 올라간 칸이다(glyphAt). 없으면 -1.
	glyphHoverRow, glyphHoverColumn int

	// 글자 모드의 커서와, Enter 가 돌아갈 열이다.
	cursorX, cursorY, lineStart int

	hoverX, hoverY int // 마우스가 짚은 판 위의 칸. 판 밖이면 -1.

	dirty     bool
	quitAsked bool // 저장 안 한 채 ctrl+c 를 한 번 눌렀다
	// imeOn 은 도구 키가 입력기를 거쳐 조합 글자로 왔는지다. 띠에 안내를 띄운다. 라틴 글자가 그대로
	// 오면 끈 것이다. 판정은 ime.go 에 있다.
	imeOn   bool
	message string
}

func newSketch(path string, canvas *Canvas) *sketch {
	return &sketch{
		path:         path,
		canvas:       canvas,
		mode:         modeBrush,
		previousMode: modeBrush,
		figure:       figureDot,
		brush:        Cell{Glyph: "#", Fg: NoColor, Bg: NoColor},
		hoverX:       -1, hoverY: -1,
	}
}

func (s *sketch) Init() tea.Cmd { return nil }

// viewHeight 는 띠를 뺀 화면 줄 수다. 창(색표 등)은 이 안 가운데에 뜬다.
func (s *sketch) viewHeight() int { return max(s.height-1, 0) }

// 판은 화면 (originX, 1) 에서 시작한다. 왼쪽 끝이 도구 줄이고(toolbar.go), 그 오른쪽 gutter 칸이 줄
// 번호, 위 한 줄이 열 눈금이다(ruler.go). 화면 칸과 판 칸을 바꾸는 곳은 전부 이것들을 본다.
func (s *sketch) gutter() int { return len(strconv.Itoa(s.canvas.Height-1)) + 1 }

func (s *sketch) originX() int { return toolbarWidth + s.gutter() }

func (s *sketch) canvasViewWidth() int { return max(s.width-s.originX(), 0) }

func (s *sketch) canvasViewHeight() int { return max(s.viewHeight()-1, 0) }

func (s *sketch) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width, s.height = msg.Width, msg.Height
		s.scrollBy(0, 0)
		return s, nil
	case tea.KeyPressMsg:
		return s.key(msg)
	case tea.MouseClickMsg:
		return s, s.click(tea.Mouse(msg))
	case tea.MouseMotionMsg:
		s.motion(tea.Mouse(msg))
		return s, nil
	case tea.MouseReleaseMsg:
		s.release()
		s.endEdit()
		return s, nil
	case tea.MouseWheelMsg:
		s.wheel(tea.Mouse(msg))
		return s, nil
	}
	// 캐럿 깜빡임 같은 비-키 메시지다. 검색어가 안 받으면 캐럿이 멈춘다.
	switch {
	case s.popup == popupCommand:
		return s, s.query.Update(msg)
	case s.popup == popupGlyph && s.glyphSearching:
		return s, s.glyphQuery.Update(msg)
	}
	return s, nil
}

// save 는 판을 파일에 쓴다. 되든 안 되든 띠에 적는다. 못 써도 그리던 것은 그대로 있어야 하므로
// 끝내지 않는다.
func (s *sketch) save() {
	if err := s.canvas.Save(s.path); err != nil {
		s.message = "저장 못 함: " + err.Error()
		return
	}
	s.dirty = false
	s.message = "저장했다: " + s.path
}

// openPopup 은 창을 바꾼다. 팔레트를 열면 검색어를 비워 깜빡임을 시작하고, 닫으면 그것을 끊는다.
func (s *sketch) openPopup(next popup) tea.Cmd {
	switch s.popup {
	case popupCommand:
		s.query.Blur()
	case popupGlyph:
		s.stopGlyphSearch()
	}
	s.popup = next
	if next == popupGlyph {
		s.glyphHoverRow, s.glyphHoverColumn = -1, -1
		s.glyphQuery = components.NewText("")
		s.glyphQuery.Blur() // 검색 줄을 누를 때까지는 글자 키가 붓이다
	}
	if next == popupCommand {
		s.query = components.NewText("")
		s.commandCursor = 0
		return s.query.Focus()
	}
	return nil
}

func (s *sketch) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() != "ctrl+c" {
		s.quitAsked = false
	}
	switch msg.String() {
	case "ctrl+c":
		if s.dirty && !s.quitAsked {
			s.quitAsked = true
			s.message = "저장 안 한 것이 있다. ctrl+c 를 한 번 더 누르면 버리고 끝낸다"
			return s, nil
		}
		return s, tea.Quit
	case "ctrl+s":
		s.save()
		return s, nil
	case "ctrl+z":
		s.undo()
		return s, nil
	case "ctrl+y":
		s.redo()
		return s, nil
	case "ctrl+p":
		return s, s.openPopup(popupCommand)
	case "esc":
		// 창이 떠 있으면 창을 먼저 닫는다. 창이 없으면 도구가 받는다(글자 모드는 그것으로 나간다).
		if s.popup == popupGlyph && s.glyphSearching {
			s.stopGlyphSearch()
			return s, nil
		}
		if s.popup != popupNone {
			return s, s.openPopup(popupNone)
		}
	}
	// 팔레트가 떠 있으면 나머지 키는 전부 검색어로 간다. Tab 이나 글자 키가 판에 닿으면 안 된다.
	if s.popup == popupCommand {
		return s, s.commandKey(msg)
	}
	// 글자표에서 Tab 은 검색 줄에 손을 대고 뗀다. 모드를 돌리지 않는다. 글자표가 떠 있는 동안에는
	// 판을 안 만지므로 모드가 바뀌어도 보이지 않고, 검색 줄을 누르러 마우스를 옮기지 않아도 된다.
	if s.popup == popupGlyph && msg.String() == "tab" {
		if s.glyphSearching {
			s.stopGlyphSearch()
			return s, nil
		}
		return s, s.startGlyphSearch()
	}
	if s.popup == popupGlyph && s.glyphSearching {
		return s, s.glyphSearchKey(msg)
	}
	switch msg.String() {
	case "tab":
		next := (slices.Index(figures, s.figure) + 1) % len(figures)
		s.setFigure(figures[next])
		return s, nil
	}
	if s.popup == popupGlyph {
		s.glyphKey(msg)
		return s, nil
	}
	if s.popup != popupNone {
		return s, nil
	}
	return s, s.mode.keyPress(s, msg)
}

// toolKey 는 글자 모드가 아닐 때의 키다. 한 글자 키가 도구를 들고(mode.shortcut), wasd 와 방향키가
// 판을 굴린다. 도구 줄의 오른쪽에 같은 키가 적혀 있다(toolbar.go).
//
// 글자 키는 붓 글자를 안 바꾼다. 바꾸던 때는 잘못 누른 키나 켜 둔 입력기 때문에 모르는 새 붓이 바뀌어
// 있었다. 붓 글자는 글자표를 연 동안에만 키로 고른다(glyphKey).
func (s *sketch) toolKey(msg tea.KeyPressMsg) tea.Cmd {
	// 글자 모드에서 한글을 치다 Esc 로 나오면 입력기가 켜진 채라 b 가 ㅠ 로 온다. 키가 안 먹는
	// 까닭이 안 보이므로 띠에 적는다. 글자 모드와 글자표는 한글이 곧 뜻이라 여기서만 잰다.
	switch {
	case imeText(msg):
		s.imeOn = true
	case asciiLetter(msg):
		s.imeOn = false
	}
	for _, m := range modes {
		if msg.String() == m.shortcut() {
			s.setMode(m)
			return nil
		}
	}
	switch msg.String() {
	case "q":
		s.pickHovered()
	case "c":
		return s.openPopup(popupColor)
	case "v":
		return s.openPopup(popupGlyph)
	case "w", "up":
		s.scrollBy(0, -1)
	case "s", "down":
		s.scrollBy(0, 1)
	case "a", "left":
		s.scrollBy(-1, 0)
	case "d", "right":
		s.scrollBy(1, 0)
	}
	return nil
}

// setMode 는 모드를 바꾼다. 바뀌면 직전 모드를 적어 둔다(글자 모드의 Esc 가 돌아갈 곳). 끌던 모양은
// 버린다. 모드가 바뀐 뒤 버튼을 떼면 무엇을 그릴지 모른다.
func (s *sketch) setMode(next mode) {
	s.dragging = nil
	if next != s.mode {
		s.previousMode = s.mode
	}
	s.mode = next
}

// textKey 는 글자 모드의 키다. 방향키는 커서를 옮기고, 글자 키는 커서 칸에 적는다.
func (s *sketch) textKey(msg tea.KeyPressMsg) {
	switch msg.String() {
	case "up":
		s.moveCursor(s.cursorX, s.cursorY-1)
	case "down":
		s.moveCursor(s.cursorX, s.cursorY+1)
	case "left":
		s.moveCursor(s.cursorX-1, s.cursorY)
	case "right":
		s.moveCursor(s.cursorX+1, s.cursorY)
	case "enter":
		s.moveCursor(s.lineStart, s.cursorY+1)
	case "backspace":
		if s.cursorX == 0 {
			return
		}
		x := s.cursorX - 1
		// 왼쪽이 넓은 글자의 오른쪽 반쪽이면 글자 전체 앞으로 물러선다.
		if s.canvas.At(x, s.cursorY).continuation() && x > 0 {
			x--
		}
		s.canvas.Erase(x, s.cursorY)
		s.dirty = true
		s.moveCursor(x, s.cursorY)
	default:
		for _, r := range typedGlyph(msg) {
			width := s.canvas.Put(s.cursorX, s.cursorY, Cell{Glyph: string(r), Fg: s.brush.Fg, Bg: s.brush.Bg})
			if width == 0 {
				return
			}
			s.dirty = true
			s.moveCursor(s.cursorX+width, s.cursorY)
		}
	}
}

// typedGlyph 는 키가 친 글자다. 조합키가 붙은 키와 이름만 있는 키(방향키 등)는 글자가 없다.
func typedGlyph(msg tea.KeyPressMsg) string {
	if msg.Mod&(tea.ModCtrl|tea.ModAlt) != 0 {
		return ""
	}
	if ansi.StringWidth(msg.Text) < 1 {
		return ""
	}
	return msg.Text
}

// moveCursor 는 커서를 판 안으로 옮기고 그 칸이 보이게 판을 굴린다.
func (s *sketch) moveCursor(x, y int) {
	s.cursorX = min(max(x, 0), s.canvas.Width-1)
	s.cursorY = min(max(y, 0), s.canvas.Height-1)
	switch {
	case s.cursorX < s.left:
		s.left = s.cursorX
	case s.cursorX >= s.left+s.canvasViewWidth():
		s.left = s.cursorX - s.canvasViewWidth() + 1
	}
	switch {
	case s.cursorY < s.top:
		s.top = s.cursorY
	case s.cursorY >= s.top+s.canvasViewHeight():
		s.top = s.cursorY - s.canvasViewHeight() + 1
	}
}

func (s *sketch) scrollBy(dx, dy int) {
	s.left = min(max(s.left+dx, 0), max(s.canvas.Width-s.canvasViewWidth(), 0))
	s.top = min(max(s.top+dy, 0), max(s.canvas.Height-s.canvasViewHeight(), 0))
}

// canvasAt 은 화면 칸을 판 칸으로 바꾼다. 눈금 위, 띠 위, 판 밖이면 ok 가 false 다.
func (s *sketch) canvasAt(mouse tea.Mouse) (x, y int, ok bool) {
	x, y = mouse.X-s.originX(), mouse.Y-1
	if x < 0 || y < 0 || x >= s.canvasViewWidth() || y >= s.canvasViewHeight() {
		return 0, 0, false
	}
	x, y = x+s.left, y+s.top
	return x, y, s.canvas.inside(x, y)
}

func (s *sketch) click(mouse tea.Mouse) tea.Cmd {
	switch s.popup {
	case popupColor:
		s.pickColor(mouse)
		return nil
	case popupGlyph:
		return s.pickGlyph(mouse)
	case popupCommand:
		s.clickCommand(mouse)
		return nil
	}
	if mouse.Y == s.height-1 {
		return s.clickStrip(mouse)
	}
	if mouse.X < toolbarWidth {
		return s.clickToolbar(mouse)
	}
	x, y, ok := s.canvasAt(mouse)
	if !ok {
		return nil
	}
	s.mode.press(s, mouse.Button, x, y)
	return nil
}

// takeCell 은 칸의 글자와 두 색을 붓에 담는다. 넓은 글자의 오른쪽 반쪽을 짚으면 그 글자를 담는다.
func (s *sketch) takeCell(x, y int) {
	if s.canvas.At(x, y).continuation() && x > 0 {
		x--
	}
	s.brush = s.canvas.At(x, y)
}

// pickHovered 는 스포이드다(q 키). 모드를 바꾸지 않고 마우스가 짚은 칸을 바로 붓에 담는다. 글자
// 모드가 아니면 어느 모드에서나 먹는다. 판 밖을 짚고 있으면 띠에 적는다.
//
// 스포이드를 모드로 두지 않는다. 한때 모드였는데(누르고 판을 클릭) q 로 바로 담게 되자 모드가 따로
// 할 일이 없었다.
func (s *sketch) pickHovered() {
	if s.hoverX < 0 {
		s.message = "스포이드: 판 위에 마우스를 두고 q 를 누른다"
		return
	}
	s.takeCell(s.hoverX, s.hoverY)
}

func (s *sketch) motion(mouse tea.Mouse) {
	if s.popup == popupGlyph {
		s.hoverGlyph(mouse)
	}
	x, y, ok := s.canvasAt(mouse)
	s.hoverX, s.hoverY = -1, -1
	if !ok {
		return
	}
	s.hoverX, s.hoverY = x, y
	if s.popup != popupNone {
		return
	}
	s.mode.move(s, mouse.Button, x, y)
}

// stroke 는 한 칸씩 모양에서 버튼이 눌린 채 지나간 칸에 손을 댄다. 우클릭은 어느 도구에서나 지우개이고,
// 좌클릭은 지금 도구를 쓴다(apply).
func (s *sketch) stroke(button tea.MouseButton, x, y int) {
	switch button {
	case tea.MouseRight:
		s.canvas.Erase(x, y)
	case tea.MouseLeft:
		s.mode.apply(s, s.canvas, x, y)
	default:
		return
	}
	s.dirty = true
}

func (s *sketch) wheel(mouse tea.Mouse) {
	if s.popup == popupGlyph {
		s.scrollGlyphs(mouse)
		s.hoverGlyph(mouse)
		return
	}
	switch mouse.Button {
	case tea.MouseWheelUp:
		s.scrollBy(0, -3)
	case tea.MouseWheelDown:
		s.scrollBy(0, 3)
	case tea.MouseWheelLeft:
		s.scrollBy(-3, 0)
	case tea.MouseWheelRight:
		s.scrollBy(3, 0)
	}
	// 판이 굴러 마우스 밑의 칸이 바뀌었다. 짚은 칸은 q(pickHovered)와 눈금이 본다.
	s.hoverX, s.hoverY = -1, -1
	if x, y, ok := s.canvasAt(mouse); ok {
		s.hoverX, s.hoverY = x, y
	}
}

// 색표: 제목 한 줄, 16x16 격자(한 색이 두 칸), "없음" 한 줄. 테두리 안쪽 기준이다.
const (
	swatchWidth   = 2
	colorColumns  = 16
	colorRows     = 16
	colorTitle    = "색표  좌클릭 전경 · 우클릭 배경 · Esc 닫기"
	noColorButton = "[없음]"
)

func colorBox() string {
	lines := []string{colorTitle}
	for row := range colorRows {
		var line strings.Builder
		for column := range colorColumns {
			swatch := lipgloss.NewStyle().Background(lipgloss.ANSIColor(row*colorColumns + column))
			line.WriteString(swatch.Render(strings.Repeat(" ", swatchWidth)))
		}
		lines = append(lines, line.String())
	}
	lines = append(lines, noColorButton)
	return popupStyle.Render(strings.Join(lines, "\n"))
}

// pickColor 는 색표의 누름이다. 칸을 누르면 그 색을 들고, 창 바깥을 누르면 창을 닫는다. 창은 여러
// 번 골라도 떠 있다(전경과 배경을 잇달아 고르므로).
func (s *sketch) pickColor(mouse tea.Mouse) {
	box := colorBox()
	if s.outsidePopup(box, mouse) {
		s.openPopup(popupNone)
		return
	}
	x, y := s.insidePopup(box, mouse)
	var color Color
	switch {
	case y >= 1 && y <= colorRows && x >= 0 && x < colorColumns*swatchWidth:
		color = Color((y-1)*colorColumns + x/swatchWidth)
	case y == colorRows+1 && x >= 0 && x < ansi.StringWidth(noColorButton):
		color = NoColor
	default:
		return
	}
	switch mouse.Button {
	case tea.MouseLeft:
		s.brush.Fg = color
	case tea.MouseRight:
		s.brush.Bg = color
	}
}

var popupStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())

// popupOrigin 은 창이 화면 가운데 놓일 때의 왼쪽 위다.
func (s *sketch) popupOrigin(box string) (x, y int) {
	return s.centerOrigin(lipgloss.Width(box), lipgloss.Height(box))
}

// centerOrigin 은 폭 width · 높이 height 인 창이 화면 가운데 놓일 때의 왼쪽 위다. 창을 그리지 않고
// 크기만 아는 곳(glyphAt)이 쓴다.
func (s *sketch) centerOrigin(width, height int) (x, y int) {
	return max((s.width-width)/2, 0), max((s.viewHeight()-height)/2, 0)
}

// outsidePopup 은 마우스가 창 바깥인지다. 테두리까지가 창이다. 색표와 글자표가 바깥 누름으로 닫힐 때 본다.
func (s *sketch) outsidePopup(box string, mouse tea.Mouse) bool {
	originX, originY := s.popupOrigin(box)
	return mouse.X < originX || mouse.Y < originY ||
		mouse.X >= originX+lipgloss.Width(box) || mouse.Y >= originY+lipgloss.Height(box)
}

// insidePopup 은 마우스 칸을 창 테두리 안쪽 기준으로 바꾼다.
func (s *sketch) insidePopup(box string, mouse tea.Mouse) (x, y int) {
	originX, originY := s.popupOrigin(box)
	return mouse.X - originX - 1, mouse.Y - originY - 1
}

func (s *sketch) View() tea.View {
	if s.width == 0 || s.height == 0 {
		return tea.NewView("")
	}
	// 끄는 중인 모양은 판의 사본에 그려 보인다. 원본에는 버튼을 뗄 때 적는다.
	canvas := s.canvas
	if s.dragging != nil {
		canvas = s.canvas.clone()
		s.drawDrag(canvas)
	}
	view := s.withToolbar(s.board(canvas))
	var box string
	switch s.popup {
	case popupColor:
		box = colorBox()
	case popupGlyph:
		box = s.glyphBox()
	case popupCommand:
		box = s.commandBox()
	}
	if box != "" {
		x, y := s.popupOrigin(box)
		view = lipgloss.NewCanvas(s.width, s.viewHeight()).
			Compose(lipgloss.NewCompositor(
				lipgloss.NewLayer(view),
				lipgloss.NewLayer(box).X(x).Y(y).Z(1),
			)).
			Render()
	}

	out := tea.NewView(view + "\n" + s.strip())
	out.AltScreen = true
	out.MouseMode = tea.MouseModeAllMotion
	// 창이 떠 있으면 커서를 숨긴다. 커서는 판 위의 칸에 두는 것이라 창 위에 뜨면 안 된다.
	if s.popup == popupNone {
		out.Cursor = s.mode.cursor(s)
	}
	return out
}

var imeWarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Red)

// stripHints 는 띠 오른쪽 끝의 키 안내다. 늘 같은 글자라 오른쪽에 붙이면 늘 같은 곳에 있다.
const stripHints = "wasd 굴림 Tab 모양 ^P 명령 ^S 저장 ^C 끝"

// stripPart 는 띠 왼쪽의 한 토막이다. opens 가 popupNone 이 아니면 그 토막을 누를 때 그 창이 뜬다.
type stripPart struct {
	text  string
	opens popup
}

// stripGap 은 띠의 토막 사이 빈칸이다. 그리는 곳과 누름을 재는 곳이 같은 값을 봐야 한다.
const stripGap = "  "

// stripParts 는 띠 왼쪽의 토막들이다. 붓과 색, 모드, 짚은 칸, 알림.
func (s *sketch) stripParts() []stripPart {
	parts := []stripPart{}
	// 입력기 안내는 맨 앞이다. 띠가 좁으면 뒤가 잘리는데 이것이 잘리면 안 된다.
	if s.imeOn {
		parts = append(parts, stripPart{text: imeWarningStyle.Render("입력기가 켜져 있다: 한/영 키로 끄면 도구 키가 먹는다")})
	}
	parts = append(parts,
		// 붓은 눌러서 글자표를, 색 둘은 색표를 연다. 붓을 바꾸려고 도구 줄까지 손을 옮기지 않아도 된다.
		stripPart{text: "붓 " + brushLabel(s.brush), opens: popupGlyph},
		stripPart{text: "전경 " + colorLabel(s.brush.Fg), opens: popupColor},
		stripPart{text: "배경 " + colorLabel(s.brush.Bg), opens: popupColor},
		stripPart{text: s.mode.label(s)},
	)
	if s.hoverX >= 0 {
		parts = append(parts, stripPart{text: fmt.Sprintf("(%d,%d)", s.hoverX, s.hoverY)})
	}
	if s.dirty {
		parts = append(parts, stripPart{text: "*수정됨"})
	}
	if s.message != "" {
		parts = append(parts, stripPart{text: s.message})
	}
	return parts
}

// strip 은 맨 아래 띠다. 왼쪽에 토막들(stripParts)을 적고 오른쪽 끝에 키 안내를 붙인다.
//
// 왼쪽은 마우스가 움직이고 알림이 뜰 때마다 길이가 바뀐다. 안내를 그 뒤에 이어 붙이면 안내가
// 따라 흔들려서 오른쪽에 떼어 두었다. 둘이 모자라면 왼쪽이 잘린다. 안내는 창이 안내보다 좁을 때만
// 잘린다.
func (s *sketch) strip() string {
	texts := []string{}
	for _, part := range s.stripParts() {
		texts = append(texts, part.text)
	}
	hints := ansi.Truncate(stripHints, s.width, "…")
	room := s.width - ansi.StringWidth(hints) - 2 // 둘 사이에 적어도 두 칸
	if room < 1 {
		return hints
	}
	status := ansi.Truncate(strings.Join(texts, stripGap), room, "…")
	return status + strings.Repeat(" ", s.width-ansi.StringWidth(status)-ansi.StringWidth(hints)) + hints
}

// clickStrip 은 띠의 누름이다. 누른 곳의 토막이 여는 창이 있으면 연다.
func (s *sketch) clickStrip(mouse tea.Mouse) tea.Cmd {
	if mouse.Button != tea.MouseLeft {
		return nil
	}
	x := 0
	for _, part := range s.stripParts() {
		width := ansi.StringWidth(part.text)
		if mouse.X >= x && mouse.X < x+width {
			if part.opens == popupNone {
				return nil
			}
			return s.openPopup(part.opens)
		}
		x += width + ansi.StringWidth(stripGap)
	}
	return nil
}

// brushLabel 은 띠에 보이는 붓이다. 글자 뒤에 코드 포인트를 붙인다(U+25B2). 비슷하게 생긴 글자(─ ━,
// ⋅ ·)를 눈으로 가릴 수 없고, 목업을 보고 코드로 옮길 때 그 값이 든다. 여러 rune 으로 된 글자는
// rune 마다 적는다. 빈칸 붓은 그대로 그리면 안 보이므로 이름도 적는다. 폭이 애매한 글자는 그 까닭을
// 뒤에 붙인다(widthDoubt). 글자표는 바탕으로만 보여서 무엇이 애매한지는 여기서 안다.
func brushLabel(brush Cell) string {
	points := []string{}
	for _, r := range brush.Glyph {
		points = append(points, fmt.Sprintf("U+%04X", r))
	}
	label := brush.style().Render(brush.Glyph)
	if name, ok := blankNames[brush.Glyph]; ok {
		label += name
	}
	label += " " + strings.Join(points, " ")
	if doubt := widthDoubt(brush.Glyph); doubt != "" {
		label += " 폭 애매(" + doubt + ")"
	}
	return label
}

func colorLabel(color Color) string {
	if color == NoColor {
		return "없음"
	}
	return lipgloss.NewStyle().Background(lipgloss.ANSIColor(color)).Render("  ") + fmt.Sprint(int(color))
}
