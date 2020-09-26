package squirssi

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"code.dopame.me/veonik/squircy3/event"
)

type Window interface {
	io.Writer
	io.StringWriter

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
	UserList() []string
	HasUser(name string) bool
	UpdateUser(name, newNew string) bool
	DeleteUser(name string) bool
}

type bufferedWindow struct {
	name    string
	lines   []string
	current int

	hasUnseen  bool
	hasNotice  bool
	autoScroll bool

	events *event.Dispatcher
	mu     sync.RWMutex
}

func newBufferedWindow(name string, events *event.Dispatcher) bufferedWindow {
	return bufferedWindow{
		name:   name,
		events: events,

		current:    -1,
		autoScroll: true,
	}
}

func (c *bufferedWindow) Title() string {
	return c.name
}

func (c *bufferedWindow) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.events.Emit("ui.DIRTY", map[string]interface{}{
		"name": c.name,
	})
	defer c.mu.Unlock()
	lines := bytes.Split(p, []byte("\n"))
	t := time.Now().Format("[15:04](fg:gray)  ")
	const padding = "       "
	firstWritten := false
	for _, l := range lines {
		if len(l) == 0 {
			continue
		}
		if !firstWritten {
			c.lines = append(c.lines, strings.TrimRight(t+string(l), "\n"))
			firstWritten = true
		} else {
			c.lines = append(c.lines, strings.TrimRight(padding+string(l), "\n"))
		}
	}
	c.hasUnseen = true
	return len(p), nil
}

func (c *bufferedWindow) WriteString(p string) (n int, err error) {
	return c.Write([]byte(p))
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
	return c.hasUnseen && !c.hasNotice
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
		return len(c.lines) - 1
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

type User struct {
	string
	modes string
}

func SomeUser(c string) User {
	u := User{}
	u.string = strings.ReplaceAll(strings.ReplaceAll(c, "@", ""), "+", "")
	r := regexp.MustCompile("[^@+%]")
	u.modes = r.ReplaceAllString(c, "")
	return u
}

func (u User) String() string {
	m := ""
	if u.modes == "@" {
		m = "[@](fg:cyan)"
	} else if u.modes == "+" {
		m = "[+](fg:yellow)"
	}
	return fmt.Sprintf("%s%s", m, u.string)
}

type Channel struct {
	bufferedWindow

	topic        string
	modes        string
	users        []User
	usersIndexed map[string]int
}

func (c *Channel) Topic() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.topic
}

func (c *Channel) Modes() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.modes
}

func (c *Channel) SetUsers(users []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.usersIndexed = make(map[string]int)
	r := make([]User, len(users))
	for i, u := range users {
		r[i] = SomeUser(u)
		c.usersIndexed[r[i].string] = i
	}
	c.users = r
}

func (c *Channel) Users() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t := make([]string, len(c.users))
	for i, u := range c.users {
		t[i] = u.string
	}
	return t
}

func (c *Channel) UserList() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t := make([]User, len(c.users))
	copy(t, c.users)
	sort.SliceStable(t, func(i, j int) bool {
		ui := t[i]
		uj := t[j]
		if ui.modes == "@" && uj.modes != "@" {
			return true
		} else if uj.modes == "@" && ui.modes != "@" {
			return false
		}
		if ui.modes == "+" && uj.modes != "+" {
			return true
		} else if uj.modes == "+" && ui.modes != "+" {
			return false
		}
		return strings.Compare(ui.string, uj.string) < 0
	})
	res := make([]string, len(c.users))
	for i, u := range t {
		res[i] = u.String()
	}
	return res
}

func (c *Channel) userIndex(name string) int {
	if v, ok := c.usersIndexed[name]; ok {
		return v
	}
	return -1
}

func (c *Channel) AddUser(user User) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if idx := c.userIndex(user.string); idx >= 0 {
		c.users[idx].modes = user.modes
		return
	}
	c.usersIndexed[user.string] = len(c.users)
	c.users = append(c.users, user)
}

func (c *Channel) UpdateUser(name, newName string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if idx := c.userIndex(name); idx >= 0 {
		delete(c.usersIndexed, name)
		c.usersIndexed[newName] = idx
		c.users[idx].string = newName
		return true
	}
	return false
}

func (c *Channel) DeleteUser(name string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if idx := c.userIndex(name); idx >= 0 {
		delete(c.usersIndexed, name)
		c.users = append(c.users[:idx], c.users[idx+1:]...)
		return true
	}
	return false
}

func (c *Channel) HasUser(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if idx := c.userIndex(name); idx >= 0 {
		return true
	}
	return false
}

type DirectMessage struct {
	bufferedWindow
}
