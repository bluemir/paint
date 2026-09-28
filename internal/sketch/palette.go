// 명령 팔레트다(ctrl+p). 검색어 한 줄과 그것이
// 걸러 낸 명령 목록이다. 색표 · 글자표 · 모드 · 저장을 단축키를 몰라도 부를 수 있게 둔다. 단축키는 그대로 있다.
//
// 마우스로도 고른다. 그림판은 손이 마우스에 있는 도구라서다.

package sketch

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// command 는 팔레트에 오르는 명령 하나다.
type command struct {
	name string // 검색어가 맞춰 보는 이름
	desc string // 목록 오른쪽에 붙는 설명. 검색어가 이것도 훑는다
	run  func(s *sketch) tea.Cmd
}

// commands 는 팔레트의 명령표다. 목록 차례가 곧 화면에 뜨는 차례다.
var commands = []command{
	{name: "color", desc: "색표를 연다", run: func(s *sketch) tea.Cmd { return s.openPopup(popupColor) }},
	{name: "glyph", desc: "글자표를 연다", run: func(s *sketch) tea.Cmd { return s.openPopup(popupGlyph) }},
	{name: "paint", desc: "칠하기 모드로", run: func(s *sketch) tea.Cmd { return s.switchMode(modePaint) }},
	{name: "text", desc: "글자 모드로", run: func(s *sketch) tea.Cmd { return s.switchMode(modeText) }},
	{name: "recolor", desc: "색칠 모드로 (글자는 두고 색만)", run: func(s *sketch) tea.Cmd { return s.switchMode(modeRecolor) }},
	{name: "erase", desc: "지우기 모드로", run: func(s *sketch) tea.Cmd { return s.switchMode(modeErase) }},
	{name: "dot", desc: "모양: 한 칸씩", run: func(s *sketch) tea.Cmd { return s.switchFigure(figureDot) }},
	{name: "line", desc: "모양: 직선", run: func(s *sketch) tea.Cmd { return s.switchFigure(figureLine) }},
	{name: "box", desc: "모양: 테두리", run: func(s *sketch) tea.Cmd { return s.switchFigure(figureBox) }},
	{name: "fill", desc: "모양: 채움", run: func(s *sketch) tea.Cmd { return s.switchFigure(figureFill) }},
	{name: "undo", desc: "되돌린다", run: func(s *sketch) tea.Cmd {
		s.undo()
		return s.openPopup(popupNone)
	}},
	{name: "redo", desc: "다시 한다", run: func(s *sketch) tea.Cmd {
		s.redo()
		return s.openPopup(popupNone)
	}},
	{name: "save", desc: "파일에 저장한다", run: func(s *sketch) tea.Cmd {
		s.save()
		return s.openPopup(popupNone)
	}},
}

// switchMode 는 모드를 바꾸고 팔레트를 닫는다.
func (s *sketch) switchMode(next mode) tea.Cmd {
	s.setMode(next)
	return s.openPopup(popupNone)
}

// switchFigure 는 모양을 바꾸고 팔레트를 닫는다.
func (s *sketch) switchFigure(next figure) tea.Cmd {
	s.setFigure(next)
	return s.openPopup(popupNone)
}

// matches 는 검색어가 걸러 남긴 명령들이다. 검색어가 비었으면 전부다. 이름이 영어라 설명문도
// 훑어야 한글로 찾힌다.
func (s *sketch) matches() []command {
	want := strings.ToLower(strings.TrimSpace(s.query.Value()))
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

func (s *sketch) commandKey(msg tea.KeyPressMsg) tea.Cmd {
	found := s.matches()
	switch msg.String() {
	case "enter":
		if s.commandCursor < len(found) {
			return found[s.commandCursor].run(s)
		}
		return nil
	case "up", "down":
		if len(found) == 0 {
			return nil
		}
		by := 1
		if msg.String() == "up" {
			by = -1
		}
		s.commandCursor = ((s.commandCursor+by)%len(found) + len(found)) % len(found)
		return nil
	}
	cmd := s.query.Update(msg)
	// 목록이 바뀌었으니 커서를 맨 위로 되돌린다. 안 그러면 짚은 줄이 목록 밖이 될 수 있다.
	s.commandCursor = 0
	return cmd
}

// clickCommand 는 목록의 줄을 누르면 그 명령을 돌린다. 첫 줄은 검색어라 목록은 둘째 줄부터다.
func (s *sketch) clickCommand(mouse tea.Mouse) {
	if mouse.Button != tea.MouseLeft {
		return
	}
	_, y := s.insidePopup(s.commandBox(), mouse)
	found := s.matches()
	if index := y - 1; index >= 0 && index < len(found) {
		found[index].run(s)
	}
}

// commandBox 는 팔레트 상자다. 명령이 몇 개 없어 화면보다 길어질 일이 없으므로 굴리지 않는다.
func (s *sketch) commandBox() string {
	const marker = "> "
	const nothing = "  맞는 명령이 없다"
	found := s.matches()

	// 폭은 걸러지기 전 목록 전체로 잰다. 치는 동안 상자가 좁아졌다 넓어지면 눈이 따라가야 한다.
	labels, tails := lipgloss.Width(nothing), 0
	for _, cmd := range commands {
		labels = max(labels, lipgloss.Width(marker+cmd.name))
		tails = max(tails, lipgloss.Width(cmd.desc))
	}
	inner := max(labels+2+tails, lipgloss.Width(s.query.String()))

	lines := []string{s.query.String()}
	if len(found) == 0 {
		lines = append(lines, dimStyle.Render(nothing))
	}
	for i, cmd := range found {
		prefix := "  "
		if i == s.commandCursor {
			prefix = marker
		}
		left := prefix + cmd.name
		pad := max(inner-lipgloss.Width(left)-lipgloss.Width(cmd.desc), 0)
		lines = append(lines, left+strings.Repeat(" ", pad)+dimStyle.Render(cmd.desc))
	}
	return popupStyle.Width(inner + 2).Render(strings.Join(lines, "\n"))
}

var dimStyle = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
