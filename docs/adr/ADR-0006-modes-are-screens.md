# ADR-0006: 모드마다 화면이다

- 상태: 채택 (Accepted)
- 날짜: 2026-09-29
- 대체: ADR-0004 의 도구(`mode` interface) 부분 전부와 모양의 `press` · `move` · `hint`. ADR-0005 의
  `under *sketch`
- 참고: zn 의 `viewEditorNormal{*editor}` · `viewEditorInsert{*editor}`

## 배경 (Context)

ADR-0004 는 도구를 `mode` interface 로, 도구마다 타입 하나로 두었다. 판(sketch)이 `s.mode` 를 들고 입력을
그리로 넘겼다. 그 뒤 ADR-0005 가 창을 "under 를 든 화면" 으로 세우고 보니, 모드도 같은 모양이었다. 지금
방향키 · 클릭이 무슨 뜻인지를 갈아 끼우는 것이 모드이고, 그것은 곧 화면이다(redzone ADR-0114 의 말).

zn 이 이렇게 짜여 있다. 공유 상태는 `editor` 하나이고 모드마다 `viewEditorNormal{*editor}` 같은 화면이
있다. 모드를 바꾸는 것은 `insertMode(e)` 가 새 화면을 돌려주는 것이다. 지금 모드를 필드로 두지 않는다.

## 결정 (Decision)

### 1. 모드마다 화면 하나다

`viewBrush` · `viewText` · `viewPaint` · `viewErase` 가 저마다 `tea.Model` 이고 `*sketch` 를 품는다.
`s.mode` 필드와 `mode` interface 는 없앴다. 지금 어느 모드인지는 떠 있는 화면이다.

- 모드를 여는 것은 `openBrush` 같은 함수다. 새 화면을 돌려준다. 모드 목록(`modes []modeEntry`)이 이름 ·
  키 · 팔레트 이름 · 여는 함수를 들고, 도구 줄 · 팔레트 · 단축키가 이것을 훑는다.
- 그리는 것은 화면이 `s.render(canvas, 도구 이름, 띠 토막, 커서)` 를 부르는 것이다. 도구 줄에서 뒤집어 보일
  도구와 띠의 안내를 화면이 넘긴다. 판이 지금 모드를 물을 곳이 없다.
- 판 밖의 일(띠 · 도구 줄 누름, 도구 키, ctrl 키)은 지금 화면을 `from` 으로 받는다. 대개 from 을 돌려주고,
  모드를 바꾸면 새 화면을, 창을 열면 창을 돌려준다.

### 2. 브러시 · 칠하기 · 지우기는 화면 셋이고, 같은 일도 각자 적는다

세 도구는 칸 하나에 하는 일 말고는 누름 · 끌기 · 뗌 · 미리 보기 · 띠 안내가 같다. 처음에는 화면 하나
(`viewDraw`)에 도구를 값으로 넘겼고, 다음에는 셋으로 나누되 같은 일을 판의 메서드로 모아 칸 하나에 하는
일을 함수로 넘겼다. 보는 사람이 둘 다 거두었다: 모드마다 화면이 하나씩 있어야 하고, 억지로 나눠 갖지 말고
펼쳐 놓으라고 했다.

그래서 세 화면이 Update · View · label · stroke · drawDrag 를 저마다 적는다. 비슷한 코드가 세 번 있다.
다른 곳은 칸 하나에 하는 일(`Put` · `Recolor` · `Erase`)과 건너 쓸 폭(브러시만 붓 폭), 띠 안내뿐이다.
한 도구만 달라져야 할 때(예: 칠하기에만 채움 규칙을 더함) 그 화면만 고치면 된다.

### 3. 모양은 어느 칸들인지만 안다

`figure` interface 에는 이름(`String` · `command`)과 덮는 칸(`points`)만 남았다. 누름 · 움직임에 무엇을
하는지는 그리는 도구의 화면이 `figure == figureDot` 으로 갈라 직접 한다. ADR-0004 의 `press` · `move` ·
`hint` 는 도구의 일을 모양에 넘기느라 있던 것이라 같이 없앴다.

끄는 중인 모양(`dragging`)은 판에 둔다. 세 화면이 함께 쓰고, 모드를 열 때(`openBrush` 등)와 모양을 돌릴
때(`nextFigure`) 비운다.

### 4. 글자 모드는 돌아갈 화면을 든다

`viewText{*sketch, back}` 이다. Esc 가 `back` 을 돌려준다. ADR-0004 의 `previousMode` 필드가 이것이 되었다.
이미 글자 모드에서 다시 글자 모드를 고르면(팔레트) 그대로 둔다. 새로 세우면 돌아갈 곳이 글자 모드 자신이
된다.

커서(cursorX · cursorY · lineStart)는 글자 모드 안에서만 사는 상태라 `viewText` 가 갖는다. 글자 모드만 쓰는
`textKey` · `moveCursor` 도 view-text.go 에 있다. 처음에는 "나갔다 들어와도 치던 곳에서 잇는다" 로 판에
두었는데, 들어올 때마다 화면을 새로 세우므로 판에 두면 모드의 상태가 판에 새어 나온다. 대신 들어올 때
마우스가 짚은 칸에서 시작한다(판 밖이면 맨 앞). 키(t)로 들어와 바로 치면 마우스 밑에 적힌다.

글자 모드는 모양을 안 쓰므로 Tab 이 하는 일이 없다. 전에는 글자 모드에서도 모양을 돌렸는데 띠에 모양이
안 보여 눌러도 아무 일이 없는 것처럼 보였다.

### 5. 창은 연 모드 화면으로 돌아간다

창(viewColor · viewGlyph · viewCommand)은 `*sketch` 를 품고 `under tea.Model` 로 연 모드 화면을 든다. 닫으면
under 를, 그릴 때는 `under.View()` 위에 상자를 얹는다(`overlay`). zn 은 창을 닫으면 늘 normal 로 가지만,
여기서는 보는 사람이 "연 모드" 를 골랐다. 칠하기 중에 색을 바꾸고 닫으면 칠하기다. ctrl 키는 창이
under 에게 넘긴다(`keyOver`).

## 결과 (Consequences)

- `s.mode` · `mode` interface · 모양의 press/move/hint · `stamp` 같은 나눠 쓰기가 없다. 모드 하나의 동작은
  그 화면 파일 하나를 읽으면 다 보인다.
- 그리는 도구 셋의 Update 가 거의 같다. 셋에 같은 것을 고쳐야 할 때가 생긴다. 셋이 늘 같이 바뀌는 것이
  보이면 그때 모은다.
- 새 모드는 화면 하나와 modes 의 한 줄이다.
- 시험은 판과 지금 화면을 함께 든 `testSketch` 로 돈다. 프로그램처럼 돌려받은 화면으로 바꾼다.
