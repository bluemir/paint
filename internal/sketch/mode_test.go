package sketch

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// 도구 키는 왼손이 닿는 키 안에 있고, 서로 또 다른 한 글자 키(스포이드 · 창 · 굴림)와 겹치지 않는다.
// 팔레트 이름도 겹치지 않는다. 새 도구를 modes 에 올릴 때 이것이 지킨다.
func TestModeShortcutsAndCommandsAreUnique(t *testing.T) {
	const leftHand = "qwertasdfgzxcvb"
	taken := map[string]string{"q": "스포이드", "c": "색표", "v": "글자표", "w": "굴림", "a": "굴림", "s": "굴림", "d": "굴림"}
	for _, m := range modes {
		key := m.key
		if len(key) != 1 || !strings.Contains(leftHand, key) {
			t.Errorf("%s: 키 %q 가 왼손 영역(%s) 밖이다", m.name, key, leftHand)
		}
		if other, ok := taken[key]; ok {
			t.Errorf("%s: 키 %q 를 %s 가 쓴다", m.name, key, other)
		}
		taken[key] = m.name
	}
	names := map[string]bool{}
	for _, c := range commands {
		if names[c.name] {
			t.Errorf("팔레트 명령 %q 가 둘이다", c.name)
		}
		names[c.name] = true
	}
}

// 창은 연 모드 화면으로 돌아간다. 글자 모드에서 띠의 색을 눌러 색표를 열고 닫으면 글자 모드이고, 그
// 글자 모드의 Esc 는 여전히 들어오기 전 도구로 간다.
func TestPopupReturnsToOpeningMode(t *testing.T) {
	s := newTestSketch(80, 5)
	s.setMode(modePaint)
	s.setMode(modeText)
	line := ansi.Strip(s.stripLine())
	x := ansi.StringWidth(line[:strings.Index(line, "전경")]) + 1
	if next := send(s, tea.MouseClickMsg{X: x, Y: s.height - 1, Button: tea.MouseLeft}); typeName(next) != "*sketch.viewColor" {
		t.Fatalf("전경을 눌렀는데 화면 = %T", next)
	}
	if send(s, escape); s.mode() != modeText {
		t.Fatalf("색표를 닫은 뒤 모드 = %s, 글자여야 한다", s.mode())
	}
	if send(s, escape); s.mode() != modePaint {
		t.Errorf("글자 모드 Esc 뒤 모드 = %s, 칠하기여야 한다", s.mode())
	}
}
