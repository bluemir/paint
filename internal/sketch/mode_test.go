package sketch

import (
	"strings"
	"testing"
)

// 도구 키는 왼손이 닿는 키 안에 있고, 서로 또 다른 한 글자 키(스포이드 · 창 · 굴림)와 겹치지 않는다.
// 팔레트 이름도 겹치지 않는다. 새 도구를 modes 에 올릴 때 이것이 지킨다.
func TestModeShortcutsAndCommandsAreUnique(t *testing.T) {
	const leftHand = "qwertasdfgzxcvb"
	taken := map[string]string{"q": "스포이드", "c": "색표", "v": "글자표", "w": "굴림", "a": "굴림", "s": "굴림", "d": "굴림"}
	for _, m := range modes {
		key := m.shortcut()
		if len(key) != 1 || !strings.Contains(leftHand, key) {
			t.Errorf("%s: 키 %q 가 왼손 영역(%s) 밖이다", m, key, leftHand)
		}
		if other, ok := taken[key]; ok {
			t.Errorf("%s: 키 %q 를 %s 가 쓴다", m, key, other)
		}
		taken[key] = m.String()
	}
	names := map[string]bool{}
	for _, c := range commands {
		if names[c.name] {
			t.Errorf("팔레트 명령 %q 가 둘이다", c.name)
		}
		names[c.name] = true
	}
}
