# AGENTS.md — initiative-tracker

TI4 initiative tracker: 9 cards (0-8), single active at a time. Tap to toggle enabled, keyboard/buttons to advance.

## Developer Commands

```bash
cd button-hub/apps/initiative-tracker

go test ./...           # 35 tests
go build -o initiative-tracker .
./build.sh              # builds + checks X11/Wayland/OpenGL deps
```

Run with:
- `--api=http://host:3000 --button=<id>` — with button control
- `--start=0` — start at Naalu (default: 1, Leadership)
- Keyboard-only: no args needed

## Architecture

- **State**: `TrackerState` struct with mutex-protected `current` (int) and `enabled[9]` (bool array)
- **Custom widgets**: `initiativeCardWidget` (implements `fyne.Tappable`) + `initiativeCardRenderer`
- **Layout**: `container.NewGridWithColumns(9, ...)` stacked over dark background
- **Refresh pattern**: renderer holds mutex, copies widget state, calls `canvas.Refresh()` on children (not widget.Refresh())

## Fyne UI Gotchas

These took significant debugging:
1. **Custom widget `Refresh()` calls don't propagate** — always use `canvas.Refresh()` on canvas primitives (`r.bg`, `r.numText`, etc.)
2. **`TextSize` in Layout()** — must track `oldNameSize`/`oldNumSize`, compare, set, then `text.Refresh()`. Fyne caches glyphs; size changes don't auto-refresh.
3. **`SetOnTypedKey`** — keyboard capture via `window.Canvas().SetOnTypedKey()`, not `desktop.CustomShortcut`
4. **No button widgets** — use `fyne.Tappable` interface directly to avoid `widget.Button` hover artifacts

## Button Event Mapping

| Action | Result |
|--------|--------|
| `ActionSingle` | `state.Next()` |
| `ActionDouble` | `state.Reset(startFlag)` |
| `ActionLongPress` | ignored |

## Keyboard Controls

| Key | Action |
|-----|--------|
| Space, Right, Up | Next |
| Backspace, Left, Down | Prev |
| R | Reset to start |
| Escape | Quit |

## Tap Behavior

- **Tap card** → `ToggleEnabled(i)` — enables/disables that card only
- **Does NOT change active initiative** — active changes only via keyboard/buttons (Next/Prev/Reset)
- If you disable the active card, initiative auto-advances to next enabled card

## Strategy Cards (0-8)

| # | Name | Color |
|---|------|-------|
| 0 | Naalu | Teal (#00B4D8) |
| 1 | Leadership | Gold (#E6A817) |
| 2 | Diplomacy | Blue (#0077B6) |
| 3 | Politics | Purple (#7B2D8B) |
| 4 | Construction | Green (#2E7D32) |
| 5 | Trade | Yellow (#F9A825) |
| 6 | Warfare | Red (#D32F2F) |
| 7 | Technology | Orange (#E65100) |
| 8 | Imperial | Black (#212121) |

Inactive cards: dimmed to 30% opacity. Disabled cards: grey (#333).

## Testing

- 35 tests in `ui_test.go` (18 UI/widget + 17 TrackerState machine)
- UI tests: always use `test.NewApp()` + `defer a.Quit()`
- Renderer tests: call `Layout()` with `fyne.NewSize(w, h)` directly (no window needed for unit tests)

## go.mod

```go
replace github.com/cristopulos/button-hub/go => ../../go
```
Critical — must have this replace directive for local development.

## Gotchas

- `NewTrackerState(-1)` or `NewTrackerState(9)` defaults to 1 (Leadership)
- When all cards disabled, Next/Prev stay at current (no wrap-around panic)
- `refreshCh` is buffered (capacity 1) with non-blocking send — prevents missed updates if multiple button presses occur before UI goroutine processes