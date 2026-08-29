package app

import (
	"strings"
	"testing"
)

func TestAppViewLineCount(t *testing.T) {
	sizes := [][2]int{
		{80, 24},
		{100, 30},
		{120, 40},
		{160, 50},
	}

	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		m := InitialModel(".")
		m.Width = w
		m.Height = h
		m.recalculateLayout()

		view := m.View()
		lines := strings.Split(view, "\n")
		lineCount := len(lines)

		t.Logf("Size %dx%d: view produced %d lines (expected <= %d)", w, h, lineCount, h)
		if lineCount > h {
			t.Errorf("FAIL: Size %dx%d produced %d lines (overflows by %d lines!)", w, h, lineCount, lineCount-h)
		}
	}
}
