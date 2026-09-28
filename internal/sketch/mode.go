// 도구(모드) 목록이다. 모드마다 제 화면이 있고(viewBrush · viewText · viewPaint · viewErase), 판(sketch)을
// 품는다. 지금 어느
// 모드인지는 필드가 아니라 떠 있는 화면이다. 모드를 바꾸는 것은 새 화면을 돌려주는 것이다. (ADR-0006)
//
// 도구 줄 · 명령 팔레트 · 단축키가 이 목록을 훑는다. 새 도구는 여기 한 줄을 올리면 세 곳에 다 나온다.

package sketch

import (
	tea "charm.land/bubbletea/v2"
)

// modeEntry 는 도구 하나다. open 은 그 모드의 화면을 세운다. from 은 지금 떠 있는 모드 화면이다. 글자
// 모드가 Esc 로 돌아갈 곳이다.
type modeEntry struct {
	name string // 도구 줄과 띠에 보이는 이름
	// key 는 도구를 드는 한 글자 키다. 왼손이 닿는 키(qwert asdfg zxcvb)에서 고른다.
	key string
	// command 와 desc 는 명령 팔레트에 오르는 이름과 설명이다.
	command, desc string
	open          func(s *sketch, from tea.Model) (tea.Model, tea.Cmd)
}

// modes 는 도구 목록이다. 목록 차례가 곧 도구 줄과 팔레트에 뜨는 차례다.
var modes = []modeEntry{
	{name: brushModeName, key: "b", command: "brush", desc: "브러시 모드로", open: openBrush},
	{name: textModeName, key: "t", command: "text", desc: "글자 모드로", open: openText},
	{name: paintModeName, key: "r", command: "paint", desc: "칠하기 모드로 (글자는 두고 색만)", open: openPaint},
	{name: eraseModeName, key: "e", command: "erase", desc: "지우기 모드로", open: openErase},
}
