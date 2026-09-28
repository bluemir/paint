// 모양이다. 도구(브러시 · 칠하기 · 지우기)가 "무엇을 하나" 라면 모양은 "어디에 하나" 다. 둘을 따로
// 고른다. 도구 셋 × 모양 넷이 모드 열둘이 되지 않게 하려는 것이다. (ADR-0001 §4)
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

type figure int

const (
	figureDot figure = iota
	figureLine
	figureBox
	figureFill
)

func (f figure) String() string {
	switch f {
	case figureLine:
		return "직선"
	case figureBox:
		return "테두리"
	case figureFill:
		return "채움"
	}
	return "한 칸씩"
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

// points 는 모양이 덮는 칸이다. 가로로는 step 칸씩 건넌다. 한 칸씩은 끌기가 아니라 여기 오지 않는다.
//
//   - 직선: 가로나 세로. 끌어 간 거리가 긴 축을 따르고, 짧은 축은 누른 곳에 붙인다.
//   - 테두리: 두 모서리를 잇는 상자의 가장자리. 안은 건드리지 않는다.
//   - 채움: 그 상자를 안까지 통째로.
func (f figure) points(d drag, step int) []point {
	columns := stampColumns(d.fromX, d.toX, step)
	out := []point{}
	switch f {
	case figureLine:
		if abs(d.toX-d.fromX) >= abs(d.toY-d.fromY) {
			for _, x := range columns {
				out = append(out, point{x, d.fromY})
			}
			return out
		}
		for _, y := range span(d.fromY, d.toY) {
			out = append(out, point{d.fromX, y})
		}
	case figureBox:
		top, bottom := min(d.fromY, d.toY), max(d.fromY, d.toY)
		for _, x := range columns {
			out = append(out, point{x, top}, point{x, bottom})
		}
		for _, y := range span(top, bottom) {
			out = append(out, point{columns[0], y}, point{columns[len(columns)-1], y})
		}
	case figureFill:
		for _, y := range span(d.fromY, d.toY) {
			for _, x := range columns {
				out = append(out, point{x, y})
			}
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

// toolStep 은 모양을 따라 가로로 몇 칸씩 건너 쓸지다. 브러시만 붓 폭이고, 칠하기 · 지우기는 칸마다다.
func (s *sketch) toolStep() int {
	if s.mode == modeBrush {
		return cellWidth(s.brush)
	}
	return 1
}

// apply 는 지금 도구를 canvas 의 한 칸에 쓴다.
func (s *sketch) apply(canvas *Canvas, x, y int) {
	switch s.mode {
	case modeBrush:
		canvas.Put(x, y, s.brush)
	case modePaint:
		canvas.Recolor(x, y, s.brush.Fg, s.brush.Bg)
	case modeErase:
		canvas.Erase(x, y)
	}
}

// drawDrag 는 끌고 있는 모양을 canvas 에 그린다.
func (s *sketch) drawDrag(canvas *Canvas) {
	for _, at := range s.figure.points(*s.dragging, s.toolStep()) {
		s.apply(canvas, at.x, at.y)
	}
}

// figureClick 은 직선 · 테두리 · 채움의 누름이다. 좌클릭이 끌기를 시작하고, 끄는 중 우클릭은 버린다.
// 끄는 중이 아닐 때 우클릭은 다른 때처럼 지우개다.
func (s *sketch) figureClick(button tea.MouseButton, x, y int) {
	switch {
	case button == tea.MouseLeft:
		s.dragging = &drag{fromX: x, fromY: y, toX: x, toY: y}
	case button == tea.MouseRight && s.dragging != nil:
		s.dragging = nil
	default:
		s.stroke(button, x, y)
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
