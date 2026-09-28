// 글자표다. 설계에 쓸 글자를 묶음마다 제목을 달아 한 창에 잇고, 휠이나 방향키로 굴린다.
//
// 폭이 0 이거나 두 칸보다 넓은 글자만 뺀다(drawable). 터미널에 따라 폭이 달라질 수 있는 글자도 두고,
// 그 칸에 바탕을 깔아 보인다(widthDoubt). 그 글자로 맞춘 줄은 다른 터미널에서 밀릴 수 있다. (ADR-0002)
//
// 글자는 유니코드 구간을 훑은 것(선 · 블록 · 화살표 · 점자 · 괘 등)과, zn 의 특수문자 표에서 가져온
// 것(curatedGlyphs)이다.

package sketch

import (
	"slices"
	"strings"
	"sync"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
	"golang.org/x/text/unicode/runenames"
	"golang.org/x/text/width"
)

const (
	minGlyphColumns = 16
	glyphCellWidth  = 3
	glyphTitle      = "글자표  클릭·키로 붓 · Tab 찾기 · Esc 닫기"
)

// blankNames 는 빈칸 글자와 그 이름이다. 그리면 안 보이므로 글자표와 띠가 이름이나 바탕으로 보인다.
var blankNames = map[string]string{
	" ": "(빈칸)",
	"　": "(전각 빈칸)",
}

// blankSwatch 는 글자표에서 빈칸 글자를 보이게 까는 바탕이다.
var blankSwatch = lipgloss.NewStyle().Background(lipgloss.ANSIColor(238))

// doubtSwatch 는 글자표에서 폭이 애매한 글자(widthDoubt)에 까는 바탕이다. 짙은 노랑이라 빈칸
// 바탕(238)과도, 마우스가 올라간 칸(244)과도 갈린다. 까닭은 띠의 붓에 적는다(brushLabel).
var doubtSwatch = lipgloss.NewStyle().Background(lipgloss.ANSIColor(58))

type glyphGroup struct {
	title  string
	glyphs []string
}

// glyphGroups 는 글자표의 묶음이다. 목록 차례가 곧 창에 뜨는 차례다.
//
// 글자표는 도중에 안 바뀌므로 한 번 만들어 둔다. 굴릴 때마다 유니코드 구간을 다시 훑으면 그만큼
// 느려진다.
var glyphGroups = sync.OnceValue(buildGlyphGroups)

func buildGlyphGroups() []glyphGroup {
	groups := []glyphGroup{
		{"빈칸", []string{" ", "　"}},
		{"선", runeRange(0x2500, 0x257F)},
		{"블록 · 도형", slices.Concat(runeRange(0x2580, 0x259F), runeRange(0x25A0, 0x25FF), runeList(curatedGlyphs["블록 · 도형"]))},
		{"화살표", slices.Concat(
			runeRange(0x2190, 0x21FF),   // Arrows
			runeRange(0x27F0, 0x27FF),   // Supplemental Arrows-A
			runeRange(0x2900, 0x297F),   // Supplemental Arrows-B
			runeRange(0x1F800, 0x1F8FF), // Supplemental Arrows-C
			runeList(curatedGlyphs["화살표"]),
		)},
		// 작은 묶음을 수학 앞에 둔다. 수학 기호 구간에 괄호(⟦⟧ ⦗⦘)가 들어 있어 먼저 나온 묶음이 가져간다.
		{"구두점", runeList(curatedGlyphs["구두점"])},
		{"괄호", runeList(curatedGlyphs["괄호"])},
		{"통화", runeList(curatedGlyphs["통화"])},
		{"수학", slices.Concat(
			runeList(curatedGlyphs["수학"]), // 자주 쓰는 것이 앞이다
			runeRange(0x2200, 0x22FF),     // Mathematical Operators
			runeRange(0x2A00, 0x2AFF),     // Supplemental Mathematical Operators
			runeRange(0x27C0, 0x27EF),     // Miscellaneous Mathematical Symbols-A
			runeRange(0x2980, 0x29FF),     // Miscellaneous Mathematical Symbols-B
		)},
		// 기호 앞에 둔다. 팔괘 ☰ 와 효 ⚊ 가 Miscellaneous Symbols 구간에 들어 있어 먼저 나온 묶음이 가져간다.
		{"괘", slices.Concat(runeRange(0x2630, 0x2637), runeRange(0x268A, 0x268F), runeRange(0x4DC0, 0x4DFF))},
		{"기호", slices.Concat(
			runeList(curatedGlyphs["기호"]), // 자주 쓰는 것이 앞이다
			runeRange(0x2600, 0x26FF),     // Miscellaneous Symbols
			runeRange(0x2700, 0x27BF),     // Dingbats
		)},
		// 수학 · 구두점 뒤에 둔다. ± × ÷ ° · 는 그쪽이 가져가고 여기는 남은 것(¤ ¬ ¯ ¸ 등)이 뜬다.
		{"라틴-1 기호", slices.Concat(runeRange(0x00A1, 0x00BF), runeList("×÷"))},
		// 글자 갈래가 분명한 구간은 zn 의 문자 · 숫자 묶음 앞에 둔다. 먼저 나온 묶음이 가져가므로 뒤에
		// 두면 α ① Ⅰ 이 저쪽으로 가고 여기는 남은 것만 뜬다.
		{"그리스 문자", runeRange(0x0370, 0x03FF)},       // Greek and Coptic
		{"원문자 · 괄호 문자", runeRange(0x2460, 0x24FF)},  // Enclosed Alphanumerics
		{"숫자 꼴 · 로마 숫자", runeRange(0x2150, 0x218F)}, // Number Forms
		{"문자 · 숫자", runeList(curatedGlyphs["문자 · 숫자"])},
		{"둘러싼 한글 · 한자 · 숫자", runeRange(0x3200, 0x32FF)}, // Enclosed CJK Letters and Months
		{"단위 · 날짜 · 조합 문자", runeRange(0x3300, 0x33FF)},  // CJK Compatibility
		{"전각 영숫자 · 기호", slices.Concat(runeRange(0xFF01, 0xFF5E), runeRange(0xFFE0, 0xFFE6))},
		{"기타 기호 · 화살표", runeRange(0x2B00, 0x2BFF)}, // Miscellaneous Symbols and Arrows
		{"기술 기호", runeRange(0x2300, 0x23FF)},       // Miscellaneous Technical
		{"점자 (8점)", runeRange(0x2801, 0x28FF)},
		{"이모지", slices.Concat(
			runeList(curatedGlyphs["이모지"]), // 자주 쓰는 것이 앞이다
			runeRange(0x1F600, 0x1F64F),    // Emoticons
			runeRange(0x1F300, 0x1F5FF),    // Miscellaneous Symbols and Pictographs
			runeRange(0x1F680, 0x1F6FF),    // Transport and Map Symbols
			runeRange(0x1F900, 0x1F9FF),    // Supplemental Symbols and Pictographs
			runeRange(0x1FA70, 0x1FAFF),    // Symbols and Pictographs Extended-A
		)},
	}
	// 한 글자는 한 곳에만 둔다. 먼저 나온 묶음이 갖는다. 거르고 나서 빈 묶음은 뺀다.
	seen := map[string]bool{}
	out := []glyphGroup{}
	for _, group := range groups {
		kept := []string{}
		for _, glyph := range group.glyphs {
			if !seen[glyph] {
				seen[glyph] = true
				kept = append(kept, glyph)
			}
		}
		if len(kept) > 0 {
			out = append(out, glyphGroup{group.title, kept})
		}
	}
	return out
}

// drawable 은 글자표에 올릴 수 있는 글자인지다. 그릴 수 있고 폭이 한 칸이나 두 칸이면 된다. 폭이 0 인
// 것(결합 문자, 이음 문자)은 칸을 차지하지 않아 판에 둘 수 없다.
func drawable(r rune) bool {
	cells := ansi.StringWidth(string(r))
	return unicode.IsGraphic(r) && (cells == 1 || cells == 2)
}

// widthDoubt 는 터미널에 따라 폭이 달라질 수 있는 글자의 까닭이다. 확실하면 "" 다. 판은 x/ansi 가
// 재는 폭을 믿고 그리므로, 까닭이 있는 글자로 맞춘 줄은 다른 터미널에서 밀릴 수 있다.
//
//   - Ambiguous: East Asian Ambiguous 다(█ ▲ ← ★ α). 터미널 설정에 따라 한 칸도 두 칸도 된다.
//     x/ansi 는 한 칸으로 잰다
//   - 라이브러리 불일치: 폭을 재는 두 라이브러리(x/ansi, uniseg)가 다르게 잰다. 괘(☰ ䷀)는 유니코드
//     16 에서 한 칸에서 두 칸으로 바뀌어 여기 걸린다
//   - 이모지 모양: 이모지인데 기본 모양이 글자다(☀ ⚠ ❤ ✌). 터미널에 따라 한 칸 글자로도, 두 칸
//     이모지로도 그린다. VS16(U+FE0F)을 붙였을 때 폭이 바뀌는 것으로 가려낸다
//
// 여러 rune 으로 된 글자는 글자표에 없으므로 재지 않는다.
func widthDoubt(glyph string) string {
	runes := []rune(glyph)
	if len(runes) != 1 {
		return ""
	}
	cells := ansi.StringWidth(glyph)
	switch {
	case width.LookupRune(runes[0]).Kind() == width.EastAsianAmbiguous:
		return "Ambiguous"
	case uniseg.StringWidth(glyph) != cells:
		return "라이브러리 불일치"
	case uniseg.StringWidth(glyph+"\uFE0F") != cells:
		return "이모지 모양"
	}
	return ""
}

// runeRange 는 first 부터 last 까지 중 글자표에 올릴 수 있는 글자다.
func runeRange(first, last rune) []string {
	out := []string{}
	for r := first; r <= last; r++ {
		if drawable(r) {
			out = append(out, string(r))
		}
	}
	return out
}

// runeList 는 runes 를 한 글자씩 끊어 글자표에 올릴 수 있는 것만 남긴다. 여러 rune 으로 된 글자(ZWJ
// 이모지, VS16 이 붙은 것)는 끊기면 조각이 되는데, VS16 · ZWJ 는 폭이 0 이라 빠지고 나머지 조각은 제
// 글자로 남는다.
func runeList(runes string) []string {
	out := []string{}
	for _, r := range runes {
		if drawable(r) {
			out = append(out, string(r))
		}
	}
	return out
}

// glyphRow 는 글자표의 한 줄이다. 묶음 제목 줄이면 glyphs 가 비어 있다.
type glyphRow struct {
	title  string
	glyphs []string
}

// glyphRows 는 글자표의 줄이다. 검색어가 있으면 맞는 글자만 남기고, 남은 것이 없는 묶음은 뺀다.
func (s *sketch) glyphRows() []glyphRow {
	query := s.glyphQuery.Value()
	rows := []glyphRow{}
	for _, group := range glyphGroups() {
		found := []string{}
		for _, glyph := range group.glyphs {
			if glyphMatches(group.title, glyph, query) {
				found = append(found, glyph)
			}
		}
		if len(found) == 0 {
			continue
		}
		rows = append(rows, glyphRow{title: group.title})
		columns := s.glyphColumns()
		for start := 0; start < len(found); start += columns {
			rows = append(rows, glyphRow{glyphs: found[start:min(start+columns, len(found))]})
		}
	}
	return rows
}

// glyphMatches 는 글자가 검색어에 걸리는지다. 검색어를 빈칸으로 끊은 말이 전부 들어 있어야 한다.
//
// 보는 말은 넷이다. 묶음 이름(`선`, `이모지`), zn 에서 옮긴 한국어 이름과 찾는 말(`세모`), 유니코드
// 영문 이름(`BOX DRAWINGS LIGHT DOWN AND RIGHT`), 글자 자신. 크고 작은 글자는 가리지 않는다.
func glyphMatches(title, glyph, query string) bool {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return true
	}
	names := []string{title, glyphNames[glyph], glyph}
	for _, r := range glyph {
		names = append(names, runenames.Name(r))
	}
	text := strings.ToLower(strings.Join(names, " "))
	for _, word := range words {
		if !strings.Contains(text, word) {
			return false
		}
	}
	return true
}

// glyphRowsShown 은 글자표가 한 번에 보이는 줄 수다. 창 전체(테두리 둘과 안내 · 검색 두 줄 포함)가
// 터미널 높이의 60% 다. 글자가 오천 자를 넘어 어차피 굴려야 하므로, 화면을 다 덮어 판을 가리는 것보다
// 낫다. 띠까지 친 터미널 높이로 재지만 60% 라 띠를 가리지는 않는다.
func (s *sketch) glyphRowsShown() int { return max(s.height*6/10-4, 1) }

// glyphColumns 는 글자표 한 줄의 글자 수다. 창 전체(테두리 포함)가 터미널 폭의 80% 를 넘지 않게
// 칸 폭(glyphCellWidth) 단위로 내림한다. 좁은 터미널에서도 minGlyphColumns 아래로는 안 줄인다.
// 그보다 좁으면 창이 80% 를 넘는다. 안내 줄(glyphTitle)이 그 폭에 맞춰져 있다.
func (s *sketch) glyphColumns() int {
	return max((s.width*8/10-2)/glyphCellWidth, minGlyphColumns)
}

func (s *sketch) scrollGlyphsBy(by int) {
	s.glyphTop = min(max(s.glyphTop+by, 0), max(len(s.glyphRows())-s.glyphRowsShown(), 0))
}

func (s *sketch) scrollGlyphs(mouse tea.Mouse) {
	switch mouse.Button {
	case tea.MouseWheelUp:
		s.scrollGlyphsBy(-3)
	case tea.MouseWheelDown:
		s.scrollGlyphsBy(3)
	}
}

// scrollGlyphKey 는 글자표를 굴리는 키다. 검색 중이든 아니든 같다. 굴리는 키였으면 true 다.
func (s *sketch) scrollGlyphKey(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "up":
		s.scrollGlyphsBy(-1)
	case "down":
		s.scrollGlyphsBy(1)
	case "pgup":
		s.scrollGlyphsBy(-s.glyphRowsShown())
	case "pgdown":
		s.scrollGlyphsBy(s.glyphRowsShown())
	default:
		return false
	}
	return true
}

// glyphKey 는 글자표가 떠 있고 검색 중이 아닐 때의 키다. 방향키가 표를 굴리고, 글자 키는 그 글자를
// 붓으로 삼고 창을 닫는다. 표에 없는 글자(한글, 영문 등)는 이 길로 고른다.
func (s *sketch) glyphKey(msg tea.KeyPressMsg) {
	if s.scrollGlyphKey(msg) {
		return
	}
	if glyph := typedGlyph(msg); glyph != "" {
		s.brush.Glyph = glyph
		s.openPopup(popupNone)
	}
}

// glyphSearchKey 는 검색 줄에 손이 가 있을 때의 키다. 글자 키는 전부 검색어로 가고, Enter 는 걸러진
// 첫 글자를 붓으로 삼는다. Esc 는 검색에서 손을 뗀다(key 가 받는다).
func (s *sketch) glyphSearchKey(msg tea.KeyPressMsg) tea.Cmd {
	if s.scrollGlyphKey(msg) {
		return nil
	}
	if msg.String() == "enter" {
		for _, row := range s.glyphRows() {
			if len(row.glyphs) > 0 {
				s.brush.Glyph = row.glyphs[0]
				s.openPopup(popupNone)
				return nil
			}
		}
		return nil
	}
	cmd := s.glyphQuery.Update(msg)
	// 목록이 바뀌었으니 맨 위부터 보인다. 안 그러면 내려 본 곳이 걸러진 목록 밖이 된다. 마우스가 올라간
	// 칸도 다른 글자가 되었으므로 지운다. 다음 움직임이 다시 잰다.
	s.glyphTop = 0
	s.glyphHoverRow, s.glyphHoverColumn = -1, -1
	return cmd
}

// startGlyphSearch 와 stopGlyphSearch 는 검색 줄에 손을 대고 뗀다. 검색어는 남는다.
func (s *sketch) startGlyphSearch() tea.Cmd {
	s.glyphSearching = true
	return s.glyphQuery.Focus()
}

func (s *sketch) stopGlyphSearch() {
	s.glyphSearching = false
	s.glyphQuery.Blur()
}

// glyphSearchLine 은 안내 아래 검색 줄이다. 누르면 검색이 시작된다.
func (s *sketch) glyphSearchLine() string {
	switch {
	case s.glyphSearching:
		return s.glyphQuery.String()
	case s.glyphQuery.Value() == "":
		return dimStyle.Render("찾기: 여기를 누르거나 Tab (한국어 · 영문 이름, 묶음)")
	}
	return "찾기: " + s.glyphQuery.Value()
}

// glyphBoxSize 는 glyphBox 의 바깥 폭과 높이다. 창을 그리지 않고 잰다. 폭은 한 줄의 칸과 테두리 둘,
// 높이는 안내 · 검색 두 줄과 목록 줄과 테두리 둘이다. glyphBox 가 빈 줄로 채워 이 크기를 지킨다.
func (s *sketch) glyphBoxSize() (width, height int) {
	return s.glyphColumns()*glyphCellWidth + 2, glyphListTop + s.glyphRowsShown() + 2
}

// glyphBox 는 글자표 창이다. 굴리거나 걸러도 창 폭과 높이가 안 바뀌게 빈 줄로 채운다.
func (s *sketch) glyphBox() string {
	s.scrollGlyphsBy(0) // 창 높이가 줄었거나 걸러져 줄이 줄었으면 내려 본 곳을 당긴다
	rows := s.glyphRows()
	inner := s.glyphColumns() * glyphCellWidth
	lines := []string{glyphTitle, ansi.Truncate(s.glyphSearchLine(), inner, "…")}
	if len(rows) == 0 {
		lines = append(lines, dimStyle.Render("  맞는 글자가 없다"))
	}
	for i := s.glyphTop; i < s.glyphTop+s.glyphRowsShown(); i++ {
		if i >= len(rows) {
			if len(rows) > 0 || i > s.glyphTop {
				lines = append(lines, "")
			}
			continue
		}
		if rows[i].glyphs == nil {
			lines = append(lines, dimStyle.Render("── "+rows[i].title))
			continue
		}
		var line strings.Builder
		for column, glyph := range rows[i].glyphs {
			pad := strings.Repeat(" ", glyphCellWidth-ansi.StringWidth(glyph))
			switch _, blank := blankNames[glyph]; {
			case i == s.glyphHoverRow && column == s.glyphHoverColumn:
				// 글자 폭만 칠한다. 뒤의 빈칸까지 칠하면 칸이 한쪽으로 쏠려 보인다.
				line.WriteString(glyphHoverStyle.Render(glyph) + pad)
			case blank:
				line.WriteString(blankSwatch.Render(glyph) + pad)
			case widthDoubt(glyph) != "":
				line.WriteString(doubtSwatch.Render(glyph) + pad)
			default:
				line.WriteString(glyph + pad)
			}
		}
		lines = append(lines, line.String())
	}
	return popupStyle.Width(inner + 2).Render(strings.Join(lines, "\n"))
}

// glyphHoverStyle 은 마우스가 올라간 글자 칸이다. 누르면 무엇이 붓이 될지 누르기 전에 보인다. 빈칸
// 바탕(blankSwatch, 238)보다 밝아야 빈칸 위에 올라가도 구별된다.
var glyphHoverStyle = lipgloss.NewStyle().Background(lipgloss.ANSIColor(244))

// glyphAt 은 화면 칸 아래의 글자표 칸이다. 줄은 걸러진 목록(glyphRows)의 번호이고 칸은 그 줄 안의
// 번호다. 글자 칸이 아니면(제목 · 검색 줄, 묶음 제목, 빈 곳, 창 밖) ok 가 false 다.
//
// 창을 그리지 않고 크기(glyphBoxSize)로 자리를 잰다. 휠 · 마우스 움직임마다 불리는데, 창을 그려 재면
// 화면을 그릴 때와 합쳐 한 번에 두 번 그리게 된다.
func (s *sketch) glyphAt(mouse tea.Mouse) (row, column int, ok bool) {
	s.scrollGlyphsBy(0) // glyphBox 가 그리기 전에 하는 것과 같다. 줄이 줄었으면 내려 본 곳을 당긴다
	rows := s.glyphRows()
	originX, originY := s.centerOrigin(s.glyphBoxSize())
	x, y := mouse.X-originX-1, mouse.Y-originY-1
	row, column = s.glyphTop+y-glyphListTop, x/glyphCellWidth
	if y < glyphListTop || y >= glyphListTop+s.glyphRowsShown() || x < 0 || row >= len(rows) || column >= len(rows[row].glyphs) {
		return 0, 0, false
	}
	return row, column, true
}

// hoverGlyph 는 마우스 아래의 글자표 칸을 적어 둔다. 움직일 때와 굴릴 때(칸 밑의 글자가 바뀐다) 부른다.
func (s *sketch) hoverGlyph(mouse tea.Mouse) {
	s.glyphHoverRow, s.glyphHoverColumn = -1, -1
	if row, column, ok := s.glyphAt(mouse); ok {
		s.glyphHoverRow, s.glyphHoverColumn = row, column
	}
}

// glyphListTop 은 창 안쪽에서 글자 목록이 시작하는 줄이다. 안내와 검색 줄 아래다.
const glyphListTop = 2

// pickGlyph 는 글자표의 누름이다. 검색 줄을 누르면 검색이 시작되고, 글자를 누르면 붓이 된다. 창 바깥을
// 누르면 닫힌다.
func (s *sketch) pickGlyph(mouse tea.Mouse) tea.Cmd {
	// 창 바깥을 누르면 닫는다. 그 누름은 판에 닿지 않는다. 색표와 같다.
	box := s.glyphBox()
	if s.outsidePopup(box, mouse) {
		s.openPopup(popupNone)
		return nil
	}
	if _, y := s.insidePopup(box, mouse); y == glyphListTop-1 {
		return s.startGlyphSearch()
	}
	if row, column, ok := s.glyphAt(mouse); ok {
		s.brush.Glyph = s.glyphRows()[row].glyphs[column]
		s.openPopup(popupNone)
	}
	return nil
}

// zn(~/src/bluemir/zn) 의 특수문자 표(internal/assets/symbols.go 의 CuratedSymbols)에서 가져온 글자다.
// 갈래도 그쪽을 따르되 이모지 여섯 갈래는 하나로 묶었다. 폭이 0 인 조각은 여기 남아 있어도
// glyphGroups 가 거른다(drawable).
var curatedGlyphs = map[string]string{
	"화살표":     "←→↑↓↔↕↖↗↘↙↚↛↝↞↠↢↣↦↩↪↰↱↲↳↵↶↷↺↻↼⇀⇄⇅⇆⇇⇉⇐⇑⇒⇓⇔⇕⇠⇡⇢⇣⇦⇧⇨⇩⇥⇤⌁⌃⌄⌘⌥⎋⏎⏏⌫⌦➔➜➡⬅⬆⬇⤴⤵⟵⟶⟷⟸⟹⟺↯⇋⇌",
	"블록 · 도형": "▲△▼▽◀◁▶▷▴▵▾▿◂◃▸▹◤◥◣◢⊿■□▪▫▬▭▮▯◆◇◈◊●○◎◉◌◯⬤◐◑◒◓⬛⬜⯀★☆✦✧✩✪✫✮✯❂✱✲❉❋⬟⬠⬢⬣⯃▰▱◫◪▩▨▤▥▦░▒▓█▄▀▌▐▁▂▃▅▆▇",
	"수학":      "±∓×÷∕∗∘∙⋅≠≈≅≡≢≒≤≥≪≫∞∝∂∇∫∬∮∑∏√∛∜∀∃∄∅∈∉∋⊂⊃⊆⊇⊄∪∩∖∧∨¬⊕⊗⊥∥∠∟°′″∴∵∼≃≜≝≟⌈⌉⌊⌋⟨⟩ℝℕℤℚℂℵℓ℘ℏ∎⊤⊢⊨⋮⋯⋱≺≻⊑⌀∆µΩÅ‰‱%",
	"구두점":     "…‥–—―‐‑·•◦‣⁃‧†‡§¶‖¦“”‘’„‚«»‹›¡¿‽⁇⁈⁉‼※⁂⌗№℗©®™℠＿〜～〰ᵎ〽",
	"괄호":      "「」『』〈〉《》【】〔〕〖〗（）［］｛｝⦗⦘⟦⟧⟪⟫⌜⌝⌞⌟",
	"통화":      "₩$¢€£¥元₽₹₫₺₴₦₱฿₪₸₡₲₵₭₮₾¤₿",
	"기호":      "✓✔☑☐☒✗✘✕✖✚✛⚠⚡☠☢☣♻⚙⚛⚖⚓⚔⚑⚐⚕☮☯☰☷♠♥♦♣♤♡♢♧☀☁☂☃❄☾☽♀♂⚥♪♫♬♩♭♯♮☎✉✂✈✏✒☞☜☝☟✍⌚⌛⏰⏳⏱⏲⚀⚁⚂⚃⚄⚅☕♨⚘✿❀❁☘⚜〄♿⚟⏩⏪⏸⏹⏺⏯♲✳✴❇✨❖➤➢⟡⌂⏻⏼⭘",
	"문자 · 숫자": "αβγδεζηθικλμνξπρστυφχψωΓΔΘΛΞΠΣΦΨ⁰¹²³⁴⁵⁶⁷⁸⁹⁺⁻ⁿ₀₁₂₃₄₅₆₇₈₉₊₋½⅓⅔¼¾⅕⅛⅜⅚⅞ⅠⅡⅢⅣⅤⅥⅦⅧⅨⅩ①②③④⑤⑥⑦⑧⑨⑩⑪⑫⑬⑭⑮⑯⑰⑱⑲⑳❶❷❸❹❺❻❼❽❾❿⑴⑵⑶⑷⑸⑹⑺⑻⑼⑽㎜㎝㎞㎡㎥㎏㎎㎖㏄㎈℃℉㎐㎅㎆㎇㎾㏈㎳㎲㉠㉡㉢㉣㉤㉥㉦㉧㉨㉩㉪㉫㉬㉭㉮㉯㉰㉱㉲㉳㉴㉵㉶㉷㉸㉹㉺㉻㈀㈁㈂㈃㈄㈅㈆㈇㈈㈉㈊㈋㈌㈍㈎㈏㈐㈑㈒㈓㈔㈕㈖㈗㈘㈙㈚㈛",
	"이모지":     "😀😃😄😁😆😅🤣😂🙂🙃😉😊😇🥰😍🤩😘😗😚😙😋😛😜🤪😝🤑🤗🤭🤫🤔🤐🤨😐😑😶😏😒🙄😬🤥😌😔😪🤤😴😷🤒🤕🤢🤮🤧🥵🥶😵🤯🤠🥳😎🤓🧐😕😟🙁😮😯😲😳🥺😦😨😰😥😢😭😱😖😣😞😓😩😫🥱😤😡😠🤬😈👿💀💩🤡👻👽🤖😺😻😹🙀😿🙈🙉🙊👍👎👌✌🤞🤟🤘🤙👈👉👆👇✋🤚🖐🖖👋🤏✊👊🤛🤜👏🙌👐🤲🤝🙏💅💪👀👁👂👃👄👅🦷🧠🦴👶🧒👦👧🧑👨👩🧓👴👵🙋🙇🤦🤷💁🙅🙆🚶🏃💃🕺👮👷💂🕵👨‍💻👩‍💻🧑‍🍳🧑‍🎓🧑‍🏫🧑‍⚕🎅🦸🧙🧚👯👪👫💏💑👤👥🗣🦶🦵❤🧡💛💚💙💜🖤🤍🤎💔❣💕💞💓💗💖💘💝💟💌💋💯💢💥💫💦💨🕳💣💬💭🗯💤🔥⭐🌟✅❌❎➕➖➗❓❔❗❕🚫⛔💡🔔🔕📌📍🔖🏷🔗📎🔒🔓🔑🔍🔎🔄🔁🔀🆗🆕🆙🔝🎉🎊🎈🎁🏆🥇🥈🥉🎯🚀💎👑🐶🐕🐱🐈🐭🐹🐰🦊🐻🐼🐨🐯🦁🐮🐷🐽🐸🐵🐒🐔🐧🐦🐤🦆🦅🦉🦇🐺🐗🐴🦄🐝🐛🦋🐌🐞🐜🕷🕸🦂🐢🐍🦎🐙🦑🦐🦀🐡🐠🐟🐬🐳🦈🐊🐘🦏🐪🦒🐇🐿🦔🐾🌸💮🌹🥀🌺🌻🌼🌷🌱🌲🌳🌴🌵🍀🍁🍂🍃🌾🌍🌏🌎🌕🌑🌙🌛🌞⛅🌧⛈🌩⛄🌈🌊💧🌋⛰🏔🌅🌄🌇🌌🍎🍏🍐🍊🍋🍌🍉🍇🍓🍑🍒🥝🍍🥥🥑🍅🍆🥕🌽🌶🥔🍠🧄🧅🥦🍄🥜🌰🍞🥐🥖🥨🧀🥚🍳🥞🥓🍔🍟🍕🌭🥪🌮🌯🥗🍝🍜🍲🍛🍚🍙🍘🍢🍡🍣🍤🍥🥟🍦🍨🍧🍩🍪🎂🍰🧁🍫🍬🍭🍮🍯🥛🍵🍶🍺🍻🥂🍷🥃🍸🍹🍾🧊🥄🍴🍽🥢💻🖥🖨⌨🖱💾💿📀💽📱📲📞📟📠🔋🔌🔦📷📸📹🎥📺📻🎙🎤🎧🎵🎶🎼🎹🎸🥁🎺🎻📖📚📕📓📔📒📃📄📑📊📈📉📋📁📂🗂🗃🗄🗑🖊🖋🖌🖍📝📐📏🔨🔧🔩🧰🪛⛏🧲🧪🔬🔭💉💊🩹🛏🚪🪟🧹🧺💰💵💳🧾📦📮📧📨📩🔐🛡⚽🏀⚾🎾🏐🏈🎱🏓🏸🥊🎮🕹🎲🧩♟🎬🎭🎨🚗🚕🚌🚑🚒🚲🛴🏍🚂🚆🚇🚁🚢⛵🚦🏠🏢🏫🏥🏦🏪⛺🗺🧭🕐📅📆",
}
