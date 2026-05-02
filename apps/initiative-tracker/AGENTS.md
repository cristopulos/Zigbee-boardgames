# AGENTS.md — initiative-tracker

TI4 initiative tracker: 8 cards (1-8) by default, or 9 cards (0-8) with --naalu. Single active at a time. Tap or number keys to toggle enabled, keyboard/buttons to advance.

## Developer Commands

```bash
cd button-hub/apps/initiative-tracker

go test ./...           # 37 tests
go build -o initiative-tracker .
./build.sh              # builds + checks X11/Wayland/OpenGL deps
```

Run with:
- `--api=http://host:3000 --button=<id1>,<id2>` — with button control (comma-separated)
- `--naalu` — include Naalu initiative 0 (default: 8 cards, Naalu excluded)
- `--start=N` — starting initiative (default: 0)
- Keyboard-only: no args needed

## Architecture

- **State**: `TrackerState` struct with mutex-protected `current` (int) and `enabled` (slice of bool, dynamic length)
- **Cards**: Created dynamically in `NewTrackerUI` based on `numInitiatives` (8 or 9)
- **Custom widgets**: `initiativeCardWidget` (implements `fyne.Tappable`) + `initiativeCardRenderer`
- **Layout**: `container.NewGridWithColumns(n, ...)` stacked over dark background
- **Refresh pattern**: renderer holds mutex, copies widget state, calls `canvas.Refresh()` on children (not widget.Refresh())

## Fyne UI Gotchas

These took significant debugging:
1. **Use `app.New()` + `a.NewWindow()`** — not `fyne.CurrentApp()` which returns nil in non-GUI contexts. Timer-switcher uses `app.New()`.
2. **Custom widget `Refresh()` calls don't propagate** — always use `canvas.Refresh()` on canvas primitives
3. **`TextSize` in Layout()** — must track `oldNameSize`/`oldNumSize`, compare, set, then `text.Refresh()`. Fyne caches glyphs.
4. **`SetOnTypedKey`** — keyboard capture via `window.Canvas().SetOnTypedKey()`, not `desktop.CustomShortcut`
5. **No button widgets** — use `fyne.Tappable` interface directly to avoid `widget.Button` hover artifacts

## Button Event Mapping

| Action | Result |
|--------|--------|
| `ActionSingle` | `state.Next()` |
| `ActionDouble` | `state.Reset(start)` |
| `ActionLongPress` | ignored |

Multiple button IDs supported (comma-separated). Each button triggers Next on Single.

## Keyboard Controls

| Key | Action |
|-----|--------|
| Space, Right, Up | Next initiative |
| Backspace, Left, Down | Previous initiative |
| R | Reset to start |
| 0-8 | Toggle enable/disable that tile |
| Escape | Quit |

## Strategy Cards (verified from TI4 artwork)

| # | Name | Color |
|---|------|-------|
| 0 | Naalu | Teal (#00B4D8) |
| 1 | Leadership | Red (#DF2322) |
| 2 | Diplomacy | Orange (#ED9237) |
| 3 | Politics | Yellow (#FAF01D) |
| 4 | Construction | Green (#30AF60) |
| 5 | Trade | Teal (#03A691) |
| 6 | Warfare | Light Blue (#1B8BCD) |
| 7 | Technology | Dark Blue (#1B4597) |
| 8 | Imperial | Purple (#894AA5) |

Inactive cards: dimmed to 30% opacity. Disabled cards: grey (#333).

## Testing

- 38 tests in `ui_test.go`
- UI tests: always use `test.NewApp()` + `defer a.Quit()`
- TrackerState tests use dynamic numInitiatives parameter

## go.mod

```go
replace github.com/cristopulos/button-hub/go => ../../go
```
Critical — must have this replace directive for local development.

## Gotchas

- Default: 8 cards (Naalu excluded). Use `--naalu` to include initiative 0.
- `NewTrackerState` uses slice instead of fixed array for dynamic length
- When all cards disabled, Next/Prev stay at current (no wrap-around panic)
- `refreshCh` is buffered (capacity 1) with non-blocking send