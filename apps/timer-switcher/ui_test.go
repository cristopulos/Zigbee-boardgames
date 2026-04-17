package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestNewTimerCardWidget(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	widget := newTimerCardWidget("TestTimer", 5, nil)
	if widget == nil {
		t.Fatal("newTimerCardWidget returned nil")
	}
	if widget.nameStr != "TestTimer" {
		t.Errorf("expected name 'TestTimer', got '%s'", widget.nameStr)
	}
	if widget.index != 5 {
		t.Errorf("expected index 5, got %d", widget.index)
	}
	if widget.onTapped != nil {
		t.Error("expected nil onTapped callback")
	}
}

func TestTimerCardWidgetExtendsBaseWidget(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	widget := newTimerCardWidget("Test", 0, nil)
	// Verify the widget properly extends BaseWidget by checking it's a valid widget
	if _, ok := interface{}(widget).(fyne.Widget); !ok {
		t.Error("timerCardWidget does not implement fyne.Widget")
	}
	// Verify BaseWidget functionality
	if widget.Size().Width != 0 || widget.Size().Height != 0 {
		t.Error("new widget should have zero size before being placed")
	}
}

func TestTimerCardWidgetTapped(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	tapped := false
	widget := newTimerCardWidget("Test", 0, func() {
		tapped = true
	})

	// Call Tapped
	widget.Tapped(nil)
	if !tapped {
		t.Error("Tapped did not invoke the onTapped callback")
	}
}

func TestTimerCardWidgetTappedNilCallback(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	// Should not panic when onTapped is nil
	widget := newTimerCardWidget("Test", 0, nil)
	widget.Tapped(nil)
}

func TestTimerCardRendererMinSize(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	card := newTimerCardWidget("Test", 0, nil)
	renderer := card.CreateRenderer().(*timerCardRenderer)

	minSize := renderer.MinSize()
	if minSize.Width != 120 {
		t.Errorf("expected MinSize Width 120, got %v", minSize.Width)
	}
	if minSize.Height != 100 {
		t.Errorf("expected MinSize Height 100, got %v", minSize.Height)
	}
}

func TestTimerCardRendererLayout(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	card := newTimerCardWidget("Test", 0, nil)
	renderer := card.CreateRenderer().(*timerCardRenderer)

	tests := []struct {
		name     string
		size     fyne.Size
		wantName float32
		minName  float32
		maxName  float32
		wantTime float32
		minTime  float32
		maxTime  float32
	}{
		{
			name:     "Small card",
			size:     fyne.NewSize(100, 80),
			wantName: 14, // 80*0.18=14.4
			minName:  10,
			maxName:  40,
			wantTime: 22, // 80*0.28=22.4
			minTime:  14,
			maxTime:  80,
		},
		{
			name:     "Medium card",
			size:     fyne.NewSize(200, 200),
			wantName: 36, // 200*0.18=36
			minName:  10,
			maxName:  40,
			wantTime: 56, // 200*0.28=56
			minTime:  14,
			maxTime:  80,
		},
		{
			name:     "Large card",
			size:     fyne.NewSize(500, 400),
			wantName: 40, // 400*0.18=72, clamped to max
			minName:  10,
			maxName:  40,
			wantTime: 80, // 400*0.28=112, clamped to max
			minTime:  14,
			maxTime:  80,
		},
		{
			name:     "Very small card",
			size:     fyne.NewSize(50, 50),
			wantName: 10, // min is 10
			minName:  10,
			maxName:  40,
			wantTime: 14, // min is 14
			minTime:  14,
			maxTime:  80,
		},
		{
			name:     "Tall narrow card",
			size:     fyne.NewSize(100, 300),
			wantName: 18, // 100*0.18=18
			minName:  10,
			maxName:  40,
			wantTime: 28, // 100*0.28=28
			minTime:  14,
			maxTime:  80,
		},
		{
			name:     "Wide short card",
			size:     fyne.NewSize(300, 100),
			wantName: 18, // 100*0.18=18
			minName:  10,
			maxName:  40,
			wantTime: 28, // 100*0.28=28
			minTime:  14,
			maxTime:  80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer.Layout(tt.size)

			nameSize := renderer.widget.nameText.TextSize
			if nameSize < tt.minName || nameSize > tt.maxName {
				t.Errorf("nameText.TextSize = %v, want between %v and %v", nameSize, tt.minName, tt.maxName)
			}

			timeSize := renderer.widget.timeText.TextSize
			if timeSize < tt.minTime || timeSize > tt.maxTime {
				t.Errorf("timeText.TextSize = %v, want between %v and %v", timeSize, tt.minTime, tt.maxTime)
			}

			// Verify background was resized to match card size
			if renderer.widget.bg.Size() != tt.size {
				t.Errorf("bg.Size() = %v, want %v", renderer.widget.bg.Size(), tt.size)
			}
		})
	}
}

func TestTimerCardRendererRefresh(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	card := newTimerCardWidget("TestTimer", 0, nil)
	canvas := test.Canvas()
	canvas.SetContent(card)
	card.Resize(fyne.NewSize(200, 200))

	// Initial state should be inactive
	card.mu.Lock()
	card.isActive = false
	card.mu.Unlock()

	renderer := card.CreateRenderer().(*timerCardRenderer)
	renderer.Refresh()

	// Check inactive state colors
	if renderer.widget.bg.FillColor != inactiveColor {
		t.Errorf("expected inactive FillColor, got %v", renderer.widget.bg.FillColor)
	}
	if renderer.widget.bg.StrokeColor != inactiveBorder {
		t.Errorf("expected inactive StrokeColor, got %v", renderer.widget.bg.StrokeColor)
	}
	if renderer.widget.bg.StrokeWidth != 1 {
		t.Errorf("expected inactive StrokeWidth 1, got %v", renderer.widget.bg.StrokeWidth)
	}
	if renderer.widget.nameText.Color != grey {
		t.Errorf("expected grey name color, got %v", renderer.widget.nameText.Color)
	}

	// Set active and refresh
	card.mu.Lock()
	card.isActive = true
	card.mu.Unlock()

	renderer.Refresh()

	// Check active state colors
	if renderer.widget.bg.FillColor != activeColor {
		t.Errorf("expected active FillColor, got %v", renderer.widget.bg.FillColor)
	}
	if renderer.widget.bg.StrokeColor != activeBorder {
		t.Errorf("expected active StrokeColor, got %v", renderer.widget.bg.StrokeColor)
	}
	if renderer.widget.bg.StrokeWidth != 2 {
		t.Errorf("expected active StrokeWidth 2, got %v", renderer.widget.bg.StrokeWidth)
	}
	if renderer.widget.nameText.Color != cyan {
		t.Errorf("expected cyan name color, got %v", renderer.widget.nameText.Color)
	}

	// Check text content update
	card.mu.Lock()
	card.nameStr = "UpdatedName"
	card.timeStr = "01:30:00"
	card.mu.Unlock()

	renderer.Refresh()

	if renderer.widget.nameText.Text != "UpdatedName" {
		t.Errorf("expected nameText 'UpdatedName', got '%s'", renderer.widget.nameText.Text)
	}
	if renderer.widget.timeText.Text != "01:30:00" {
		t.Errorf("expected timeText '01:30:00', got '%s'", renderer.widget.timeText.Text)
	}
}

func TestTimerCardRendererObjects(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	card := newTimerCardWidget("Test", 0, nil)
	renderer := card.CreateRenderer().(*timerCardRenderer)

	objects := renderer.Objects()
	if len(objects) != 3 {
		t.Errorf("expected 3 objects, got %d", len(objects))
	}

	// Verify all objects are canvas objects
	for i, obj := range objects {
		if _, ok := obj.(fyne.CanvasObject); !ok {
			t.Errorf("Objects()[%d] does not implement fyne.CanvasObject", i)
		}
	}
}

func TestTimerCardRendererRefreshNoPanicOnZeroSize(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	card := newTimerCardWidget("Test", 0, nil)
	renderer := card.CreateRenderer().(*timerCardRenderer)

	// Refresh on a widget with zero size should not panic
	card.mu.Lock()
	card.isActive = true
	card.mu.Unlock()

	renderer.Refresh()
}

func TestTimerCardColors(t *testing.T) {
	// Verify color constants are defined correctly
	if bgColor.A != 255 {
		t.Error("bgColor should be fully opaque")
	}
	if inactiveColor.A != 255 {
		t.Error("inactiveColor should be fully opaque")
	}
	if activeColor.A != 255 {
		t.Error("activeColor should be fully opaque")
	}
}

func TestTimerCardWidgetImplementsTappable(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	card := newTimerCardWidget("Test", 0, nil)
	if _, ok := interface{}(card).(fyne.Tappable); !ok {
		t.Error("timerCardWidget does not implement fyne.Tappable")
	}
}

func TestTimerCardWidgetRefresh(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	card := newTimerCardWidget("Test", 0, nil)
	canvas := test.Canvas()
	canvas.SetContent(card)
	card.Resize(fyne.NewSize(200, 200))

	// Get the renderer created when the canvas sets the content
	renderer := card.CreateRenderer().(*timerCardRenderer)

	// Update the card state
	card.mu.Lock()
	card.isActive = true
	card.timeStr = "00:05:30"
	card.mu.Unlock()

	// Call Refresh through the renderer
	renderer.Refresh()

	// Check the colors were updated
	if renderer.widget.bg.FillColor != activeColor {
		t.Errorf("expected active FillColor after Refresh, got %v", renderer.widget.bg.FillColor)
	}
}
