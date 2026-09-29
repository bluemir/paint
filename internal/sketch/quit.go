// 끝내기 확인 창이다. 저장 안 한 것이 있을 때 ctrl+c 를 누르면 뜬다. 저장 후 끝 · 버리고 끝 · 취소를
// 고른다. (ADR-0007)
//
// 고르는 길이 셋이다. 버튼마다 글자 키(s · d · Esc)가 있고, 좌우 방향키로 옮겨 Enter 로 고르고, 버튼을
// 눌러 고른다. 창 바깥을 누르면 취소다. 창이 떠 있을 때 ctrl+c 를 한 번 더 누르면 버리고 끝낸다. 두 번
// 누르면 끝나는 길이 손에 익어 있어서다.

package sketch

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const quitTitle = "저장 안 한 것이 있다"

// quitChoice 는 확인 창의 버튼 하나다. key 는 그 버튼을 바로 고르는 키이고 버튼에 적힌다.
type quitChoice struct {
	key, label string
}

// quitChoices 는 버튼이다. 목록 차례가 곧 왼쪽부터의 차례이고, 번호가 곧 viewQuit.cursor 다.
var quitChoices = []quitChoice{
	{key: "s", label: "저장 후 끝"},
	{key: "d", label: "버리고 끝"},
	{key: "esc", label: "취소"},
}

const (
	quitSave = iota
	quitDiscard
	quitCancel
)

// quitButtonGap 은 버튼 사이 빈칸이다. 그리는 곳과 누름을 재는 곳이 같은 값을 봐야 한다.
const quitButtonGap = "  "

// quitFocusStyle 은 방향키가 짚은 버튼이다.
var quitFocusStyle = lipgloss.NewStyle().Reverse(true)

// viewQuit 는 끝내기 확인 창이 떠 있는 화면이다. under 는 창 밑의 모드 화면이고 취소하면 그리로 돌아간다.
// cursor 는 방향키가 짚은 버튼이다. 처음에는 잃는 것이 없는 "저장 후 끝" 이다.
type viewQuit struct {
	*sketch
	under  tea.Model
	cursor int
}

func (v *viewQuit) Init() tea.Cmd { return nil }

func (v *viewQuit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg, tea.MouseReleaseMsg:
		v.under.Update(msg)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return v.choose(quitDiscard)
		case "left":
			v.cursor = max(v.cursor-1, 0)
		case "right":
			v.cursor = min(v.cursor+1, len(quitChoices)-1)
		case "enter":
			return v.choose(v.cursor)
		}
		for i, choice := range quitChoices {
			if msg.String() == choice.key {
				return v.choose(i)
			}
		}
		// ctrl+s 로 저장만 하고 창을 둘 수도 있다. 저장되면 저장 안 한 것이 없어지지만 창은 스스로 닫지
		// 않는다. 무엇을 할지는 고르는 사람이 정한다.
		if slices.Contains(sketchKeys, msg.String()) {
			return v.keyOver(v, v.under, msg)
		}
	case tea.MouseClickMsg:
		return v.click(tea.Mouse(msg))
	}
	return v, nil
}

func (v *viewQuit) View() tea.View { return v.overlay(v.under, v.box()) }

// choose 는 버튼 i 를 고른 것이다. 저장에 실패하면 끝내지 않고 창을 닫는다. 그 까닭은 띠에 있다(save).
func (v *viewQuit) choose(i int) (tea.Model, tea.Cmd) {
	switch i {
	case quitSave:
		v.save()
		if v.dirty {
			return v.under, nil
		}
		return v, tea.Quit
	case quitDiscard:
		return v, tea.Quit
	}
	return v.under, nil
}

// buttons 는 버튼 줄이다. 짚은 버튼은 뒤집어 보인다.
func (v *viewQuit) buttons() string {
	parts := []string{}
	for i, choice := range quitChoices {
		button := quitButtonLabel(choice)
		if i == v.cursor {
			button = quitFocusStyle.Render(button)
		}
		parts = append(parts, button)
	}
	return strings.Join(parts, quitButtonGap)
}

func quitButtonLabel(choice quitChoice) string {
	key := choice.key
	if key == "esc" {
		key = "Esc"
	}
	return "[" + key + " " + choice.label + "]"
}

// box 는 확인 창이다. 제목 한 줄, 빈 줄, 버튼 줄이다.
func (v *viewQuit) box() string {
	return popupStyle.Padding(0, 1).Render(strings.Join([]string{quitTitle, "", v.buttons()}, "\n"))
}

// click 은 확인 창의 누름이다. 버튼을 누르면 그것을 고르고, 창 바깥을 누르면 취소다.
func (v *viewQuit) click(mouse tea.Mouse) (tea.Model, tea.Cmd) {
	box := v.box()
	if v.outsidePopup(box, mouse) {
		return v.under, nil
	}
	if mouse.Button != tea.MouseLeft {
		return v, nil
	}
	x, y := v.insidePopup(box, mouse)
	if y != 2 {
		return v, nil
	}
	x-- // 왼쪽 여백 한 칸
	left := 0
	for i, choice := range quitChoices {
		width := ansi.StringWidth(quitButtonLabel(choice))
		if x >= left && x < left+width {
			return v.choose(i)
		}
		left += width + ansi.StringWidth(quitButtonGap)
	}
	return v, nil
}
