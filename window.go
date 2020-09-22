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
		name:    name,
		events:  events,
		current: -1,
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
	t := time.Now().Format("[15:04:05] ")
	c.lines = append(c.lines, strings.TrimRight(t+string(p), "\n"))
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

func (c *bufferedWindow) ScrollTo(pos int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = pos
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
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.users
}

type DirectMessage struct {
	bufferedWindow
}
