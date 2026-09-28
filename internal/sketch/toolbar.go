// 왼쪽 도구 줄이다. 그림판처럼 도구와 붓 · 색을 한 줄씩 세워 두고 클릭으로 부른다.
//
// 모드가 넷을 넘고 스포이드까지 생기자 Tab 순환과 띠 안내만으로는 무엇이 있는지 안 보였다.
// 스포이드를 조합키(ctrl · alt + 클릭)로 두지 않은 것은 macOS 터미널이 ctrl+클릭을 우클릭으로 바꿔
// 보내기도 해서다. 그러면 스포이드가 지우개가 된다. 버튼은 터미널마다 다르지 않다. (ADR-0001 §4)
//
// 단축키와 팔레트는 그대로 있다. 도구 줄은 같은 일을 하는 또 하나의 길이다.

package sketch

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// toolbarWidth 는 도구 줄의 폭이다. 가장 긴 줄(" 전경 없음 c ", 13칸)에 맞춘다.
const toolbarWidth = 13

var (
	toolbarStyle = lipgloss.NewStyle().Background(lipgloss.ANSIColor(236))
	// toolActiveStyle 은 지금 든 도구다.
	toolActiveStyle = lipgloss.NewStyle().Reverse(true)
)

// tool 은 도구 줄의 한 줄이다. 빈 줄(구분)은 label 이 nil 이다.
//
// label 이 함수인 것은 붓 · 전경 · 배경 줄이 지금 값을 보여야 해서다. active 는 모드 도구만 쓴다.
// key 는 오른쪽 끝에 적는 단축키다. 누르는 것은 paintKey 가 받는다. 여기는 보이기만 한다.
type tool struct {
	label  func(s *sketch) string
	key    string
	active func(s *sketch) bool
	run    func(s *sketch) tea.Cmd
}

func modeTool(m mode, key string) tool {
	return tool{
		label:  func(*sketch) string { return m.String() },
		key:    key,
		active: func(s *sketch) bool { return s.mode == m },
		run: func(s *sketch) tea.Cmd {
			s.setMode(m)
			return nil
		},
	}
}

// figureTool 은 모양 하나를 고르는 줄이다. 키 대신 Tab 이 돌므로 키 칸은 비운다.
func figureTool(f figure) tool {
	return tool{
		label:  func(*sketch) string { return f.String() },
		active: func(s *sketch) bool { return s.figure == f },
		run: func(s *sketch) tea.Cmd {
			s.setFigure(f)
			return nil
		},
	}
}

func fixedLabel(text string) func(*sketch) string { return func(*sketch) string { return text } }

// tools 는 도구 줄이다. 목록 차례가 곧 위에서부터의 줄이다.
var tools = []tool{
	modeTool(modePaint, "b"),
	modeTool(modeText, "t"),
	modeTool(modeRecolor, "f"),
	modeTool(modeErase, "e"),
	{},
	// 모양은 Tab 으로 돈다. 눌러서 고를 수도 있다. 칠하기 · 색칠 · 지우기가 함께 쓴다.
	figureTool(figureDot),
	figureTool(figureLine),
	figureTool(figureBox),
	figureTool(figureFill),
	{},
	{label: func(s *sketch) string { return "붓 " + s.brush.style().Render(s.brush.Glyph) }, key: "v", run: func(s *sketch) tea.Cmd { return s.openPopup(popupGlyph) }},
	{label: func(s *sketch) string { return "전경 " + swatch(s.brush.Fg) }, key: "c", run: func(s *sketch) tea.Cmd { return s.openPopup(popupColor) }},
	{label: func(s *sketch) string { return "배경 " + swatch(s.brush.Bg) }, key: "c", run: func(s *sketch) tea.Cmd { return s.openPopup(popupColor) }},
	// 스포이드는 모드가 아니라 키다(pickHovered). 판 위의 칸을 짚고 눌러야 하므로 버튼으로는 할 일이
	// 없어, 붓 · 색 아래에 그 키를 적어 두기만 한다. 담기는 것이 바로 위의 셋이다.
	{label: fixedLabel("스포이드"), key: "q"},
	{},
	{label: fixedLabel("명령"), key: "^P", run: func(s *sketch) tea.Cmd { return s.openPopup(popupCommand) }},
	{label: fixedLabel("저장"), key: "^S", run: func(s *sketch) tea.Cmd {
		s.save()
		return nil
	}},
	{label: fixedLabel("되돌리기"), key: "^Z", run: func(s *sketch) tea.Cmd {
		s.undo()
		return nil
	}},
	{label: fixedLabel("다시"), key: "^Y", run: func(s *sketch) tea.Cmd {
		s.redo()
		return nil
	}},
}

// swatch 는 색 한 칸을 두 칸 폭으로 보인다. 색 없음은 이름으로 적는다.
func swatch(color Color) string {
	if color == NoColor {
		return "없음"
	}
	return lipgloss.NewStyle().Background(lipgloss.ANSIColor(color)).Render("  ")
}

// toolbarLine 은 도구 줄의 row 번째 줄이다. 폭이 늘 toolbarWidth 다.
func (s *sketch) toolbarLine(row int) string {
	if row >= len(tools) || tools[row].label == nil {
		return toolbarStyle.Render(strings.Repeat(" ", toolbarWidth))
	}
	item := tools[row]
	key := item.key + " "
	text := ansi.Truncate(" "+item.label(s), toolbarWidth-ansi.StringWidth(key), "")
	text += strings.Repeat(" ", toolbarWidth-ansi.StringWidth(text)-ansi.StringWidth(key)) + key
	if item.active != nil && item.active(s) {
		return toolActiveStyle.Render(text)
	}
	return toolbarStyle.Render(text)
}

// withToolbar 는 판 줄마다 왼쪽에 도구 줄을 붙인다.
func (s *sketch) withToolbar(board string) string {
	lines := strings.Split(board, "\n")
	for row := range lines {
		lines[row] = s.toolbarLine(row) + lines[row]
	}
	return strings.Join(lines, "\n")
}

func (s *sketch) clickToolbar(mouse tea.Mouse) tea.Cmd {
	if mouse.Button != tea.MouseLeft || mouse.Y >= len(tools) || tools[mouse.Y].run == nil {
		return nil
	}
	return tools[mouse.Y].run(s)
}
