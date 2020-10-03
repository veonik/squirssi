package squirssi

import (
	"fmt"
	"sync"

	"code.dopame.me/veonik/squircy3/event"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squirssi/widget"
)

type WindowManager struct {
	windows     []Window
	activeIndex int

	status *StatusWindow

	events *event.Dispatcher

	mu sync.RWMutex
}

func NewWindowManager(ev *event.Dispatcher) *WindowManager {
	wm := &WindowManager{events: ev}
	wm.status = &StatusWindow{bufferedWindow: newBufferedWindow("status", ev)}
	wm.windows = []Window{wm.status}
	return wm
}

func (wm *WindowManager) TabNames() ([]string, map[int]widget.ActivityType) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	res := make([]string, len(wm.windows))
	activity := make(map[int]widget.ActivityType)
	for i := 0; i < len(wm.windows); i++ {
		win := wm.windows[i]
		if win.HasNotice() {
			activity[i] = widget.TabHasNotice
		} else if win.HasActivity() {
			activity[i] = widget.TabHasActivity
		}
		if wm.activeIndex == i {
			res[i] = fmt.Sprintf(" %s ", win.Title())
		} else {
			res[i] = fmt.Sprintf(" %d ", i)
		}
	}
	return res, activity
}

func (wm *WindowManager) Windows() []Window {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	wins := make([]Window, len(wm.windows))
	copy(wins, wm.windows)
	return wins
}

func (wm *WindowManager) Len() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return len(wm.windows)
}

func (wm *WindowManager) ActiveIndex() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.activeIndex
}

func (wm *WindowManager) Active() Window {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.windows[wm.activeIndex]
}

func (wm *WindowManager) Append(w Window) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.windows = append(wm.windows, w)
}

// Named returns the window with the given name, if it exists.
func (wm *WindowManager) Named(name string) Window {
	var win Window
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	for _, w := range wm.windows {
		if w.Title() == name {
			win = w
			break
		}
	}
	return win
}

func (wm *WindowManager) Index(idx int) Window {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	if idx >= len(wm.windows) || idx < 0 {
		return nil
	}
	return wm.windows[idx]
}

func (wm *WindowManager) SelectIndex(idx int) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	if idx >= len(wm.windows) || idx < 0 {
		logrus.Warnf("failed to select window; no window #%d", idx)
		return
	}
	wm.activeIndex = idx
	wm.events.Emit("ui.DIRTY", nil)
}

func (wm *WindowManager) SelectNext() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	idx := wm.activeIndex + 1
	if idx >= len(wm.windows) || idx < 0 {
		idx = 0
	}
	wm.activeIndex = idx
	wm.events.Emit("ui.DIRTY", nil)
}

func (wm *WindowManager) SelectPrev() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	idx := wm.activeIndex - 1
	if idx >= len(wm.windows) || idx < 0 {
		idx = len(wm.windows) - 1
	}
	wm.activeIndex = idx
	wm.events.Emit("ui.DIRTY", nil)
}

// CloseIndex closes a window denoted by tab index.
func (wm *WindowManager) CloseIndex(ch int) {
	if ch == 0 {
		logrus.Warnln("cannot close status window")
		return
	}
	wm.mu.Lock()
	defer wm.mu.Unlock()
	if ch >= len(wm.windows) {
		logrus.Warnf("failed to close window; no window #%d", ch)
		return
	}
	wm.windows = append(wm.windows[:ch], wm.windows[ch+1:]...)
	if ch >= len(wm.windows) {
		wm.activeIndex = len(wm.windows) - 1
	}
	wm.events.Emit("ui.DIRTY", nil)
}

// ScrollTo scrolls the currently active window up relative to the current line.
func (wm *WindowManager) ScrollOffset(offset int) {
	wm.mu.RLock()
	win := wm.windows[wm.activeIndex]
	wm.mu.RUnlock()
	wm.ScrollTo(win.CurrentLine() + offset)
}

// ScrollTo scrolls the currently active window to the given position.
func (wm *WindowManager) ScrollTo(pos int) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	win := wm.windows[wm.activeIndex]
	if pos < 0 {
		pos = 0
	} else if pos >= len(win.Lines()) {
		pos = -1
	}
	win.ScrollTo(pos)
	wm.events.Emit("ui.DIRTY", nil)
}
