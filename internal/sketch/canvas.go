// 그림판이 그리는 판이다. 칸마다 글자 하나와 전경·배경색을 든다.
//
// 넓은 글자(한글, 전각 글자)는 터미널에서 두 칸을 먹는다. 그래서 그 글자는 제 칸(x)에 두고
// 오른쪽 칸(x+1)은 이어짐 칸(Glyph "")으로 비워 둔다. 그리는 쪽은 이어짐 칸을 건너뛴다.
// 넓은 글자의 반쪽을 덮으면 남은 반쪽이 홀로 남으므로 그 짝을 공백으로 되돌린다.
//
// 파일은 JSON 이다. 글자는 줄마다 문자열 하나, 색은 줄마다 화면 칸 수만큼의 배열이다. 다시 열어 고치고,
// 사람이나 Claude 가 읽어 해석하려는 것이라 ANSI 가 아니라 JSON 이다. 글자를 줄 문자열로 두어 파일을 열면
// 그림이 그대로 보인다. 들여쓰기는 표준(json.MarshalIndent)에 맡긴다. (ADR-0001, ADR-0003)

package sketch

import (
	"encoding/json"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/cockroachdb/errors"
)

// Color 는 터미널 256색의 번호다. NoColor 는 터미널 기본색이다.
type Color int

const NoColor Color = -1

type Cell struct {
	Glyph  string
	Fg, Bg Color
}

// blank 는 아무것도 없는 칸이다. 파일에 적지 않는 칸이 이것이다.
var blank = Cell{Glyph: " ", Fg: NoColor, Bg: NoColor}

// continuation 은 넓은 글자의 오른쪽 반쪽이다.
func (cell Cell) continuation() bool { return cell.Glyph == "" }

type Canvas struct {
	Width, Height int
	cells         [][]Cell
}

func NewCanvas(width, height int) *Canvas {
	cells := make([][]Cell, height)
	for y := range cells {
		cells[y] = make([]Cell, width)
		for x := range cells[y] {
			cells[y][x] = blank
		}
	}
	return &Canvas{Width: width, Height: height, cells: cells}
}

func (canvas *Canvas) inside(x, y int) bool {
	return x >= 0 && y >= 0 && x < canvas.Width && y < canvas.Height
}

func (canvas *Canvas) At(x, y int) Cell { return canvas.cells[y][x] }

// Put 은 (x, y) 에 cell 을 놓고 그 글자가 먹은 폭을 돌려준다. 판 밖이거나 넓은 글자가 오른쪽 끝에
// 안 들어가면 놓지 않고 0 을 돌려준다.
func (canvas *Canvas) Put(x, y int, cell Cell) int {
	width := ansi.StringWidth(cell.Glyph)
	if width < 1 || !canvas.inside(x, y) || !canvas.inside(x+width-1, y) {
		return 0
	}
	for i := range width {
		canvas.breakWide(x+i, y)
	}
	canvas.cells[y][x] = cell
	if width == 2 {
		canvas.cells[y][x+1] = Cell{Glyph: "", Fg: cell.Fg, Bg: cell.Bg}
	}
	return width
}

// Erase 는 (x, y) 를 빈 칸으로 되돌린다. 넓은 글자의 어느 반쪽이든 글자 전체가 지워진다.
func (canvas *Canvas) Erase(x, y int) {
	if !canvas.inside(x, y) {
		return
	}
	canvas.breakWide(x, y)
	canvas.cells[y][x] = blank
}

// Recolor 는 (x, y) 의 글자를 두고 전경 · 배경색만 바꾼다. 넓은 글자는 어느 반쪽을 짚든 두 칸이
// 함께 바뀐다. 빈 칸도 바뀌므로 빈 판에 배경만 칠할 수 있다.
func (canvas *Canvas) Recolor(x, y int, fg, bg Color) {
	if !canvas.inside(x, y) {
		return
	}
	if canvas.cells[y][x].continuation() && x > 0 {
		x--
	}
	canvas.cells[y][x].Fg, canvas.cells[y][x].Bg = fg, bg
	if x+1 < canvas.Width && canvas.cells[y][x+1].continuation() {
		canvas.cells[y][x+1].Fg, canvas.cells[y][x+1].Bg = fg, bg
	}
}

// breakWide 는 (x, y) 가 넓은 글자의 반쪽이면 그 글자의 두 칸을 모두 빈 칸으로 되돌린다.
func (canvas *Canvas) breakWide(x, y int) {
	row := canvas.cells[y]
	switch {
	case row[x].continuation():
		row[x] = blank
		if x > 0 {
			row[x-1] = blank
		}
	case x+1 < canvas.Width && row[x+1].continuation():
		row[x] = blank
		row[x+1] = blank
	}
}

// renderArea 는 판 중 (left, top) 부터 width x height 만 그린다. 왼쪽 끝에 넓은 글자의 반쪽만
// 걸리면 공백으로 채운다. 오른쪽 끝에 걸린 넓은 글자는 부르는 쪽이 잘라 낸다(board).
//
// 색이 같은 칸이 잇달면 한 번에 칠한다. 칸마다 칠하면 120x40 판에서 한 번 그리는 데 칸 4800 번의
// 스타일 렌더가 들어, 마우스를 움직이거나 굴릴 때마다 그것이 쌓여 반응이 느렸다.
func (canvas *Canvas) renderArea(left, top, width, height int) string {
	lines := []string{}
	for y := top; y < min(top+height, canvas.Height); y++ {
		var line, run strings.Builder
		runColor := canvas.cells[y][min(left, canvas.Width-1)]
		flush := func() {
			line.WriteString(runColor.paint(run.String()))
			run.Reset()
		}
		for x := left; x < min(left+width, canvas.Width); x++ {
			cell := canvas.cells[y][x]
			glyph := cell.Glyph
			if cell.continuation() {
				if x != left {
					continue
				}
				glyph = " "
			}
			if cell.Fg != runColor.Fg || cell.Bg != runColor.Bg {
				flush()
				runColor = cell
			}
			run.WriteString(glyph)
		}
		flush()
		lines = append(lines, line.String())
	}
	return strings.Join(lines, "\n")
}

// Render 는 판 전체를 ANSI 글로 그린다. 그림판을 열지 않고 터미널에 바로 찍는 데 쓴다.
func (canvas *Canvas) Render() string {
	return canvas.renderArea(0, 0, canvas.Width, canvas.Height)
}

// paint 는 text 를 이 칸의 색으로 칠한다. 색이 없으면 스타일을 거치지 않는다.
func (cell Cell) paint(text string) string {
	if text == "" || (cell.Fg == NoColor && cell.Bg == NoColor) {
		return text
	}
	return cell.style().Render(text)
}

func (cell Cell) style() lipgloss.Style {
	style := lipgloss.NewStyle()
	if cell.Fg != NoColor {
		style = style.Foreground(lipgloss.ANSIColor(cell.Fg))
	}
	if cell.Bg != NoColor {
		style = style.Background(lipgloss.ANSIColor(cell.Bg))
	}
	return style
}

// 파일에 적히는 꼴이다. 레이어는 아직 하나만 쓴다(레이어 기능 전에 형식만 맞춰 둔다).
//
// glyphs 는 줄마다 판 폭만큼의 칸을 채운 문자열이다. 넓은 글자는 문자열 안에서 두 칸을 먹고, 그 오른쪽
// 반쪽(이어짐 칸)은 문자열에 없다. fg · bg 는 줄마다 판 폭 길이의 배열이고 x 가 곧 번호다. 넓은 글자는
// 두 칸에 같은 색이 두 번 적힌다. 색이 null 이면 터미널 기본색이다.
type canvasFile struct {
	Size   sizeFile    `json:"size"`
	Layers []layerFile `json:"layers"`
}

type sizeFile struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type layerFile struct {
	Glyphs []string   `json:"glyphs"`
	Fg     [][]*Color `json:"fg"`
	Bg     [][]*Color `json:"bg"`
}

func colorOut(color Color) *Color {
	if color == NoColor {
		return nil
	}
	return &color
}

func colorIn(color *Color) Color {
	if color == nil {
		return NoColor
	}
	return *color
}

// Save 는 판을 파일로 쓴다. 들여쓰기는 표준(json.MarshalIndent)에 맡긴다. 색 배열은 칸마다 줄이 바뀌어
// 길어지지만, 그림은 glyphs 에서 줄마다 한 줄로 보인다.
func (canvas *Canvas) Save(path string) error {
	layer := layerFile{Glyphs: []string{}, Fg: [][]*Color{}, Bg: [][]*Color{}}
	for _, row := range canvas.cells {
		var line strings.Builder
		fgRow, bgRow := []*Color{}, []*Color{}
		for _, cell := range row {
			if !cell.continuation() {
				line.WriteString(cell.Glyph)
			}
			fgRow, bgRow = append(fgRow, colorOut(cell.Fg)), append(bgRow, colorOut(cell.Bg))
		}
		layer.Glyphs = append(layer.Glyphs, line.String())
		layer.Fg, layer.Bg = append(layer.Fg, fgRow), append(layer.Bg, bgRow)
	}
	out := canvasFile{Size: sizeFile{Width: canvas.Width, Height: canvas.Height}, Layers: []layerFile{layer}}
	buf, err := json.MarshalIndent(out, "", "\t")
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(os.WriteFile(path, append(buf, '\n'), 0o644))
}

func Load(path string) (*Canvas, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var in canvasFile
	if err := json.Unmarshal(buf, &in); err != nil {
		return nil, errors.Wrapf(err, "%s", path)
	}
	width, height := in.Size.Width, in.Size.Height
	if width < 1 || height < 1 {
		return nil, errors.Errorf("%s: 판 크기가 없다 (%dx%d)", path, width, height)
	}
	if len(in.Layers) != 1 {
		return nil, errors.Errorf("%s: 레이어가 %d 개다. 아직 하나만 읽는다", path, len(in.Layers))
	}
	layer := in.Layers[0]
	if len(layer.Glyphs) != height || len(layer.Fg) != height || len(layer.Bg) != height {
		return nil, errors.Errorf("%s: 줄 수가 판 높이 %d 와 다르다 (glyphs %d, fg %d, bg %d)",
			path, height, len(layer.Glyphs), len(layer.Fg), len(layer.Bg))
	}
	canvas := NewCanvas(width, height)
	for y := range height {
		if len(layer.Fg[y]) != width || len(layer.Bg[y]) != width {
			return nil, errors.Errorf("%s: %d 째 줄의 색이 판 폭 %d 와 다르다 (fg %d, bg %d)",
				path, y, width, len(layer.Fg[y]), len(layer.Bg[y]))
		}
		// 글자를 grapheme 하나씩 떼어 폭만큼 나아간다. 이어짐 칸의 색은 Put 이 제 글자 색으로 채우므로
		// 파일의 그 칸 값은 보지 않는다.
		x, rest := 0, layer.Glyphs[y]
		for rest != "" {
			glyph, _ := ansi.FirstGraphemeCluster(rest, ansi.GraphemeWidth)
			rest = rest[len(glyph):]
			if x >= width {
				return nil, errors.Errorf("%s: %d 째 줄이 판 폭 %d 보다 길다", path, y, width)
			}
			put := canvas.Put(x, y, Cell{Glyph: glyph, Fg: colorIn(layer.Fg[y][x]), Bg: colorIn(layer.Bg[y][x])})
			if put == 0 {
				return nil, errors.Errorf("%s: (%d,%d) 의 %q 를 놓을 수 없다", path, x, y, glyph)
			}
			x += put
		}
		if x != width {
			return nil, errors.Errorf("%s: %d 째 줄이 %d 칸이다. 판 폭 %d 여야 한다", path, y, x, width)
		}
	}
	return canvas, nil
}
