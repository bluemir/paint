// 색표다(c). 256색을 16x16 으로 늘어놓고 좌클릭으로 전경, 우클릭으로 배경을 고른다.

package sketch

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// 색표: 제목 한 줄, 16x16 격자(한 색이 두 칸), "없음" 한 줄. 테두리 안쪽 기준이다.
const (
	swatchWidth   = 2
	colorColumns  = 16
	colorRows     = 16
	colorTitle    = "색표  좌클릭 전경 · 우클릭 배경 · Esc 닫기"
	noColorButton = "[없음]"
)

// viewColor 는 색표가 떠 있는 화면이다. under 는 창 밑의 모드 화면이고 닫으면 그리로 돌아간다. 창은 여러
// 번 골라도 떠 있다(전경과 배경을 잇달아 고르므로). 창 바깥을 누르거나 Esc 를 누르면 닫힌다.
type viewColor struct {
	*sketch
	under tea.Model
}

func (v *viewColor) Init() tea.Cmd { return nil }

func (v *viewColor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg, tea.MouseReleaseMsg:
		v.under.Update(msg)
	case tea.KeyPressMsg:
		if slices.Contains(sketchKeys, msg.String()) {
			return v.keyOver(v, v.under, msg)
		}
		if msg.String() == "esc" {
			return v.under, nil
		}
	case tea.MouseClickMsg:
		return v.pick(tea.Mouse(msg))
	}
	return v, nil
}

func (v *viewColor) View() tea.View { return v.overlay(v.under, colorBox()) }

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

// pick 은 색표의 누름이다. 칸을 누르면 그 색을 붓에 담고, 창 바깥을 누르면 창을 닫는다. 그 누름은 판에
// 닿지 않는다.
func (v *viewColor) pick(mouse tea.Mouse) (tea.Model, tea.Cmd) {
	box := colorBox()
	if v.outsidePopup(box, mouse) {
		return v.under, nil
	}
	x, y := v.insidePopup(box, mouse)
	var color Color
	switch {
	case y >= 1 && y <= colorRows && x >= 0 && x < colorColumns*swatchWidth:
		color = Color((y-1)*colorColumns + x/swatchWidth)
	case y == colorRows+1 && x >= 0 && x < ansi.StringWidth(noColorButton):
		color = NoColor
	default:
		return v, nil
	}
	switch mouse.Button {
	case tea.MouseLeft:
		v.brush.Fg = color
	case tea.MouseRight:
		v.brush.Bg = color
	}
	return v, nil
}
