package squirssi

import (
	"sync"

	"code.dopame.me/veonik/squirssi/widget"
)

type HistoryManager struct {
	histories map[Window][]widget.ModedText
	cursors   map[Window]int

	mu sync.Mutex
}

func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		histories: make(map[Window][]widget.ModedText),
		cursors:   make(map[Window]int),
	}
}

func (hm *HistoryManager) Append(win Window, input widget.ModedText) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.cursors[win] = len(hm.histories[win])
	hm.append(win, input)
	hm.cursors[win] = len(hm.histories[win])
}

func (hm *HistoryManager) Insert(win Window, input widget.ModedText) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	if hm.current(win) == input {
		return
	}
	hm.append(win, input)
}

func (hm *HistoryManager) append(win Window, input widget.ModedText) {
	hm.histories[win] = append(append(append([]widget.ModedText{}, hm.histories[win][:hm.cursors[win]]...), input), hm.histories[win][hm.cursors[win]:]...)
}

func (hm *HistoryManager) current(win Window) widget.ModedText {
	if hm.cursors[win] < 0 {
		hm.cursors[win] = 0
	}
	if hm.cursors[win] >= len(hm.histories[win]) {
		hm.cursors[win] = len(hm.histories[win])
		return widget.ModedText{}
	}
	return hm.histories[win][hm.cursors[win]]
}

func (hm *HistoryManager) Current(win Window) widget.ModedText {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	return hm.current(win)
}

func (hm *HistoryManager) Previous(win Window) widget.ModedText {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.cursors[win] -= 1
	res := hm.current(win)
	return res
}

func (hm *HistoryManager) Next(win Window) widget.ModedText {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.cursors[win] += 1
	res := hm.current(win)
	return res
}
