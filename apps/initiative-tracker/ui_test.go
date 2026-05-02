package main

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestInitiativeCardWidgetConstructor(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	tapped := false
	w := newInitiativeCardWidget(3, func() { tapped = true })

	if w.index != 3 {
		t.Errorf("expected index 3, got %d", w.index)
	}
	if !w.isEnabled {
		t.Error("expected isEnabled to be true initially")
	}
	if w.onTapped == nil {
		t.Error("onTapped should not be nil")
	}

	w.Tapped(nil)
	if !tapped {
		t.Error("expected tapped callback to be called")
	}
}

func TestInitiativeCardWidgetTapped(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	called := false
	w := newInitiativeCardWidget(1, func() { called = true })
	w.Tapped(nil)
	if !called {
		t.Error("expected tapped callback to be called")
	}
}

func TestInitiativeCardWidgetTappedNilCallback(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	w := newInitiativeCardWidget(2, nil)
	w.Tapped(nil)
}

func TestInitiativeCardRendererMinSize(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	w := newInitiativeCardWidget(0, nil)
	r := newInitiativeCardRenderer(w)

	min := r.MinSize()
	if min.Width < 1 || min.Height < 1 {
		t.Errorf("MinSize should have positive dimensions, got %v", min)
	}
}

func TestInitiativeCardRendererObjects(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	w := newInitiativeCardWidget(0, nil)
	r := newInitiativeCardRenderer(w)

	objs := r.Objects()
	if len(objs) != 3 {
		t.Errorf("expected 3 objects (bg, numText, nameText), got %d", len(objs))
	}
}

func TestInitiativeCardColors(t *testing.T) {
	expectedColors := [numInitiatives]color.RGBA{
		{R: 0x00, G: 0xB4, B: 0xD8, A: 0xFF}, // Naalu - Teal
		{R: 0xE6, G: 0xA8, B: 0x17, A: 0xFF}, // Leadership - Gold
		{R: 0x00, G: 0x77, B: 0xB6, A: 0xFF}, // Diplomacy - Blue
		{R: 0x7B, G: 0x2D, B: 0x8B, A: 0xFF}, // Politics - Purple
		{R: 0x2E, G: 0x7D, B: 0x32, A: 0xFF}, // Construction - Green
		{R: 0xF9, G: 0xA8, B: 0x25, A: 0xFF}, // Trade - Yellow/Gold
		{R: 0xD3, G: 0x2F, B: 0x2F, A: 0xFF}, // Warfare - Red
		{R: 0xE6, G: 0x51, B: 0x00, A: 0xFF}, // Technology - Orange
		{R: 0x21, G: 0x21, B: 0x21, A: 0xFF}, // Imperial - Black
	}

	for i := 0; i < numInitiatives; i++ {
		bg, _, _ := colorsForState(i, true, true)
		exp := expectedColors[i]
		if bg.R != exp.R || bg.G != exp.G || bg.B != exp.B || bg.A != exp.A {
			t.Errorf("card %d: expected color %v, got %v", i, exp, bg)
		}
	}
}

func TestInitiativeCardActiveState(t *testing.T) {
	bg, num, _ := colorsForState(3, true, true)

	if bg.A == 0 {
		t.Error("active card should have full opacity background")
	}
	if num != (color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}) {
		t.Errorf("active card should have white number, got %v", num)
	}
}

func TestInitiativeCardInactiveState(t *testing.T) {
	bg, num, name := colorsForState(4, false, true)

	if bg.A == 0 {
		t.Error("inactive enabled card should have dimmed background")
	}
	if num == (color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}) {
		t.Error("inactive card should not have white number")
	}
	_ = name
}

func TestInitiativeCardDisabledState(t *testing.T) {
	bg, num, name := colorsForState(5, true, false)

	if bg != (color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}) {
		t.Errorf("disabled card should have #333 background, got %v", bg)
	}
	if num != (color.RGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xFF}) {
		t.Errorf("disabled card should have dim number, got %v", num)
	}
	if name != (color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}) {
		t.Errorf("disabled card should have dim name, got %v", name)
	}
}

func TestInitiativeCardRendererLayoutSmall(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	w := newInitiativeCardWidget(0, nil)
	r := newInitiativeCardRenderer(w)
	r.Layout(fyne.NewSize(60, 60))

	if len(r.Objects()) != 3 {
		t.Error("renderer should have 3 objects")
	}
}

func TestInitiativeCardRendererLayoutLarge(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	w := newInitiativeCardWidget(4, nil)
	r := newInitiativeCardRenderer(w)
	r.Layout(fyne.NewSize(200, 150))

	if len(r.Objects()) != 3 {
		t.Error("renderer should have 3 objects")
	}
}

func TestInitiativeCardRendererLayoutTall(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	w := newInitiativeCardWidget(6, nil)
	r := newInitiativeCardRenderer(w)
	r.Layout(fyne.NewSize(80, 200))

	if len(r.Objects()) != 3 {
		t.Error("renderer should have 3 objects")
	}
}

func TestInitiativeCardRendererLayoutWide(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	w := newInitiativeCardWidget(7, nil)
	r := newInitiativeCardRenderer(w)
	r.Layout(fyne.NewSize(300, 80))

	if len(r.Objects()) != 3 {
		t.Error("renderer should have 3 objects")
	}
}

func TestInitiativeCardRendererRefreshNoPanicOnZeroSize(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	w := newInitiativeCardWidget(2, nil)
	r := newInitiativeCardRenderer(w)
	r.Refresh()
}

func TestCardName(t *testing.T) {
	names := []string{
		"Naalu", "Leadership", "Diplomacy", "Politics", "Construction",
		"Trade", "Warfare", "Technology", "Imperial",
	}

	for i, expected := range names {
		got := cardName(i)
		if got != expected {
			t.Errorf("cardName(%d): expected %q, got %q", i, expected, got)
		}
	}

	if cardName(-1) != "" {
		t.Error("cardName(-1) should return empty string")
	}
	if cardName(numInitiatives) != "" {
		t.Error("cardName(numInitiatives) should return empty string")
	}
}

func TestCardNumber(t *testing.T) {
	if cardNumber(0) != "0" {
		t.Errorf("cardNumber(0) should be \"0\", got %q", cardNumber(0))
	}
	for i := 1; i < numInitiatives; i++ {
		got := cardNumber(i)
		if got == "" {
			t.Errorf("cardNumber(%d): expected non-empty string, got empty", i)
		}
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		input    float32
		expected float32
	}{
		{5, 10},
		{10, 10},
		{30, 30},
		{60, 60},
		{65, 60},
		{0, 10},
	}

	for _, tc := range tests {
		got := clamp(tc.input)
		if got != tc.expected {
			t.Errorf("clamp(%v) = %v; want %v", tc.input, got, tc.expected)
		}
	}
}

func TestDimColor(t *testing.T) {
	c := color.RGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}
	dimmed := dimColor(c, 0x80)

	if dimmed.R != 0xFF || dimmed.G != 0x00 || dimmed.B != 0x00 || dimmed.A != 0x80 {
		t.Errorf("dimColor: expected RGBA{FF,00,00,80}, got %v", dimmed)
	}
}

// --- TrackerState tests ---

func TestNewTrackerState(t *testing.T) {
	s := NewTrackerState(1)
	if s.Current() != 1 {
		t.Errorf("expected current 1, got %d", s.Current())
	}
	for i := 0; i < numInitiatives; i++ {
		if !s.Enabled(i) {
			t.Errorf("expected all cards enabled, card %d is disabled", i)
		}
	}
}

func TestNewTrackerStateInvalidStart(t *testing.T) {
	s := NewTrackerState(-1)
	if s.Current() != 1 {
		t.Errorf("expected current 1 for invalid start -1, got %d", s.Current())
	}

	s = NewTrackerState(9)
	if s.Current() != 1 {
		t.Errorf("expected current 1 for invalid start 9, got %d", s.Current())
	}

	s = NewTrackerState(99)
	if s.Current() != 1 {
		t.Errorf("expected current 1 for invalid start 99, got %d", s.Current())
	}
}

func TestNext(t *testing.T) {
	s := NewTrackerState(1)
	s.Next()
	if s.Current() != 2 {
		t.Errorf("expected current 2, got %d", s.Current())
	}

	s.Next()
	if s.Current() != 3 {
		t.Errorf("expected current 3, got %d", s.Current())
	}
}

func TestNextWraps(t *testing.T) {
	s := NewTrackerState(8)
	s.Next()
	if s.Current() != 0 {
		t.Errorf("expected current 0 after wrapping from 8, got %d", s.Current())
	}

	s.Next()
	if s.Current() != 1 {
		t.Errorf("expected current 1 after wrapping from 0, got %d", s.Current())
	}
}

func TestNextSkipsDisabled(t *testing.T) {
	s := NewTrackerState(0)
	s.enabled[1] = false
	s.enabled[2] = false
	s.Next()
	if s.Current() != 3 {
		t.Errorf("expected current 3 after skipping 1,2, got %d", s.Current())
	}
}

func TestPrev(t *testing.T) {
	s := NewTrackerState(3)
	s.Prev()
	if s.Current() != 2 {
		t.Errorf("expected current 2, got %d", s.Current())
	}

	s.Prev()
	if s.Current() != 1 {
		t.Errorf("expected current 1, got %d", s.Current())
	}
}

func TestPrevWraps(t *testing.T) {
	s := NewTrackerState(0)
	s.Prev()
	if s.Current() != 8 {
		t.Errorf("expected current 8 after wrapping from 0, got %d", s.Current())
	}

	s.Prev()
	if s.Current() != 7 {
		t.Errorf("expected current 7 after wrapping from 8, got %d", s.Current())
	}
}

func TestPrevSkipsDisabled(t *testing.T) {
	s := NewTrackerState(4)
	s.enabled[3] = false
	s.enabled[2] = false
	s.Prev()
	if s.Current() != 1 {
		t.Errorf("expected current 1 after skipping 3,2, got %d", s.Current())
	}
}

func TestReset(t *testing.T) {
	s := NewTrackerState(1)
	s.Next()
	s.Next()
	if s.Current() != 3 {
		t.Errorf("expected current 3, got %d", s.Current())
	}

	s.Reset(1)
	if s.Current() != 1 {
		t.Errorf("expected current 1 after reset, got %d", s.Current())
	}
}

func TestResetInvalidStartUsesFirstEnabled(t *testing.T) {
	s := NewTrackerState(5)
	s.enabled[5] = false
	s.enabled[6] = false
	s.Reset(5)
	if s.Current() != 0 {
		t.Errorf("expected current 0 when reset target disabled, got %d", s.Current())
	}
}

func TestToggleEnabled(t *testing.T) {
	s := NewTrackerState(1)

	if !s.Enabled(3) {
		t.Error("expected card 3 to be enabled initially")
	}

	s.ToggleEnabled(3)
	if s.Enabled(3) {
		t.Error("expected card 3 to be disabled after toggle")
	}

	s.ToggleEnabled(3)
	if !s.Enabled(3) {
		t.Error("expected card 3 to be enabled after second toggle")
	}
}

func TestToggleEnabledCurrentCardAdvances(t *testing.T) {
	s := NewTrackerState(2)

	s.ToggleEnabled(2)
	if s.Current() != 3 {
		t.Errorf("expected current 3 after disabling current card 2, got %d", s.Current())
	}
}

func TestToggleEnabledOutOfBounds(t *testing.T) {
	s := NewTrackerState(1)
	s.ToggleEnabled(-1)
	s.ToggleEnabled(99)
	if s.Current() != 1 {
		t.Errorf("expected current unchanged after out-of-bounds toggle, got %d", s.Current())
	}
}

func TestNextAllDisabled(t *testing.T) {
	s := NewTrackerState(1)
	for i := 0; i < numInitiatives; i++ {
		s.enabled[i] = false
	}

	s.Next()
	if s.Current() != 1 {
		t.Errorf("expected current unchanged when all disabled, got %d", s.Current())
	}
}

func TestPrevAllDisabled(t *testing.T) {
	s := NewTrackerState(3)
	for i := 0; i < numInitiatives; i++ {
		s.enabled[i] = false
	}

	s.Prev()
	if s.Current() != 3 {
		t.Errorf("expected current unchanged when all disabled, got %d", s.Current())
	}
}

func TestSetCurrent(t *testing.T) {
	s := NewTrackerState(1)
	s.SetCurrent(7)
	if s.Current() != 7 {
		t.Errorf("expected current 7, got %d", s.Current())
	}
}

func TestAllEnabled(t *testing.T) {
	s := NewTrackerState(1)
	s.enabled[4] = false
	all := s.AllEnabled()

	if all[4] {
		t.Error("expected allEnabled[4] to be false")
	}
	if !all[5] {
		t.Error("expected allEnabled[5] to be true")
	}
}