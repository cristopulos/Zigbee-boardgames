// Package main provides a Fyne-based GUI for switching between multiple countdown timers.
//
// The UI uses a custom timerCardWidget implementing fyne.Tappable with a custom renderer
// for clickable timer cards. Text scaling is dynamic based on card dimensions using
// canvas.Text objects. State updates are thread-safe via sync.Mutex, as the timer tick
// goroutine may update state concurrently with renderer refreshes.
//
// The window is centered on screen and requests focus on launch. Note: always-on-top
// is not supported by Fyne.
package main

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	bgColor        = color.NRGBA{R: 26, G: 26, B: 46, A: 255}
	inactiveColor  = color.NRGBA{R: 40, G: 40, B: 50, A: 255}
	activeColor    = color.NRGBA{R: 26, G: 58, B: 92, A: 255}
	activeBorder   = color.NRGBA{R: 0, G: 180, B: 216, A: 255}
	inactiveBorder = color.NRGBA{R: 85, G: 85, B: 85, A: 255}
	white          = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	cyan           = color.NRGBA{R: 144, G: 224, B: 239, A: 255}
	grey           = color.NRGBA{R: 170, G: 170, B: 170, A: 255}
	amber          = color.NRGBA{R: 255, G: 193, B: 7, A: 255}
)

// timerCardWidget is a single timer display card that scales its text to fit
// the available space and handles tap events directly (no button overlay).
type timerCardWidget struct {
	widget.BaseWidget
	mu sync.Mutex

	index    int
	isActive bool
	isPaused bool
	nameStr  string
	timeStr  string
	onTapped func()

	// Renderer-managed objects
	bg       *canvas.Rectangle
	nameText *canvas.Text
	timeText *canvas.Text
}

func newTimerCardWidget(name string, index int, onTapped func()) *timerCardWidget {
	t := &timerCardWidget{
		index:    index,
		nameStr:  name,
		onTapped: onTapped,
	}
	t.ExtendBaseWidget(t)
	return t
}

// Tapped implements fyne.Tappable so the card responds to mouse/touch clicks
// without needing a widget.Button overlay (which adds unwanted hover visuals).
func (t *timerCardWidget) Tapped(_ *fyne.PointEvent) {
	if t.onTapped != nil {
		t.onTapped()
	}
}

// CreateRenderer builds the visual representation of the card.
func (t *timerCardWidget) CreateRenderer() fyne.WidgetRenderer {
	t.bg = canvas.NewRectangle(inactiveColor)
	t.bg.CornerRadius = 12
	t.bg.StrokeColor = inactiveBorder
	t.bg.StrokeWidth = 1

	t.nameText = canvas.NewText(t.nameStr, grey)
	t.nameText.Alignment = fyne.TextAlignCenter
	t.nameText.TextStyle = fyne.TextStyle{Bold: true}

	t.timeText = canvas.NewText(t.timeStr, white)
	t.timeText.Alignment = fyne.TextAlignCenter
	t.timeText.TextStyle = fyne.TextStyle{Bold: true}

	return &timerCardRenderer{
		widget:  t,
		objects: []fyne.CanvasObject{t.bg, t.nameText, t.timeText},
	}
}

// timerCardRenderer draws the card background and two text lines.
type timerCardRenderer struct {
	widget  *timerCardWidget
	objects []fyne.CanvasObject
}

func (r *timerCardRenderer) Layout(size fyne.Size) {
	// Background fills the whole card.
	r.widget.bg.Resize(size)
	r.widget.bg.Move(fyne.NewPos(0, 0))

	// Scale text relative to the smaller card dimension so it looks good
	// whether the window is wide/short or tall/narrow.
	minDim := size.Width
	if size.Height < minDim {
		minDim = size.Height
	}

	nameSize := minDim * 0.18
	if nameSize < 10 {
		nameSize = 10
	}
	if nameSize > 40 {
		nameSize = 40
	}

	timeSize := minDim * 0.28
	if timeSize < 14 {
		timeSize = 14
	}
	if timeSize > 80 {
		timeSize = 80
	}

	// Name sits in the upper portion.
	oldNameSize := r.widget.nameText.TextSize
	r.widget.nameText.TextSize = nameSize
	r.widget.nameText.Resize(fyne.NewSize(size.Width, nameSize*1.4))
	r.widget.nameText.Move(fyne.NewPos(0, size.Height*0.12))
	if oldNameSize != nameSize {
		r.widget.nameText.Refresh()
	}

	// Time sits below the name.
	oldTimeSize := r.widget.timeText.TextSize
	r.widget.timeText.TextSize = timeSize
	r.widget.timeText.Resize(fyne.NewSize(size.Width, timeSize*1.4))
	r.widget.timeText.Move(fyne.NewPos(0, size.Height*0.42))
	if oldTimeSize != timeSize {
		r.widget.timeText.Refresh()
	}
}

func (r *timerCardRenderer) MinSize() fyne.Size {
	return fyne.NewSize(120, 100)
}

func (r *timerCardRenderer) Refresh() {
	r.widget.mu.Lock()
	isActive := r.widget.isActive
	isPaused := r.widget.isPaused
	nameStr := r.widget.nameStr
	timeStr := r.widget.timeStr
	r.widget.mu.Unlock()

	if isActive {
		r.widget.bg.FillColor = activeColor
		r.widget.bg.StrokeColor = activeBorder
		r.widget.bg.StrokeWidth = 2
		r.widget.nameText.Color = cyan
		if isPaused {
			r.widget.timeText.Color = amber
		} else {
			r.widget.timeText.Color = white
		}
	} else {
		r.widget.bg.FillColor = inactiveColor
		r.widget.bg.StrokeColor = inactiveBorder
		r.widget.bg.StrokeWidth = 1
		r.widget.nameText.Color = grey
		r.widget.timeText.Color = white
	}

	r.widget.nameText.Text = nameStr
	r.widget.timeText.Text = timeStr

	// Re-run layout so text sizes stay correct after a resize.
	if s := r.widget.Size(); s.Width > 0 && s.Height > 0 {
		r.Layout(s)
	}

	for _, obj := range r.objects {
		canvas.Refresh(obj)
	}
}

func (r *timerCardRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *timerCardRenderer) Destroy() {}

// TimerUI manages the Fyne window and all timer cards. Call Show() to display the window.
type TimerUI struct {
	window fyne.Window
	tm     *TimerManager
	cards  []*timerCardWidget
	ticker *time.Ticker
	quit   chan struct{}
	debug  bool
}

// NewTimerUI creates the GUI window, centered on screen with focus requested.
// The window is not set always-on-top (Fyne limitation).
func NewTimerUI(tm *TimerManager) *TimerUI {
	a := app.New()
	w := a.NewWindow("Timer Switcher")
	w.Resize(fyne.NewSize(900, 280))
	w.SetFixedSize(false)
	w.CenterOnScreen()
	w.RequestFocus()

	ui := &TimerUI{
		window: w,
		tm:     tm,
		quit:   make(chan struct{}),
	}

	ui.buildUI()
	ui.setupKeyboard()
	ui.startTicker()

	w.SetOnClosed(func() { ui.Stop() })
	return ui
}

func (ui *TimerUI) buildUI() {
	count := ui.tm.Count()
	ui.cards = make([]*timerCardWidget, count)

	cardsContainer := container.NewGridWithColumns(count)
	for i := 0; i < count; i++ {
		idx := i
		card := newTimerCardWidget(ui.tm.TimerName(i), i, func() {
			if ui.debug {
				fmt.Printf("[ui] card tapped: index=%d\n", idx)
			}
			ui.tm.SwitchTo(idx)
			ui.refreshAll()
		})
		ui.cards[i] = card
		cardsContainer.Add(card)
	}

	hint := widget.NewLabel("SPACE: Switch  ENTER: Reset  P: Pause  ESC: Quit")
	hint.Alignment = fyne.TextAlignCenter
	hint.TextStyle = fyne.TextStyle{Italic: true}

	bg := canvas.NewRectangle(bgColor)
	content := container.NewBorder(nil, hint, nil, nil, cardsContainer)
	ui.window.SetContent(container.NewStack(bg, content))

	ui.refreshAll()
}

func (ui *TimerUI) refreshAll() {
	_, elapsed, activeIdx, paused := ui.tm.Snapshot()
	if ui.debug {
		fmt.Printf("[ui] refreshAll: active=%d paused=%v elapsed=%v\n", activeIdx, paused, elapsed)
	}
	for i, card := range ui.cards {
		card.mu.Lock()
		card.isActive = i == activeIdx
		card.isPaused = paused
		card.timeStr = FormatElapsed(elapsed[i])
		card.mu.Unlock()
		card.Refresh()
	}
}

func (ui *TimerUI) setupKeyboard() {
	ui.window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ui.debug {
			fmt.Printf("[ui] key pressed: %s\n", ev.Name)
		}
		switch ev.Name {
		case fyne.KeySpace:
			ui.tm.Cycle()
			ui.refreshAll()
		case fyne.KeyReturn:
			ui.tm.Reset()
			ui.refreshAll()
		case fyne.KeyP:
			ui.tm.TogglePause()
			ui.refreshAll()
		case fyne.KeyEscape:
			ui.Stop()
			ui.window.Close()
		}
	})
}

func (ui *TimerUI) startTicker() {
	ui.ticker = time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-ui.ticker.C:
				if ui.debug {
					fmt.Printf("[ui] ticker fired\n")
				}
				ui.tm.Tick()
				ui.refreshAll()
			case <-ui.quit:
				if ui.debug {
					fmt.Printf("[ui] ticker goroutine exiting\n")
				}
				return
			}
		}
	}()
}

func (ui *TimerUI) Stop() {
	select {
	case <-ui.quit:
	default:
		close(ui.quit)
	}
	if ui.ticker != nil {
		ui.ticker.Stop()
	}
}

// Show displays the window and runs the Fyne event loop. Blocks until the window is closed.
func (ui *TimerUI) Show() {
	ui.window.ShowAndRun()
}
