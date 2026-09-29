package sketch

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

var quitKey = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}

// openQuitDialog 는 판에 한 칸 칠하고 ctrl+c 로 끝내기 확인 창을 띄운다. mode 는 창을 띄울 도구다.
func openQuitDialog(t *testing.T, s *testSketch, mode string) *viewQuit {
	t.Helper()
	s.Update(canvasClick(s, 0, 0, tea.MouseLeft))
	s.setMode(mode)
	dialog, ok := send(s, quitKey).(*viewQuit)
	if !ok {
		t.Fatalf("저장 안 했는데 ctrl+c 에 확인 창이 안 떴다: %T", s.screen)
	}
	return dialog
}

// 버튼마다 키가 있다. 저장 후 끝은 저장하고 끝내고, 버리고 끝은 저장 없이 끝내고, Esc 는 창만 닫는다.
func TestQuitDialogKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mock.json")
	for _, tc := range []struct {
		key        tea.KeyPressMsg
		quits      bool
		wantSaved  bool
		wantScreen string
	}{
		{typed("s"), true, true, ""},
		{typed("d"), true, false, ""},
		{escape, false, false, modeBrush},
		{quitKey, true, false, ""}, // 두 번째 ctrl+c 는 버리고 끝낸다
	} {
		s := newTestScreen(path, NewCanvas(10, 5))
		s.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
		openQuitDialog(t, s, modeBrush)
		_, cmd := s.Update(tc.key)
		if quits := cmd != nil; quits != tc.quits {
			t.Errorf("%q: 끝냄 %v, %v 여야 한다", tc.key.String(), quits, tc.quits)
		}
		if saved := !s.dirty; saved != tc.wantSaved {
			t.Errorf("%q: 저장됨 %v, %v 여야 한다", tc.key.String(), saved, tc.wantSaved)
		}
		if tc.wantScreen != "" && s.mode() != tc.wantScreen {
			t.Errorf("%q: 모드 %s, %s 여야 한다", tc.key.String(), s.mode(), tc.wantScreen)
		}
	}
}

// 방향키로 버튼을 옮기고 Enter 로 고른다. 처음 짚은 것은 저장 후 끝이다.
func TestQuitDialogArrowsAndEnter(t *testing.T) {
	s := newTestSketch(10, 5)
	dialog := openQuitDialog(t, s, modeBrush)
	if dialog.cursor != quitSave {
		t.Fatalf("처음 짚은 버튼 = %d, 저장 후 끝이어야 한다", dialog.cursor)
	}
	send(s, tea.KeyPressMsg{Code: tea.KeyRight}, tea.KeyPressMsg{Code: tea.KeyRight}, tea.KeyPressMsg{Code: tea.KeyRight})
	if dialog.cursor != quitCancel {
		t.Fatalf("오른쪽 셋 뒤 버튼 = %d, 끝(취소)에 멈춰야 한다", dialog.cursor)
	}
	send(s, tea.KeyPressMsg{Code: tea.KeyLeft})
	if _, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd == nil || !s.dirty {
		t.Errorf("버리고 끝에서 Enter: 끝냄 %v, 수정됨 %v", cmd != nil, s.dirty)
	}
}

// 버튼을 누르면 고르고, 창 바깥을 누르면 취소다. 취소하면 창을 연 모드로 돌아간다.
func TestQuitDialogClicks(t *testing.T) {
	s := newTestSketch(40, 10)
	dialog := openQuitDialog(t, s, modePaint)
	originX, originY := s.popupOrigin(dialog.box())
	// 테두리 한 칸, 여백 한 칸 뒤에 첫 버튼이 있다. 둘째 버튼은 첫 버튼과 사이 빈칸 뒤다.
	second := originX + 2 + ansi.StringWidth(quitButtonLabel(quitChoices[0])+quitButtonGap)
	if _, cmd := s.Update(tea.MouseClickMsg{X: second, Y: originY + 1 + 2, Button: tea.MouseLeft}); cmd == nil {
		t.Error("버리고 끝을 눌렀는데 안 끝났다")
	}

	s = newTestSketch(40, 10)
	dialog = openQuitDialog(t, s, modePaint)
	originX, originY = s.popupOrigin(dialog.box())
	if _, cmd := s.Update(tea.MouseClickMsg{X: originX - 1, Y: originY, Button: tea.MouseLeft}); cmd != nil || s.mode() != modePaint {
		t.Errorf("바깥 누름: 끝냄 %v, 모드 %s", cmd != nil, s.mode())
	}
	if _, ok := s.screen.(*viewQuit); ok {
		t.Error("바깥을 눌렀는데 창이 떠 있다")
	}
}

// 저장에 실패하면 끝내지 않고 창을 닫는다. 까닭은 띠에 있다.
func TestQuitDialogSaveFailureKeepsRunning(t *testing.T) {
	s := newTestScreen(filepath.Join(t.TempDir(), "없는 곳", "mock.json"), NewCanvas(10, 5))
	s.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	openQuitDialog(t, s, modeBrush)
	if _, cmd := s.Update(typed("s")); cmd != nil {
		t.Fatal("저장에 실패했는데 끝났다")
	}
	if !onMode(s.screen) || s.message == "" {
		t.Errorf("화면 %T, 알림 %q", s.screen, s.message)
	}
}

// 글자 모드에서도 ctrl+c 가 창을 띄우고, 창의 키(s · d)는 글자가 아니라 버튼이다.
func TestQuitDialogFromTextMode(t *testing.T) {
	s := newTestSketch(10, 5)
	openQuitDialog(t, s, modeText)
	if _, cmd := s.Update(typed("d")); cmd == nil {
		t.Error("글자 모드에서 띄운 창의 d 가 안 끝냈다")
	}
	if s.canvas.At(0, 0).Glyph == "d" {
		t.Error("d 가 판에 적혔다")
	}
}
