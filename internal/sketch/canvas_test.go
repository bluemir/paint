package sketch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestPutWideGlyphTakesTwoCells(t *testing.T) {
	canvas := NewCanvas(4, 1)
	if width := canvas.Put(1, 0, Cell{Glyph: "한", Fg: NoColor, Bg: NoColor}); width != 2 {
		t.Fatalf("폭 = %d, 2 여야 한다", width)
	}
	if !canvas.At(2, 0).continuation() {
		t.Errorf("오른쪽 칸이 이어짐 칸이 아니다: %+v", canvas.At(2, 0))
	}
	if got := ansi.Strip(canvas.renderArea(0, 0, 4, 1)); got != " 한 " {
		t.Errorf("그림 = %q", got)
	}
}

// 넓은 글자는 오른쪽 끝에 반쪽만 걸쳐 놓지 않는다.
func TestPutWideGlyphRefusesRightEdge(t *testing.T) {
	canvas := NewCanvas(3, 1)
	if width := canvas.Put(2, 0, Cell{Glyph: "한", Fg: NoColor, Bg: NoColor}); width != 0 {
		t.Errorf("폭 = %d, 놓지 않아야 한다", width)
	}
	if canvas.At(2, 0) != blank {
		t.Errorf("칸이 바뀌었다: %+v", canvas.At(2, 0))
	}
}

// 넓은 글자의 어느 반쪽을 덮든 남은 반쪽이 홀로 남지 않는다.
func TestOverwritingHalfOfWideGlyphClearsTheOtherHalf(t *testing.T) {
	for _, x := range []int{0, 1} {
		canvas := NewCanvas(3, 1)
		canvas.Put(0, 0, Cell{Glyph: "한", Fg: NoColor, Bg: NoColor})
		canvas.Put(x, 0, Cell{Glyph: "a", Fg: NoColor, Bg: NoColor})
		if got := ansi.Strip(canvas.renderArea(0, 0, 3, 1)); ansi.StringWidth(got) != 3 || strings.Contains(got, "한") {
			t.Errorf("x=%d 를 덮은 그림 = %q", x, got)
		}
	}
}

func TestEraseWideGlyphFromEitherHalf(t *testing.T) {
	for _, x := range []int{0, 1} {
		canvas := NewCanvas(2, 1)
		canvas.Put(0, 0, Cell{Glyph: "한", Fg: NoColor, Bg: NoColor})
		canvas.Erase(x, 0)
		if canvas.At(0, 0) != blank || canvas.At(1, 0) != blank {
			t.Errorf("x=%d 를 지운 뒤 = %+v %+v", x, canvas.At(0, 0), canvas.At(1, 0))
		}
	}
}

// 왼쪽 끝에 넓은 글자의 오른쪽 반쪽만 보이면 공백으로 채워 폭이 맞는다.
func TestRenderAreaPadsWideGlyphCutOnTheLeft(t *testing.T) {
	canvas := NewCanvas(4, 1)
	canvas.Put(0, 0, Cell{Glyph: "한", Fg: NoColor, Bg: NoColor})
	if got := ansi.Strip(canvas.renderArea(1, 0, 3, 1)); got != "   " {
		t.Errorf("그림 = %q", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mock.json")
	canvas := NewCanvas(5, 2)
	canvas.Put(0, 0, Cell{Glyph: "┌", Fg: 250, Bg: NoColor})
	canvas.Put(1, 1, Cell{Glyph: "한", Fg: NoColor, Bg: 17})
	canvas.Put(4, 1, Cell{Glyph: " ", Fg: NoColor, Bg: 196})
	canvas.Put(3, 0, Cell{Glyph: "e\u0301", Fg: 3, Bg: NoColor}) // e 와 결합 문자, 두 rune 이 한 칸이다
	if err := canvas.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Width != 5 || loaded.Height != 2 {
		t.Fatalf("크기 = %dx%d", loaded.Width, loaded.Height)
	}
	for y := range 2 {
		for x := range 5 {
			if loaded.At(x, y) != canvas.At(x, y) {
				t.Errorf("(%d,%d) = %+v, %+v 여야 한다", x, y, loaded.At(x, y), canvas.At(x, y))
			}
		}
	}
}

// 파일은 줄마다 한 줄이다. 넓은 글자는 문자열에서 두 칸을 먹고 색은 두 번 적힌다. 색 없음은 null
// 이고, < > & 는 그대로 적힌다.
func TestSaveWritesRowsPerLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mock.json")
	canvas := NewCanvas(4, 2)
	canvas.Put(0, 0, Cell{Glyph: "집", Fg: 252, Bg: NoColor})
	canvas.Put(3, 0, Cell{Glyph: "┌", Fg: 250, Bg: NoColor})
	canvas.Put(0, 1, Cell{Glyph: "<", Fg: NoColor, Bg: 17})
	canvas.Put(1, 1, Cell{Glyph: "&", Fg: NoColor, Bg: NoColor})
	if err := canvas.Save(path); err != nil {
		t.Fatal(err)
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
	"size": {"width": 4, "height": 2},
	"layers": [
		{
			"glyphs": [
				"집 ┌",
				"<&  "
			],
			"fg": [
				[252,252,null,250],
				[null,null,null,null]
			],
			"bg": [
				[null,null,null,null],
				[17,null,null,null]
			]
		}
	]
}
`
	if string(buf) != want {
		t.Errorf("파일 =\n%s\n이어야 한다\n%s", buf, want)
	}
}

// 형식이 어긋난 파일은 열지 않는다.
func TestLoadRejectsBadFiles(t *testing.T) {
	for name, text := range map[string]string{
		"레이어 둘":    `{"size":{"width":1,"height":1},"layers":[{"glyphs":[" "],"fg":[[null]],"bg":[[null]]},{"glyphs":[" "],"fg":[[null]],"bg":[[null]]}]}`,
		"레이어 없음":   `{"size":{"width":1,"height":1},"layers":[]}`,
		"줄 모자람":    `{"size":{"width":1,"height":2},"layers":[{"glyphs":[" "],"fg":[[null]],"bg":[[null]]}]}`,
		"글자 짧음":    `{"size":{"width":2,"height":1},"layers":[{"glyphs":[" "],"fg":[[null,null]],"bg":[[null,null]]}]}`,
		"글자 김":     `{"size":{"width":1,"height":1},"layers":[{"glyphs":["ab"],"fg":[[null]],"bg":[[null]]}]}`,
		"넓은 글자 넘침": `{"size":{"width":1,"height":1},"layers":[{"glyphs":["한"],"fg":[[null]],"bg":[[null]]}]}`,
		"색 모자람":    `{"size":{"width":2,"height":1},"layers":[{"glyphs":["ab"],"fg":[[null]],"bg":[[null,null]]}]}`,
		"예전 형식":    `{"width":1,"height":1,"cells":[]}`,
	} {
		path := filepath.Join(t.TempDir(), "mock.json")
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Errorf("%s: 열렸다", name)
		}
	}
}

// 색칠은 글자를 두고 색만 바꾼다. 넓은 글자는 어느 반쪽을 짚든 두 칸이 함께 바뀐다.
func TestRecolorKeepsGlyph(t *testing.T) {
	for _, x := range []int{0, 1} {
		canvas := NewCanvas(3, 1)
		canvas.Put(0, 0, Cell{Glyph: "한", Fg: 1, Bg: NoColor})
		canvas.Recolor(x, 0, 200, 17)
		head, tail := canvas.At(0, 0), canvas.At(1, 0)
		if head != (Cell{Glyph: "한", Fg: 200, Bg: 17}) || !tail.continuation() || tail.Fg != 200 || tail.Bg != 17 {
			t.Errorf("x=%d 색칠 뒤 = %+v %+v", x, head, tail)
		}
		if canvas.At(2, 0) != blank {
			t.Errorf("x=%d: 옆 칸까지 바뀌었다 %+v", x, canvas.At(2, 0))
		}
	}
}
