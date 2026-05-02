package main

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
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
	bg := canvas.NewRectangle(color.RGBA{R: 0xDF, G: 0x23, B: 0x22, A: 0xFF})
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

// Max 9 initiatives (including Naalu). TrackerState manages actual count.
const numInitiatives = 9

// --- Strategy card data (verified from TI4 artwork) ---
// 0=Naalu, 1=Leadership, 2=Diplomacy, 3=Politics, 4=Construction, 5=Trade, 6=Warfare, 7=Technology, 8=Imperial

var initiativeData = [numInitiatives]struct {
	name  string
	color color.RGBA
}{
	{"Naalu", color.RGBA{R: 0x00, G: 0xB4, B: 0xD8, A: 0xFF}},       // 0 - Teal (Naalu token)
	{"Leadership", color.RGBA{R: 0xDF, G: 0x23, B: 0x22, A: 0xFF}},   // 1 - Red
	{"Diplomacy", color.RGBA{R: 0xED, G: 0x92, B: 0x37, A: 0xFF}},    // 2 - Orange
	{"Politics", color.RGBA{R: 0xFA, G: 0xF0, B: 0x1D, A: 0xFF}},    // 3 - Yellow
	{"Construction", color.RGBA{R: 0x30, G: 0xAF, B: 0x60, A: 0xFF}}, // 4 - Green
	{"Trade", color.RGBA{R: 0x03, G: 0xA6, B: 0x91, A: 0xFF}},        // 5 - Teal
	{"Warfare", color.RGBA{R: 0x1B, G: 0x8B, B: 0xCD, A: 0xFF}},      // 6 - Light Blue
	{"Technology", color.RGBA{R: 0x1B, G: 0x45, B: 0x97, A: 0xFF}},   // 7 - Dark Blue
	{"Imperial", color.RGBA{R: 0x89, G: 0x4A, B: 0xA5, A: 0xFF}},     // 8 - Purple
}

func cardName(index int) string {
	if index < 0 || index >= len(initiativeData) {
		return ""
	}
	return initiativeData[index].name
}

func cardNumber(index int) string {
	if index < 0 || index >= len(initiativeData) {
		return ""
	}
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
	cards     []*initiativeCardWidget
	bg        *canvas.Rectangle
	quit      chan struct{}
	startInit int
}

func NewTrackerUI(state *TrackerState, refreshCh <-chan struct{}, numInitiatives int) *TrackerUI {
	a := app.New()
	w := a.NewWindow("Initiative Tracker")

	ui := &TrackerUI{
		window:    w,
		state:     state,
		bg:        canvas.NewRectangle(color.RGBA{R: 0x1a, G: 0x1a, B: 0x2e, A: 0xFF}),
		quit:      make(chan struct{}),
		startInit: state.Current(),
	}

	// Create cards based on numInitiatives
	ui.cards = make([]*initiativeCardWidget, 0, numInitiatives)
	for i := 0; i < numInitiatives; i++ {
		idx := i
		card := newInitiativeCardWidget(idx, func() {
			ui.state.ToggleEnabled(idx)
			ui.refreshAll()
		})
		ui.cards = append(ui.cards, card)
	}

	w.SetContent(ui.buildUI())
	w.Resize(fyne.NewSize(1200, 200))
	w.CenterOnScreen()
	w.RequestFocus()

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
		default:
			// Number keys 0-8 to toggle tile enabled/disabled
			changed = handleNumberKey(ev.Name, state)
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

func handleNumberKey(name fyne.KeyName, state *TrackerState) bool {
	var index int = -1
	switch name {
	case fyne.Key0:
		index = 0
	case fyne.Key1:
		index = 1
	case fyne.Key2:
		index = 2
	case fyne.Key3:
		index = 3
	case fyne.Key4:
		index = 4
	case fyne.Key5:
		index = 5
	case fyne.Key6:
		index = 6
	case fyne.Key7:
		index = 7
	case fyne.Key8:
		index = 8
	}
	if index >= 0 {
		state.ToggleEnabled(index)
		return true
	}
	return false
}

func (ui *TrackerUI) buildUI() fyne.CanvasObject {
	cards := make([]fyne.CanvasObject, 0, len(ui.cards))
	for _, card := range ui.cards {
		cards = append(cards, card)
	}

	grid := container.NewGridWithColumns(len(cards), cards...)
	return container.NewStack(ui.bg, grid)
}

func (ui *TrackerUI) refreshAll() {
	current := ui.state.Current()
	allEnabled := ui.state.AllEnabled()

	for i := 0; i < len(ui.cards); i++ {
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