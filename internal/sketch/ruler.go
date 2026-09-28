// 판의 테두리다. 빈 판은 터미널 바탕과 똑같아 어디까지가 판인지, 지금 어디를 보고 있는지 알 수
// 없다. 그래서 위에 열 눈금, 왼쪽에 줄 번호를 두고, 화면이 판보다 크면 판 밖을 흐린 점으로 채운다.
//
// 테두리 선이 아니라 무늬인 것은, 판과 화면이 같은 크기면 선을 그을 곳이 없고 무늬는 칸을 안
// 먹기 때문이다. 점은 `⋅`(U+22C5)다. 흔한 가운데 점 `·`(U+00B7)은 East Asian Ambiguous 라 터미널에
// 따라 두 칸을 먹어 줄이 밀린다. (ADR-0001 §4)

package sketch

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const offCanvasGlyph = "⋅"

var (
	offCanvasStyle = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(238))
	rulerStyle     = dimStyle
	// hoverRulerStyle 는 마우스가 짚은 열과 줄의 눈금이다. 띠의 (x,y) 를 눈으로 따라가지 않아도 된다.
	hoverRulerStyle = lipgloss.NewStyle().Reverse(true)
)

// board 는 눈금과 줄 번호를 두른 판이다. 띠를 뺀 화면 전체(viewHeight 줄)를 채운다. canvas 는
// s.canvas 이거나, 끄는 중인 모양을 그린 그 사본이다.
func (s *sketch) board(canvas *Canvas) string {
	width, height := s.canvasViewWidth(), s.canvasViewHeight()
	// 판이 보이는 칸 수다. 이 오른쪽은 판 밖이다.
	visible := max(min(width, s.canvas.Width-s.left), 0)
	canvasLines := strings.Split(canvas.renderArea(s.left, s.top, width, height), "\n")

	lines := []string{strings.Repeat(" ", s.gutter()) + s.columnRuler(visible)}
	for row := range height {
		y := s.top + row
		if y >= s.canvas.Height {
			lines = append(lines, strings.Repeat(" ", s.gutter())+offCanvasStyle.Render(strings.Repeat(offCanvasGlyph, width)))
			continue
		}
		// 오른쪽 끝에 넓은 글자가 걸리면 한 칸 넘친다. 잘라 낸 곳은 판 안이라 빈칸으로 메운다.
		line := ansi.Truncate(canvasLines[row], visible, "")
		line += strings.Repeat(" ", visible-ansi.StringWidth(line))
		line += offCanvasStyle.Render(strings.Repeat(offCanvasGlyph, width-visible))
		lines = append(lines, s.rowLabel(y)+line)
	}
	return strings.Join(lines, "\n")
}

// columnRuler 는 판 위의 열 눈금이다. 10 칸마다 번호, 5 칸마다 `|`, 나머지는 `.` 이다. 번호가
// 두 자리 이상이면 뒤 칸의 `.` 을 덮는다.
func (s *sketch) columnRuler(visible int) string {
	marks := make([]string, visible)
	for i := range marks {
		x := s.left + i
		switch {
		case x%5 == 0:
			marks[i] = "|"
		default:
			marks[i] = "."
		}
	}
	for i := range marks {
		x := s.left + i
		if x%10 != 0 {
			continue
		}
		for j, digit := range strconv.Itoa(x) {
			if i+j < visible {
				marks[i+j] = string(digit)
			}
		}
	}
	// 짚은 열 앞 · 짚은 열 · 뒤 셋으로 나눠 칠한다. 칸마다 칠하면 그릴 때마다 느려진다(renderArea).
	hover := s.hoverX - s.left
	if hover < 0 || hover >= visible {
		return rulerStyle.Render(strings.Join(marks, ""))
	}
	return rulerStyle.Render(strings.Join(marks[:hover], "")) +
		hoverRulerStyle.Render(marks[hover]) +
		rulerStyle.Render(strings.Join(marks[hover+1:], ""))
}

// rowLabel 은 판 왼쪽의 줄 번호다. 오른쪽으로 붙이고 판과 한 칸 띄운다.
func (s *sketch) rowLabel(y int) string {
	label := fmt.Sprintf("%*d", s.gutter()-1, y)
	if y == s.hoverY {
		return hoverRulerStyle.Render(label) + " "
	}
	return rulerStyle.Render(label) + " "
}
