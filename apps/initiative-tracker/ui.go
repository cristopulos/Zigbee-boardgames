package main

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type initiativeCardWidget struct {
	widget.BaseWidget
	mu        sync.Mutex
	index     int
	isActive  bool
	isEnabled bool
	onTapped  func()
}

func newInitiativeCardWidget(index int, onTapped func()) *initiativeCardWidget {
	w := &initiativeCardWidget{
		index:     index,
		isActive:  false,
		isEnabled: true,
		onTapped:  onTapped,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *initiativeCardWidget) CreateRenderer() fyne.WidgetRenderer {
	return newInitiativeCardRenderer(w)
}

func (w *initiativeCardWidget) Tapped(*fyne.PointEvent) {
	if w.onTapped != nil {
		w.onTapped()
	}
}

func (w *initiativeCardWidget) TappedSecondary(*fyne.PointEvent) {}

type initiativeCardRenderer struct {
	widget      *initiativeCardWidget
	bg          *canvas.Rectangle
	numText     *canvas.Text
	nameText    *canvas.Text
	minSize     fyne.Size
	oldNameSize float32
	oldNumSize  float32
}

func newInitiativeCardRenderer(w *initiativeCardWidget) *initiativeCardRenderer {
	bg := canvas.NewRectangle(color.RGBA{R: 0x00, G: 0xB4, B: 0xD8, A: 0xFF})
	numText := canvas.NewText("", color.White)
	nameText := canvas.NewText("", color.White)
	numText.Alignment = fyne.TextAlignCenter
	nameText.Alignment = fyne.TextAlignCenter
	return &initiativeCardRenderer{
		widget:   w,
		bg:       bg,
		numText:  numText,
		nameText: nameText,
		minSize:  fyne.NewSize(80, 80),
	}
}

func (r *initiativeCardRenderer) MinSize() fyne.Size {
	return r.minSize
}

func (r *initiativeCardRenderer) Layout(size fyne.Size) {
	padding := float32(6.0)
	innerW := size.Width - padding*2
	innerH := size.Height - padding*2

	nameH := innerH * 0.25
	r.nameText.Resize(fyne.NewSize(innerW, nameH))
	r.nameText.Move(fyne.NewPos(padding, padding))

	numH := innerH * 0.65
	r.numText.Resize(fyne.NewSize(innerW, numH))
	numTop := padding + nameH
	remainingH := innerH - nameH
	r.numText.Move(fyne.NewPos(padding, numTop+(remainingH-numH)/2))

	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))

	minDim := size.Width
	if size.Height < minDim {
		minDim = size.Height
	}
	nameSize := clamp(minDim * 0.18)
	numSize := clamp(minDim * 0.45)

	if nameSize != r.oldNameSize {
		r.nameText.TextSize = nameSize
		r.nameText.Refresh()
		r.oldNameSize = nameSize
	}
	if numSize != r.oldNumSize {
		r.numText.TextSize = numSize
		r.numText.Refresh()
		r.oldNumSize = numSize
	}
}

func (r *initiativeCardRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.numText, r.nameText}
}

func (r *initiativeCardRenderer) Refresh() {
	r.widget.mu.Lock()
	isActive := r.widget.isActive
	isEnabled := r.widget.isEnabled
	index := r.widget.index
	r.widget.mu.Unlock()

	bgColor, numColor, nameColor := colorsForState(index, isActive, isEnabled)

	if r.bg.FillColor != bgColor {
		r.bg.FillColor = bgColor
	}
	if r.numText.Color != numColor {
		r.numText.Color = numColor
	}
	if r.nameText.Color != nameColor {
		r.nameText.Color = nameColor
	}

	r.numText.Text = cardNumber(index)
	r.nameText.Text = cardName(index)

	canvas.Refresh(r.bg)
	canvas.Refresh(r.numText)
	canvas.Refresh(r.nameText)
}

func (r *initiativeCardRenderer) Destroy() {}

func clamp(val float32) float32 {
	if val < 10 {
		return 10
	}
	if val > 60 {
		return 60
	}
	return val
}

// --- Strategy card data ---

var initiativeData = [numInitiatives]struct {
	name  string
	color color.RGBA
}{
	{"Naalu", color.RGBA{R: 0x00, G: 0xB4, B: 0xD8, A: 0xFF}},         // 0 - Teal
	{"Leadership", color.RGBA{R: 0xE6, G: 0xA8, B: 0x17, A: 0xFF}},     // 1 - Gold
	{"Diplomacy", color.RGBA{R: 0x00, G: 0x77, B: 0xB6, A: 0xFF}},      // 2 - Blue
	{"Politics", color.RGBA{R: 0x7B, G: 0x2D, B: 0x8B, A: 0xFF}},       // 3 - Purple
	{"Construction", color.RGBA{R: 0x2E, G: 0x7D, B: 0x32, A: 0xFF}},   // 4 - Green
	{"Trade", color.RGBA{R: 0xF9, G: 0xA8, B: 0x25, A: 0xFF}},          // 5 - Yellow/Gold
	{"Warfare", color.RGBA{R: 0xD3, G: 0x2F, B: 0x2F, A: 0xFF}},        // 6 - Red
	{"Technology", color.RGBA{R: 0xE6, G: 0x51, B: 0x00, A: 0xFF}},     // 7 - Orange
	{"Imperial", color.RGBA{R: 0x21, G: 0x21, B: 0x21, A: 0xFF}},       // 8 - Black
}

func cardName(index int) string {
	names := [numInitiatives]string{
		"Naalu", "Leadership", "Diplomacy", "Politics", "Construction",
		"Trade", "Warfare", "Technology", "Imperial",
	}
	if index < 0 || index >= numInitiatives {
		return ""
	}
	return names[index]
}

func cardNumber(index int) string {
	if index == 0 {
		return "0"
	}
	digits := ""
	for i := index; i > 0; i /= 10 {
		digits = string(rune('0'+i%10)) + digits
	}
	return digits
}

func dimColor(c color.RGBA, alpha uint8) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: alpha}
}

func colorsForState(index int, isActive, isEnabled bool) (bg, num, name color.RGBA) {
	if !isEnabled {
		return color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF},
			color.RGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xFF},
			color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}
	}

	cardColor := initiativeData[index].color

	if isActive {
		return cardColor,
			color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
			color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	}

	return dimColor(cardColor, 0x4D),
		color.RGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xFF},
		color.RGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xFF}
}

// --- TrackerUI ---

type TrackerUI struct {
	window    fyne.Window
	state     *TrackerState
	cards     [numInitiatives]*initiativeCardWidget
	bg        *canvas.Rectangle
	quit      chan struct{}
	startInit int
}

func NewTrackerUI(state *TrackerState, refreshCh <-chan struct{}) *TrackerUI {
	app := fyne.CurrentApp()
	window := app.NewWindow("Initiative Tracker")

	ui := &TrackerUI{
		window:    window,
		state:     state,
		bg:        canvas.NewRectangle(color.RGBA{R: 0x1a, G: 0x1a, B: 0x2e, A: 0xFF}),
		quit:      make(chan struct{}),
		startInit: state.Current(),
	}

	window.SetContent(ui.buildUI())
	window.Resize(fyne.NewSize(1200, 200))
	window.CenterOnScreen()

	ui.window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		changed := false
		switch ev.Name {
		case fyne.KeySpace, fyne.KeyRight, fyne.KeyUp:
			state.Next()
			changed = true
		case fyne.KeyBackspace, fyne.KeyLeft, fyne.KeyDown:
			state.Prev()
			changed = true
		case fyne.KeyR:
			state.Reset(ui.startInit)
			changed = true
		case fyne.KeyEscape:
			ui.Stop()
			ui.window.Close()
		}
		if changed {
			ui.refreshAll()
		}
	})

	go func() {
		for {
			select {
			case <-refreshCh:
				ui.refreshAll()
			case <-ui.quit:
				return
			}
		}
	}()

	return ui
}

func (ui *TrackerUI) buildUI() fyne.CanvasObject {
	cards := make([]fyne.CanvasObject, numInitiatives)
	for i := 0; i < numInitiatives; i++ {
		idx := i
		ui.cards[i] = newInitiativeCardWidget(idx, func() {
			ui.state.ToggleEnabled(idx)
			ui.refreshAll()
		})
		cards[i] = ui.cards[i]
	}

	grid := container.NewGridWithColumns(numInitiatives, cards...)
	return container.NewStack(ui.bg, grid)
}

func (ui *TrackerUI) refreshAll() {
	current := ui.state.Current()
	allEnabled := ui.state.AllEnabled()

	for i := 0; i < numInitiatives; i++ {
		ui.cards[i].mu.Lock()
		ui.cards[i].isActive = (i == current)
		ui.cards[i].isEnabled = allEnabled[i]
		ui.cards[i].mu.Unlock()
		ui.cards[i].Refresh()
	}
}

func (ui *TrackerUI) Stop() {
	select {
	case <-ui.quit:
	default:
		close(ui.quit)
	}
}

func (ui *TrackerUI) Show() {
	ui.refreshAll()
	ui.window.ShowAndRun()
}