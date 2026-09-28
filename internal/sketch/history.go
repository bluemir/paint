// 되돌리기다. 한 번의 손길(누름부터 뗌까지, 글자 모드의 키 하나)이 바꾼 칸을 전 · 후 값으로 적어 두고,
// 되돌리면 전 값을, 다시 하면 후 값을 판에 나른다.
//
// 명령(edit)은 도구마다 따로 두지 않는다. 칠하기 · 색칠 · 지우기 · 박스 · 직선 · 글자가 모두 "칸을
// 바꾼다" 는 한 가지 일이라, 바뀐 칸만 들면 무엇으로 바꿨는지는 몰라도 된다. 도구가 늘어도 이 파일은
// 그대로다.
//
// 바뀐 칸은 손길을 시작할 때 떠 둔 판의 사본과 끝난 뒤의 판을 견주어 얻는다. 판을 바꾸는 곳마다 기록을
// 끼워 넣으면 넓은 글자가 옆 칸을 건드리는 것까지 따라가야 한다. 견주면 그런 것이 저절로 잡힌다.
// 120x40 판이 칸 4800 개라 손길마다 한 번 떠도 무겁지 않다. (ADR-0001 §4)

package sketch

// historyLimit 는 되돌릴 수 있는 손길의 수다. 넘으면 가장 오래된 것부터 버린다.
const historyLimit = 100

// cellChange 는 칸 하나가 어떻게 바뀌었는지다.
type cellChange struct {
	x, y          int
	before, after Cell
}

// edit 은 손길 하나가 바꾼 칸들이다. 되돌리기의 명령 하나다.
type edit []cellChange

type history struct {
	done   []edit
	undone []edit
	// pending 은 손길을 시작할 때 떠 둔 판이다. 손길 중이 아니면 nil 이다.
	pending *Canvas
}

// beginEdit 은 판을 바꿀 손길을 시작한다. 앞 손길이 끝나지 않았으면(뗌을 못 받았으면) 그것을 먼저 닫는다.
func (s *sketch) beginEdit() {
	s.endEdit()
	s.history.pending = s.canvas.clone()
}

// endEdit 은 손길을 닫고 바뀐 칸을 되돌리기 목록에 올린다. 바뀐 것이 없으면 아무것도 안 올린다.
// 새 손길이 오르면 다시 하기 목록은 버린다. 갈라진 앞날은 되살릴 수 없다.
func (s *sketch) endEdit() {
	before := s.history.pending
	s.history.pending = nil
	if before == nil {
		return
	}
	changes := edit{}
	for y := range s.canvas.Height {
		for x := range s.canvas.Width {
			if old, now := before.At(x, y), s.canvas.At(x, y); old != now {
				changes = append(changes, cellChange{x: x, y: y, before: old, after: now})
			}
		}
	}
	if len(changes) == 0 {
		return
	}
	s.history.done = append(s.history.done, changes)
	if len(s.history.done) > historyLimit {
		s.history.done = s.history.done[len(s.history.done)-historyLimit:]
	}
	s.history.undone = nil
}

// undo 와 redo 는 손길 하나를 되돌리고 다시 한다. 넓은 글자의 두 칸은 둘 다 바뀐 칸으로 적혀 있으므로
// 칸을 그대로 덮어도 짝이 맞는다.
func (s *sketch) undo() {
	s.endEdit()
	last := len(s.history.done) - 1
	if last < 0 {
		s.message = "되돌릴 것이 없다"
		return
	}
	changes := s.history.done[last]
	s.history.done = s.history.done[:last]
	for _, change := range changes {
		s.canvas.cells[change.y][change.x] = change.before
	}
	s.history.undone = append(s.history.undone, changes)
	s.dirty = true
}

func (s *sketch) redo() {
	s.endEdit()
	last := len(s.history.undone) - 1
	if last < 0 {
		s.message = "다시 할 것이 없다"
		return
	}
	changes := s.history.undone[last]
	s.history.undone = s.history.undone[:last]
	for _, change := range changes {
		s.canvas.cells[change.y][change.x] = change.after
	}
	s.history.done = append(s.history.done, changes)
	s.dirty = true
}
