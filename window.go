package squirssi

import (
	"io"
	"strings"
	"sync"

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

	// Clears the activity indicator for the window, it it's set.
	Touch()
	// Returns true if the Window has new lines since the last touch.
	HasActivity() bool
}

type WindowWithUserList interface {
	Window
	Users() []string
}

type bufferedWindow struct {
	name    string
	lines   []string
	current int

	hasUnseen bool

	events *event.Dispatcher
	mu     sync.Mutex
}

func newBufferedWindow(name string, events *event.Dispatcher) bufferedWindow {
	return bufferedWindow{
		name:   name,
		events: events,
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
	c.lines = append(c.lines, strings.TrimRight(string(p), "\n"))
	c.hasUnseen = true
	return len(p), nil
}

func (c *bufferedWindow) Touch() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hasUnseen = false
}

func (c *bufferedWindow) HasActivity() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hasUnseen
}

func (c *bufferedWindow) Lines() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lines
}

func (c *bufferedWindow) CurrentLine() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

type Status struct {
	bufferedWindow
}

func (c *Status) Title() string {
	return "status"
}

type Channel struct {
	bufferedWindow

	topic string
	modes string
	users []string
}

func (c *Channel) Users() []string {
	return c.users
}

type DirectMessage struct {
	bufferedWindow
}
