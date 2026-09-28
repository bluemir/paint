// 입력기(IME) 판정. 도구 키가 전부 한 글자 ASCII 라 입력기가 켜져 있으면 그것들을 삼켜 조합 글자로
// 바꿔 버린다. 키가 안 먹는 까닭이 안 보이므로 띠에 적는다(sketch.imeOn).

package sketch

import (
	"unicode"

	tea "charm.land/bubbletea/v2"
)

// imeText 는 그 입력이 입력기를 거쳐 온 글자인지다.
//
// 도구 키가 전부 ASCII 라는 것이 판정의 근거다. 인쇄 가능한 non-ASCII 글자가 왔다는 것은 곧
// 입력기가 켜져 있다는 뜻이다. 한글만 보지 않는 이유가 그것이고, 일본어·중국어 입력기도 같은
// 문제라 한 가지 규칙으로 다 잡힌다.
//
// Text 를 본다. Code 는 조합 전 키가 아니라 받은 글자라(bubbletea 는 kitty 프로토콜 없이는 원래
// 키를 모른다) 그쪽으로는 갈릴 것이 없다.
func imeText(key tea.KeyPressMsg) bool {
	for _, r := range key.Text {
		if r > unicode.MaxASCII && unicode.IsPrint(r) {
			return true
		}
	}
	return false
}

// asciiLetter 는 그 입력이 입력기를 끈 증거인지다. 라틴 글자가 그대로 왔으면 껐다는 뜻이다.
//
// 방향키·enter 는 증거가 아니다. 입력기가 삼키지 않으므로 켜진 채로도 그대로 온다. 숫자·기호도
// 아니다. 한글 입력기는 `.` 같은 기호를 그대로 흘려보내므로 그것이 왔다고 껐다고 할 수 없다.
func asciiLetter(key tea.KeyPressMsg) bool {
	runes := []rune(key.Text)
	if len(runes) != 1 {
		return false
	}
	r := runes[0]
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')
}
