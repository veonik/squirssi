package squirssi

import (
	"strings"
	"sync"
)

type TabCompleter struct {
	active bool

	input   string
	match   string
	matches []string
	extra   string
	pos     int

	mu sync.Mutex
}

func NewTabCompleter() *TabCompleter {
	return &TabCompleter{}
}

func (t *TabCompleter) Active() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.active
}

func (t *TabCompleter) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.active = false
}

func (t *TabCompleter) Reset(input string, window Window) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	parts := strings.Split(input, " ")
	t.match = parts[len(parts)-1]
	t.extra = ""
	if t.match == parts[0] {
		t.extra = ": "
	}
	var m []string
	if wul, ok := window.(WindowWithUserList); ok {
		for _, v := range wul.Users() {
			if strings.HasPrefix(v, t.match) {
				m = append(m, v+t.extra)
			}
		}
	}
	// put the match on the end of the stack so we can tab back to it.
	m = append(m, t.match)
	t.input = input
	t.matches = m
	t.pos = 0
	t.active = true
	return strings.Replace(input, t.match, t.matches[t.pos], 1)
}

func (t *TabCompleter) Tab() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.active {
		return ""
	}
	t.pos++
	if t.pos >= len(t.matches) {
		t.pos = 0
	}
	return strings.Replace(t.input, t.match, t.matches[t.pos], 1)
}
