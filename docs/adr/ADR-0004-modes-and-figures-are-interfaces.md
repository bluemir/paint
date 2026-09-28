# ADR-0004: 도구와 모양을 interface 로 둔다

- 상태: 채택 (Accepted)
- 날짜: 2026-09-29

## 배경 (Context)

도구(mode)와 모양(figure)은 `int` 상수였다. 도구마다 다른 일이 여러 파일의 switch 와 `mode == modeText`
분기에 흩어져 있었다.

- 이름(`mode.String`), 띠 안내(`toolLabel`), 건너 쓸 폭(`toolStep`), 칸 하나에 하는 일(`apply`)
- 판 누름 · 끌기(`click` · `motion`), 키(`key`), 커서(`textCursor`), Esc 로 돌아갈 곳(`setMode`)
- 단축키는 `toolKey` 의 switch 에, 도구 줄 키는 `tools` 에, 팔레트 이름은 `commands` 에 따로 적혀 있었다.
  도구 이름과 키를 바꿀 때(ADR-0001 §4) 세 곳을 함께 고쳐야 했다.

바라는 것은 셋이다: 도구 · 모양을 더하기 쉽게, 흩어진 분기를 도구 한 곳에 모으기, 키 · 이름을 한 곳에.

AGENTS.md 는 "interface 는 반드시 필요하기 전에는 도입하지 않는다"고 한다. 함수 필드를 가진 구조체 표로도
같은 것을 할 수 있다(도구 줄의 `toolbarRow` 가 그 꼴이다).

## 결정 (Decision)

### 1. interface 로 둔다

`mode` 와 `figure` 를 interface 로, 도구 · 모양마다 타입 하나씩 둔다. 함수 필드 표가 아니라 interface 인 것은
보는 사람이 골랐다. 도구가 해야 할 일이 메서드 목록으로 한 곳에 보이고, 빠뜨리면 컴파일이 막는다. AGENTS.md
규칙의 예외로 이 둘을 둔다.

- `mode`(mode.go): `String` · `shortcut` · `command` · `desc` · `label` · `press` · `move` · `keyPress` ·
  `cursor` · `apply` · `step`
- `figure`(shape.go): `String` · `command` · `hint` · `press` · `move` · `points`

### 2. 글자 모드도 전략으로 옮긴다

글자 모드는 키로 적고, 커서를 두고, 모양을 안 쓴다. 이것까지 `textMode` 의 메서드로 옮겨 `mode == modeText`
분기를 없앴다. 대가로 `textMode` 에는 쓰이지 않는 `apply` · `step` 이 있고, 다른 세 도구에는 하는 일이 없는
`cursor` 가 있다.

### 3. 같은 일은 품어서 나눈다

- 브러시 · 칠하기 · 지우기는 `figureMode` 를 품는다. 이름 · 키 · 안내 같은 값과, 누름 · 끌기를 지금 모양에
  넘기는 일을 함께 쓴다. 각자는 `apply`(와 브러시의 `step`)만 둔다.
- 직선 · 테두리 · 채움은 `dragFigure` 를 품는다. 끌기 시작 · 끝 옮기기 · 우클릭 취소를 함께 쓰고 각자는
  `points` 만 둔다.

### 4. 목록 하나에서 다 나온다

`modes` 와 `figures` 가 곧 차례다. 도구 줄 · 명령 팔레트 · 단축키(`toolKey`) · Tab 순환이 이 목록을 훑는다.
새 도구는 타입 하나를 두고 `modes` 에 올리면 된다. 도구 키가 왼손 영역(qwert asdfg zxcvb) 안에 있고
다른 한 글자 키와 겹치지 않는지, 팔레트 이름이 겹치지 않는지를 시험이 지킨다.

### 5. Esc 가 돌아갈 곳은 "직전 도구"다

전에는 글자 모드로 들어갈 때만 돌아갈 도구를 적었다(`next == modeText`). 이제 도구가 바뀔 때마다 직전 도구를
적고(`previousMode`), 글자 모드의 Esc 가 그리로 간다. 글자 모드에서 곧장 글자 모드로 가는 일이 없으므로 보이는
동작은 같다.

## 결과 (Consequences)

- 도구 · 모양별 동작이 타입 하나에 모인다. `sketch.go` 의 click · motion · key · View 는 `s.mode` 에 넘기기만 한다.
- 도구 이름과 키는 도구 타입의 값 한 곳에만 있다.
- interface 메서드가 늘면 모든 도구가 따라 구현해야 한다. 쓰이지 않는 빈 구현(`textMode.apply` 등)이 생긴다.
- AGENTS.md 의 interface 규칙에 이 ADR 이 예외를 둔다. 규칙 자체는 바꾸지 않았다.
