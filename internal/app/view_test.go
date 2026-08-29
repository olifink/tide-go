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

		t.Logf("Size %dx%d (default): view produced %d lines (expected <= %d)", w, h, lineCount, h)
		if lineCount > h {
			t.Errorf("FAIL: Size %dx%d produced %d lines (overflows by %d lines!)", w, h, lineCount, lineCount-h)
		}

		// Test maximized console line count
		m.ConsoleMaximized = true
		m.recalculateLayout()
		viewMax := m.View()
		linesMax := strings.Split(viewMax, "\n")
		lineCountMax := len(linesMax)

		t.Logf("Size %dx%d (maximized): view produced %d lines (expected <= %d)", w, h, lineCountMax, h)
		if lineCountMax > h {
			t.Errorf("FAIL: Size %dx%d maximized produced %d lines (overflows by %d lines!)", w, h, lineCountMax, lineCountMax-h)
		}

		// Test fullscreen editor line count
		m.ConsoleMaximized = false
		m.EditorFullscreen = true
		m.recalculateLayout()
		viewFull := m.View()
		linesFull := strings.Split(viewFull, "\n")
		lineCountFull := len(linesFull)

		t.Logf("Size %dx%d (fullscreen): view produced %d lines (expected <= %d)", w, h, lineCountFull, h)
		if lineCountFull > h {
			t.Errorf("FAIL: Size %dx%d fullscreen produced %d lines (overflows by %d lines!)", w, h, lineCountFull, lineCountFull-h)
		}
	}
}
