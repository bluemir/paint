// sketch 는 터미널 그림판이다. 칸 단위로 글자와 256색을 칠한다.
//
// 텍스트 목업으로는 터미널 색을 입힐 수 없어서 있다. 마우스로 글자와 256색을 칠해 파일로 남기고,
// 그 파일을 다시 열어 고친다. (ADR-0001)
//
// 도구(mode.go)는 넷이고 키로 고르며, Tab 은 모양(shape.go)을 돈다. 브러시에서는 좌클릭이 붓을 찍고
// 우클릭이 지운다. 글자에서는 클릭으로 커서를 놓고 치는 대로 적힌다. 칠하기에서는 글자를 두고 색만 붓
// 색으로 바꾼다. 지우기에서는 좌클릭도 지운다. 붓 글자는 글자표(v)에서 고른다. 색표(c)와 글자표는 판 위에
// 뜨는 창이고, 명령 팔레트(ctrl+p)와 도구 줄로도 열린다. 도구 키는 toolKey 에 있다.
//
// sketch 는 판과 붓 같은 공유 상태이고 화면이 아니다. 화면은 모드마다 하나(viewBrush · viewText ·
// viewPaint · viewErase)와 창마다 하나(viewColor · viewGlyph · viewCommand)이고, 모두 sketch 를 품는다. 지금 어느 모드인지, 무엇이 떠
// 있는지는 필드가 아니라 떠 있는 화면이다. bubbletea 는 Update 가 돌려준 모델로 화면을 바꾼다.
// (ADR-0005, ADR-0006)
//
// 여기 있는 것은 화면들이 함께 쓰는 판의 일이다: 크기, 좌표, 굴리기, 도구 키, ctrl 키, 그리기.

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
	_, err := tea.NewProgram(&viewBrush{sketch: newSketch(path, canvas)}, tea.WithContext(ctx)).Run()
	return errors.WithStack(err)
}

type sketch struct {
	path   string
	canvas *Canvas

	width, height int // 터미널 크기. 맨 아래 한 줄은 띠다.
	left, top     int // 판의 어디부터 보이는지

	// figure 는 지금 모양이다. 도구를 바꿔도 남는다. 그리는 도구가 모두 쓴다(shape.go).
	figure figure
	// dragging 은 직선 · 테두리 · 채움에서 끌고 있는 모양이다. 끌고 있지 않으면 nil 이다. 도구나 모양을
	// 바꾸면 버린다. 바뀐 뒤 버튼을 떼면 무엇을 그릴지 모른다.
	dragging *drag
	history  history
	brush    Cell

	hoverX, hoverY int // 마우스가 짚은 판 위의 칸. 판 밖이면 -1.

	dirty bool
	// imeOn 은 도구 키가 입력기를 거쳐 조합 글자로 왔는지다. 띠에 안내를 띄운다. 라틴 글자가 그대로
	// 오면 끈 것이다. 판정은 ime.go 에 있다.
	imeOn   bool
	message string
}

func newSketch(path string, canvas *Canvas) *sketch {
	return &sketch{
		path:   path,
		canvas: canvas,
		figure: figureDot,
		brush:  Cell{Glyph: "#", Fg: NoColor, Bg: NoColor},
		hoverX: -1, hoverY: -1,
	}
}

// viewHeight 는 띠를 뺀 화면 줄 수다. 창(색표 등)은 이 안 가운데에 뜬다.
func (s *sketch) viewHeight() int { return max(s.height-1, 0) }

// 판은 화면 (originX, 1) 에서 시작한다. 왼쪽 끝이 도구 줄이고(toolbar.go), 그 오른쪽 gutter 칸이 줄
// 번호, 위 한 줄이 열 눈금이다(ruler.go). 화면 칸과 판 칸을 바꾸는 곳은 전부 이것들을 본다.
func (s *sketch) gutter() int { return len(strconv.Itoa(s.canvas.Height-1)) + 1 }

func (s *sketch) originX() int { return toolbarWidth + s.gutter() }

func (s *sketch) canvasViewWidth() int { return max(s.width-s.originX(), 0) }

func (s *sketch) canvasViewHeight() int { return max(s.viewHeight()-1, 0) }

func (s *sketch) resize(msg tea.WindowSizeMsg) {
	s.width, s.height = msg.Width, msg.Height
	s.scrollBy(0, 0)
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

// openColors · openGlyphs · openCommands 는 판 위에 창을 띄운다. from 은 창을 연 모드 화면이고 창을
// 닫으면 그리로 돌아간다. 돌려주는 모델이 곧 다음 화면이다.
func (s *sketch) openColors(from tea.Model) (tea.Model, tea.Cmd) {
	return &viewColor{sketch: s, under: from}, nil
}

func (s *sketch) openGlyphs(from tea.Model) (tea.Model, tea.Cmd) { return newViewGlyph(s, from), nil }

func (s *sketch) openCommands(from tea.Model) (tea.Model, tea.Cmd) {
	palette := &viewCommand{sketch: s, under: from, query: components.NewText("")}
	return palette, palette.query.Focus()
}

// sketchKeys 는 어느 화면에서나 판이 받는 키다. 저장 · 되돌리기 · 끝내기는 창을 닫지 않고도 된다.
var sketchKeys = []string{"ctrl+c", "ctrl+s", "ctrl+z", "ctrl+y", "ctrl+p"}

// sketchKey 는 판이 받는 키(sketchKeys)다. from 은 지금 모드 화면이다. 대개 from 을 그대로 돌려주고,
// ctrl+p 는 from 위에 팔레트를, 저장 안 한 것이 있을 때의 ctrl+c 는 끝내기 확인 창을 띄운다.
func (s *sketch) sketchKey(from tea.Model, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		// 저장 안 한 것이 있으면 묻는다(quit.go).
		if s.dirty {
			return &viewQuit{sketch: s, under: from}, nil
		}
		return from, tea.Quit
	case "ctrl+s":
		s.save()
	case "ctrl+z":
		s.undo()
	case "ctrl+y":
		s.redo()
	case "ctrl+p":
		return s.openCommands(from)
	}
	return from, nil
}

// keyOver 는 창 over 가 떠 있을 때 받은 판의 키(sketchKeys)를 판에 넘긴다. under 는 창 밑의 모드
// 화면이다. 판이 모드 화면을 그대로 두면(저장 · 되돌리기) 창도 그대로 떠 있고, 다른 화면을 세우면
// (ctrl+p 의 팔레트) 그리로 간다.
func (s *sketch) keyOver(over, under tea.Model, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, cmd := s.sketchKey(under, msg)
	if next == under {
		return over, cmd
	}
	return next, cmd
}

// nextFigure 는 모양을 다음 것으로 돌린다(Tab). 끌던 것은 버린다.
func (s *sketch) nextFigure() {
	s.dragging = nil
	s.figure = figures[(slices.Index(figures, s.figure)+1)%len(figures)]
}

// toolKey 는 글자 모드가 아닐 때의 키다. 한 글자 키가 도구를 들고(modes), wasd 와 방향키가 판을
// 굴린다. 도구 줄의 오른쪽에 같은 키가 적혀 있다(toolbar.go). from 은 지금 모드 화면이다.
//
// 글자 키는 붓 글자를 안 바꾼다. 바꾸던 때는 잘못 누른 키나 켜 둔 입력기 때문에 모르는 새 붓이 바뀌어
// 있었다. 붓 글자는 글자표를 연 동안에만 키로 고른다(viewGlyph.key).
func (s *sketch) toolKey(from tea.Model, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// 글자 모드에서 한글을 치다 Esc 로 나오면 입력기가 켜진 채라 b 가 ㅠ 로 온다. 키가 안 먹는
	// 까닭이 안 보이므로 띠에 적는다. 글자 모드와 글자표는 한글이 곧 뜻이라 여기서만 잰다.
	switch {
	case imeText(msg):
		s.imeOn = true
	case asciiLetter(msg):
		s.imeOn = false
	}
	for _, m := range modes {
		if msg.String() == m.key {
			return m.open(s, from)
		}
	}
	switch msg.String() {
	case "q":
		s.pickHovered()
	case "c":
		return s.openColors(from)
	case "v":
		return s.openGlyphs(from)
	case "w", "up":
		s.scrollBy(0, -1)
	case "s", "down":
		s.scrollBy(0, 1)
	case "a", "left":
		s.scrollBy(-1, 0)
	case "d", "right":
		s.scrollBy(1, 0)
	}
	return from, nil
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

// onChrome 은 누름이 판이 아니라 띠나 도구 줄 위인지다.
func (s *sketch) onChrome(mouse tea.Mouse) bool {
	return mouse.Y == s.height-1 || mouse.X < toolbarWidth
}

// clickChrome 은 띠와 도구 줄의 누름이다. from 은 지금 모드 화면, label 은 그 화면이 띠에 적은 도구
// 토막이다. 띠의 토막 자리를 그린 것과 같게 재야 한다.
func (s *sketch) clickChrome(from tea.Model, label string, mouse tea.Mouse) (tea.Model, tea.Cmd) {
	if mouse.Y == s.height-1 {
		return s.clickStrip(from, label, mouse)
	}
	return s.clickToolbar(from, mouse)
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

// hover 는 마우스가 짚은 판 칸을 적어 두고 돌려준다. 판 밖이면 ok 가 false 다. 짚은 칸은 q
// (pickHovered)와 눈금, 띠가 본다.
func (s *sketch) hover(mouse tea.Mouse) (x, y int, ok bool) {
	x, y, ok = s.canvasAt(mouse)
	s.hoverX, s.hoverY = -1, -1
	if ok {
		s.hoverX, s.hoverY = x, y
	}
	return x, y, ok
}

func (s *sketch) wheel(mouse tea.Mouse) {
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
	// 판이 굴러 마우스 밑의 칸이 바뀌었다.
	s.hover(mouse)
}

var popupStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())

// popupOrigin 은 창이 화면 가운데 놓일 때의 왼쪽 위다.
func (s *sketch) popupOrigin(box string) (x, y int) {
	return s.centerOrigin(lipgloss.Width(box), lipgloss.Height(box))
}

// centerOrigin 은 폭 width · 높이 height 인 창이 화면 가운데 놓일 때의 왼쪽 위다. 창을 그리지 않고
// 크기만 아는 곳(viewGlyph.at)이 쓴다.
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

// render 는 모드 화면이 그리는 판 화면이다. canvas 는 그릴 판(끄는 중이면 미리 보기 사본), tool 은
// 도구 줄에서 뒤집어 보일 도구 이름, label 은 띠의 도구 토막, cursor 는 터미널 커서(없으면 nil)다.
func (s *sketch) render(canvas *Canvas, tool, label string, cursor *tea.Cursor) tea.View {
	if s.width == 0 || s.height == 0 {
		return tea.NewView("")
	}
	out := tea.NewView(s.withToolbar(s.board(canvas), tool) + "\n" + s.strip(label))
	out.AltScreen = true
	out.MouseMode = tea.MouseModeAllMotion
	out.Cursor = cursor
	return out
}

// overlay 는 모드 화면 under 위에 창 box 를 얹는다. 창들이 제 상자를 그려 이것으로 얹는다(viewColor.View
// 등). 커서는 판 위의 칸에 두는 것이라 창이 떠 있으면 숨긴다.
func (s *sketch) overlay(under tea.Model, box string) tea.View {
	out := under.View()
	if s.width == 0 || s.height == 0 {
		return out
	}
	x, y := s.popupOrigin(box)
	out.SetContent(lipgloss.NewCanvas(s.width, s.height).
		Compose(lipgloss.NewCompositor(
			lipgloss.NewLayer(out.Content),
			lipgloss.NewLayer(box).X(x).Y(y).Z(1),
		)).
		Render())
	out.Cursor = nil
	return out
}

var imeWarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Red)

// stripHints 는 띠 오른쪽 끝의 키 안내다. 늘 같은 글자라 오른쪽에 붙이면 늘 같은 곳에 있다.
const stripHints = "wasd 굴림 Tab 모양 ^P 명령 ^S 저장 ^C 끝"

// stripPart 는 띠 왼쪽의 한 토막이다. opens 가 있으면 그 토막을 누를 때 그 창이 뜬다.
type stripPart struct {
	text  string
	opens func(s *sketch, from tea.Model) (tea.Model, tea.Cmd)
}

// stripGap 은 띠의 토막 사이 빈칸이다. 그리는 곳과 누름을 재는 곳이 같은 값을 봐야 한다.
const stripGap = "  "

// stripParts 는 띠 왼쪽의 토막들이다. 붓과 색, 도구(label), 짚은 칸, 알림.
func (s *sketch) stripParts(label string) []stripPart {
	parts := []stripPart{}
	// 입력기 안내는 맨 앞이다. 띠가 좁으면 뒤가 잘리는데 이것이 잘리면 안 된다.
	if s.imeOn {
		parts = append(parts, stripPart{text: imeWarningStyle.Render("입력기가 켜져 있다: 한/영 키로 끄면 도구 키가 먹는다")})
	}
	parts = append(parts,
		// 붓은 눌러서 글자표를, 색 둘은 색표를 연다. 붓을 바꾸려고 도구 줄까지 손을 옮기지 않아도 된다.
		stripPart{text: "붓 " + brushLabel(s.brush), opens: (*sketch).openGlyphs},
		stripPart{text: "전경 " + colorLabel(s.brush.Fg), opens: (*sketch).openColors},
		stripPart{text: "배경 " + colorLabel(s.brush.Bg), opens: (*sketch).openColors},
		stripPart{text: label},
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
func (s *sketch) strip(label string) string {
	texts := []string{}
	for _, part := range s.stripParts(label) {
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
func (s *sketch) clickStrip(from tea.Model, label string, mouse tea.Mouse) (tea.Model, tea.Cmd) {
	if mouse.Button != tea.MouseLeft {
		return from, nil
	}
	x := 0
	for _, part := range s.stripParts(label) {
		width := ansi.StringWidth(part.text)
		if mouse.X >= x && mouse.X < x+width {
			if part.opens == nil {
				return from, nil
			}
			return part.opens(s, from)
		}
		x += width + ansi.StringWidth(stripGap)
	}
	return from, nil
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
