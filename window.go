package squirssi

import (
	"io"
	"strings"
	"sync"
	"time"

	"code.dopame.me/veonik/squircy3/event"
)

type Window interface {
	io.Writer

	// Title of the Window.
	Title() string
	// Contents of the Window, separated by line.
	Lines() []string

	// The bottom-most visible line number, or negative to indicate
	// the window is pinned to the end of input.
	CurrentLine() int
	// Set the current line to pos. Set to negative to pin to the end of input.
	ScrollTo(pos int)
	AutoScroll() bool

	// Clears the activity indicator for the window, it it's set.
	Touch()
	// Returns true if the Window has new lines since the last touch.
	HasActivity() bool
	// Set notice indicator for this Window.
	Notice()
	// Returns true if the Window has new lines considered important since last touch.
	HasNotice() bool
}

type WindowWithUserList interface {
	Window
	Users() []string
	HasUser(name string) bool
}

type bufferedWindow struct {
	name    string
	lines   []string
	current int

	hasUnseen bool
	hasNotice bool
	autoScroll bool

	events *event.Dispatcher
	mu     sync.RWMutex
}


func newBufferedWindow(name string, events *event.Dispatcher) bufferedWindow {
	return bufferedWindow{
		name:    name,
		events:  events,

		current: -1,
		autoScroll: true,
	}
}

func (c *bufferedWindow) Title() string {
	return c.name
}

func (c *bufferedWindow) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer c.events.Emit("ui.DIRTY", map[string]interface{}{
		"name": c.name,
	})
	t := time.Now().Format("[15:04](fg:gray)  ")
	c.lines = append(c.lines, strings.TrimRight(t+string(p), "\n"))
	c.hasUnseen = true
	return len(p), nil
}

func (c *bufferedWindow) Touch() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hasUnseen = false
	c.hasNotice = false
}

func (c *bufferedWindow) Notice() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hasNotice = true
}

func (c *bufferedWindow) HasNotice() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hasNotice
}

func (c *bufferedWindow) HasActivity() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hasUnseen
}

func (c *bufferedWindow) AutoScroll() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.autoScroll
}

func (c *bufferedWindow) Lines() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lines
}

func (c *bufferedWindow) CurrentLine() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.autoScroll {
		return len(c.lines)-1
	}
	return c.current
}

func (c *bufferedWindow) ScrollTo(pos int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = pos
	if pos < 0 {
		c.autoScroll = true
	} else {
		c.autoScroll = false
	}
}

type StatusWindow struct {
	bufferedWindow
}

func (c *StatusWindow) Title() string {
	return "status"
}

type Channel struct {
	bufferedWindow

	topic string
	modes string
	users []string
}

func (c *Channel) Users() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.users
}

func (c *Channel) HasUser(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, c := range c.users {
		if strings.ReplaceAll(strings.ReplaceAll(c, "@", ""), "+", "") == name {
			return true
		}
	}
	return false
}

type DirectMessage struct {
	bufferedWindow
}
