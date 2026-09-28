// 모양이다. 도구(브러시 · 칠하기 · 지우기)가 "무엇을 하나" 라면 모양은 "어디에 하나" 다. 둘을 따로
// 고른다. 도구 셋 × 모양 넷이 모드 열둘이 되지 않게 하려는 것이다. (ADR-0001 §4)
//
// 모양마다 타입이 하나다(figure). 새 모양은 타입 하나를 두고 figures 에 올리면 Tab · 도구 줄 · 팔레트에
// 다 나온다. (ADR-0004)
//
//   - 한 칸씩: 누른 채 지나간 칸마다 도구를 쓴다.
//   - 직선 · 테두리 · 채움: 누른 곳에서 끌고 가는 동안 모양이 미리 보이고, 버튼을 떼면 판에 적힌다.
//     우클릭은 끌던 것을 버린다.
//
// 브러시는 붓 글자를 모양을 따라 찍는다. 선 글자(─│┌┐)로 모서리를 맞춰 그리지 않고 만나는 선을 이어
// 주지도 않는다. 나중에 그린 것이 덮는다. 넓은 붓(한글, 전각 글자)은 가로로 두 칸씩 건너 찍는다. 한
// 칸씩 찍으면 앞 글자의 반쪽을 덮어 둘 다 깨진다. 칠하기 · 지우기는 칸마다 쓴다.

package sketch

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type figure interface {
	// String 은 도구 줄과 띠에 보이는 이름이다. command 는 명령 팔레트에 오르는 이름이다.
	String() string
	command() string
	// hint 는 띠에 적는 마우스가 하는 일이다. 한 칸씩은 도구마다 다르므로 도구의 안내(toolHint)를 그대로
	// 쓰고, 끄는 모양은 도구와 상관없이 같다.
	hint(toolHint string) string
	// press 와 move 는 판 위의 누름과, 누른 채 움직인 것이다. 도구(figureMode)가 넘겨준다.
	press(s *sketch, button tea.MouseButton, x, y int)
	move(s *sketch, button tea.MouseButton, x, y int)
	// points 는 끌어 간 모양이 덮는 칸이다. 가로로는 step 칸씩 건넌다. 한 칸씩은 끌지 않아 쓰지 않는다.
	points(d drag, step int) []point
}

var (
	figureDot  figure = dotFigure{}
	figureLine figure = lineFigure{dragFigure{name: "직선", commandName: "line"}}
	figureBox  figure = boxFigure{dragFigure{name: "테두리", commandName: "box"}}
	figureFill figure = fillFigure{dragFigure{name: "채움", commandName: "fill"}}
)

// figures 는 모양 목록이다. 목록 차례가 곧 Tab 이 도는 차례이자 도구 줄과 팔레트에 뜨는 차례다.
var figures = []figure{figureDot, figureLine, figureBox, figureFill}

// dotFigure 는 누른 채 지나간 칸마다 도구를 쓴다.
type dotFigure struct{}

func (dotFigure) String() string              { return "한 칸씩" }
func (dotFigure) command() string             { return "dot" }
func (dotFigure) hint(toolHint string) string { return toolHint }

func (dotFigure) press(s *sketch, button tea.MouseButton, x, y int) { s.stroke(button, x, y) }
func (dotFigure) move(s *sketch, button tea.MouseButton, x, y int)  { s.stroke(button, x, y) }

func (dotFigure) points(drag, int) []point { return nil }

// dragFigure 는 끌어서 그리는 모양(직선 · 테두리 · 채움)이 함께 쓰는 부분이다. 누른 곳에서 끌고 가는
// 동안 판의 사본에 모양을 그려 미리 보이고(View), 버튼을 떼면 원본에 적는다(release). 덮는 칸
// (points)은 품은 쪽이 정한다.
type dragFigure struct {
	name, commandName string
}

func (f dragFigure) String() string   { return f.name }
func (f dragFigure) command() string  { return f.commandName }
func (dragFigure) hint(string) string { return "끌어 그림·우 취소" }

// press 는 좌클릭이 끌기를 시작하고, 끄는 중 우클릭은 끌던 것을 버린다. 끄는 중이 아닐 때 우클릭은
// 다른 때처럼 지우개다.
func (dragFigure) press(s *sketch, button tea.MouseButton, x, y int) {
	switch {
	case button == tea.MouseLeft:
		s.dragging = &drag{fromX: x, fromY: y, toX: x, toY: y}
	case button == tea.MouseRight && s.dragging != nil:
		s.dragging = nil
	default:
		s.stroke(button, x, y)
	}
}

// move 는 끄는 동안 끝만 옮긴다. 칸에는 버튼을 뗄 때 적는다.
func (dragFigure) move(s *sketch, _ tea.MouseButton, x, y int) {
	if s.dragging != nil {
		s.dragging.toX, s.dragging.toY = x, y
	}
}

// drag 는 끌고 있는 모양의 두 끝이다.
type drag struct {
	fromX, fromY, toX, toY int
}

// point 는 판의 칸 하나다.
type point struct{ x, y int }

// stampColumns 는 from 에서 to 쪽으로 step 칸씩 건너 찍을 열이다. 마지막 글자가 to 를 넘지 않는다.
// to 가 from 보다 왼쪽이어도 된다.
func stampColumns(from, to, step int) []int {
	left, right := min(from, to), max(from, to)
	columns := []int{left}
	for x := left + step; x+step-1 <= right; x += step {
		columns = append(columns, x)
	}
	return columns
}

func span(from, to int) []int {
	out := []int{}
	for i := min(from, to); i <= max(from, to); i++ {
		out = append(out, i)
	}
	return out
}

// lineFigure 는 가로나 세로 직선이다. 끌어 간 거리가 긴 축을 따르고, 짧은 축은 누른 곳에 붙인다.
type lineFigure struct{ dragFigure }

func (lineFigure) points(d drag, step int) []point {
	out := []point{}
	if abs(d.toX-d.fromX) >= abs(d.toY-d.fromY) {
		for _, x := range stampColumns(d.fromX, d.toX, step) {
			out = append(out, point{x, d.fromY})
		}
		return out
	}
	for _, y := range span(d.fromY, d.toY) {
		out = append(out, point{d.fromX, y})
	}
	return out
}

// boxFigure 는 두 모서리를 잇는 상자의 가장자리다. 안은 건드리지 않는다.
type boxFigure struct{ dragFigure }

func (boxFigure) points(d drag, step int) []point {
	columns := stampColumns(d.fromX, d.toX, step)
	top, bottom := min(d.fromY, d.toY), max(d.fromY, d.toY)
	out := []point{}
	for _, x := range columns {
		out = append(out, point{x, top}, point{x, bottom})
	}
	for _, y := range span(top, bottom) {
		out = append(out, point{columns[0], y}, point{columns[len(columns)-1], y})
	}
	return out
}

// fillFigure 는 그 상자를 안까지 통째로 덮는다.
type fillFigure struct{ dragFigure }

func (fillFigure) points(d drag, step int) []point {
	out := []point{}
	for _, y := range span(d.fromY, d.toY) {
		for _, x := range stampColumns(d.fromX, d.toX, step) {
			out = append(out, point{x, y})
		}
	}
	return out
}

func abs(n int) int { return max(n, -n) }

// cellWidth 는 붓이 먹는 칸 수다. 빈 글자여도 한 칸으로 센다.
func cellWidth(brush Cell) int { return max(ansi.StringWidth(brush.Glyph), 1) }

// clone 은 미리 보기에 쓸 판의 사본이다. 끄는 동안 사본에 모양을 그려 보이고, 떼면 원본에 그린다.
func (canvas *Canvas) clone() *Canvas {
	out := &Canvas{Width: canvas.Width, Height: canvas.Height, cells: make([][]Cell, canvas.Height)}
	for y, row := range canvas.cells {
		out.cells[y] = append([]Cell(nil), row...)
	}
	return out
}

// drawDrag 는 끌고 있는 모양을 canvas 에 그린다.
func (s *sketch) drawDrag(canvas *Canvas) {
	for _, at := range s.figure.points(*s.dragging, s.mode.step(s)) {
		s.mode.apply(s, canvas, at.x, at.y)
	}
}

// release 는 버튼을 뗀 것이다. 끌던 모양이 있으면 판에 적는다. 판 밖에서 떼도 마지막으로 짚은 칸까지다.
func (s *sketch) release() {
	if s.dragging == nil {
		return
	}
	s.drawDrag(s.canvas)
	s.dragging = nil
	s.dirty = true
}

// setFigure 는 모양을 고른다. 끌던 것은 버린다. 모양이 바뀐 뒤 버튼을 떼면 무엇을 그릴지 모른다.
func (s *sketch) setFigure(next figure) {
	s.dragging = nil
	s.figure = next
}
