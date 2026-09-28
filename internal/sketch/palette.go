// 명령 팔레트다(ctrl+p). 검색어 한 줄과 그것이
// 걸러 낸 명령 목록이다. 색표 · 글자표 · 모드 · 저장을 단축키를 몰라도 부를 수 있게 둔다. 단축키는 그대로 있다.
//
// 마우스로도 고른다. 그림판은 손이 마우스에 있는 도구라서다.

package sketch

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bluemir/paint/internal/tui/components"
)

// command 는 팔레트에 오르는 명령 하나다. run 은 판과 팔레트 밑의 모드 화면(from)을 받아 다음 화면을
// 돌려준다. 대개 from 이라 팔레트가 닫히고, 도구를 바꾸면 새 모드 화면이, 창을 여는 명령은 그 창이다.
type command struct {
	name string // 검색어가 맞춰 보는 이름
	desc string // 목록 오른쪽에 붙는 설명. 검색어가 이것도 훑는다
	run  func(s *sketch, from tea.Model) (tea.Model, tea.Cmd)
}

// commands 는 팔레트의 명령표다. 목록 차례가 곧 화면에 뜨는 차례다. 도구와 모양은 제 목록(modes,
// figures)에서 가져온다.
var commands = slices.Concat(
	[]command{
		{name: "color", desc: "색표를 연다", run: (*sketch).openColors},
		{name: "glyph", desc: "글자표를 연다", run: (*sketch).openGlyphs},
	},
	modeCommands(),
	figureCommands(),
	editCommands,
)

func modeCommands() []command {
	out := []command{}
	for _, m := range modes {
		out = append(out, command{name: m.command, desc: m.desc, run: m.open})
	}
	return out
}

func figureCommands() []command {
	out := []command{}
	for _, f := range figures {
		out = append(out, command{name: f.command(), desc: "모양: " + f.String(), run: func(s *sketch, from tea.Model) (tea.Model, tea.Cmd) {
			s.figure = f
			return from, nil
		}})
	}
	return out
}

var editCommands = []command{
	{name: "undo", desc: "되돌린다", run: func(s *sketch, from tea.Model) (tea.Model, tea.Cmd) {
		s.undo()
		return from, nil
	}},
	{name: "redo", desc: "다시 한다", run: func(s *sketch, from tea.Model) (tea.Model, tea.Cmd) {
		s.redo()
		return from, nil
	}},
	{name: "save", desc: "파일에 저장한다", run: func(s *sketch, from tea.Model) (tea.Model, tea.Cmd) {
		s.save()
		return from, nil
	}},
}

// viewCommand 는 팔레트가 떠 있는 화면이다. 떠 있는 동안 키는 전부 검색어로 간다. Tab 이나 글자 키가
// 판에 닿으면 안 된다. 검색어와 짚은 줄은 이 화면이 갖는다. 열 때마다 새로 세우므로 비어 있다.
type viewCommand struct {
	*sketch
	// under 는 팔레트 밑의 모드 화면이다. 닫으면 그리로 돌아가고, 명령은 그것을 from 으로 받는다.
	under tea.Model

	query  components.Text
	cursor int
}

func (v *viewCommand) Init() tea.Cmd { return nil }

func (v *viewCommand) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg, tea.MouseReleaseMsg:
		v.under.Update(msg)
	case tea.KeyPressMsg:
		return v.key(msg)
	case tea.MouseClickMsg:
		return v.click(tea.Mouse(msg))
	case tea.MouseMotionMsg, tea.MouseWheelMsg:
		// 팔레트는 마우스 움직임을 쓰지 않는다. 판에도 안 넘긴다.
	default:
		// 캐럿 깜빡임이다. 검색어가 안 받으면 캐럿이 멈춘다.
		return v, v.query.Update(msg)
	}
	return v, nil
}

func (v *viewCommand) View() tea.View { return v.overlay(v.under, v.box()) }

// matches 는 검색어가 걸러 남긴 명령들이다. 검색어가 비었으면 전부다. 이름이 영어라 설명문도
// 훑어야 한글로 찾힌다.
func (v *viewCommand) matches() []command {
	want := strings.ToLower(strings.TrimSpace(v.query.Value()))
	if want == "" {
		return commands
	}
	found := []command{}
	for _, cmd := range commands {
		if strings.Contains(strings.ToLower(cmd.name+" "+cmd.desc), want) {
			found = append(found, cmd)
		}
	}
	return found
}

func (v *viewCommand) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// ctrl+p 는 팔레트를 새로 연다. 판이 받으므로 검색어가 비워진다.
	if slices.Contains(sketchKeys, msg.String()) {
		return v.keyOver(v, v.under, msg)
	}
	found := v.matches()
	switch msg.String() {
	case "esc":
		return v.under, nil
	case "enter":
		if v.cursor < len(found) {
			return found[v.cursor].run(v.sketch, v.under)
		}
		return v, nil
	case "up", "down":
		if len(found) == 0 {
			return v, nil
		}
		by := 1
		if msg.String() == "up" {
			by = -1
		}
		v.cursor = ((v.cursor+by)%len(found) + len(found)) % len(found)
		return v, nil
	}
	cmd := v.query.Update(msg)
	// 목록이 바뀌었으니 커서를 맨 위로 되돌린다. 안 그러면 짚은 줄이 목록 밖이 될 수 있다.
	v.cursor = 0
	return v, cmd
}

// click 은 목록의 줄을 누르면 그 명령을 돌린다. 첫 줄은 검색어라 목록은 둘째 줄부터다.
func (v *viewCommand) click(mouse tea.Mouse) (tea.Model, tea.Cmd) {
	if mouse.Button != tea.MouseLeft {
		return v, nil
	}
	_, y := v.insidePopup(v.box(), mouse)
	found := v.matches()
	if index := y - 1; index >= 0 && index < len(found) {
		return found[index].run(v.sketch, v.under)
	}
	return v, nil
}

// box 는 팔레트 상자다. 명령이 몇 개 없어 화면보다 길어질 일이 없으므로 굴리지 않는다.
func (v *viewCommand) box() string {
	const marker = "> "
	const nothing = "  맞는 명령이 없다"
	found := v.matches()

	// 폭은 걸러지기 전 목록 전체로 잰다. 치는 동안 상자가 좁아졌다 넓어지면 눈이 따라가야 한다.
	labels, tails := lipgloss.Width(nothing), 0
	for _, cmd := range commands {
		labels = max(labels, lipgloss.Width(marker+cmd.name))
		tails = max(tails, lipgloss.Width(cmd.desc))
	}
	inner := max(labels+2+tails, lipgloss.Width(v.query.String()))

	lines := []string{v.query.String()}
	if len(found) == 0 {
		lines = append(lines, dimStyle.Render(nothing))
	}
	for i, cmd := range found {
		prefix := "  "
		if i == v.cursor {
			prefix = marker
		}
		left := prefix + cmd.name
		pad := max(inner-lipgloss.Width(left)-lipgloss.Width(cmd.desc), 0)
		lines = append(lines, left+strings.Repeat(" ", pad)+dimStyle.Render(cmd.desc))
	}
	return popupStyle.Width(inner + 2).Render(strings.Join(lines, "\n"))
}

var dimStyle = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
