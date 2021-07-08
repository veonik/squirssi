package squirssi

import (
	"strings"

	"code.dopame.me/veonik/squircy3/event"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squirssi/widget"
)

func bindUIHandlers(srv *Server, events *event.Dispatcher) {
	events.Bind("ui.DIRTY", HandleUIEvent(srv, onUIDirty))
}

type UIHandler func(*Server, *event.Event)

func HandleUIEvent(srv *Server, fn UIHandler) event.Handler {
	return event.HandlerFunc(func(ev *event.Event) {
		fn(srv, ev)
	})
}

func onUIDirty(srv *Server, _ *event.Event) {
	srv.Update()
	srv.Render()
}

// onUIKeyPress handles keyboard input from termui.
// Not a regular event handler but instead called before the actual
// ui.KEYPRESS event is emitted. This is done to avoid extra lag between
// pressing a key and seeing the UI react.
func onUIKeyPress(srv *Server, key string) {
	if key != "<Tab>" {
		srv.tabber.Clear()
	}
	switch key {
	case "<C-c>":
		srv.inputTextBox.Append(string(rune(0x03)))
		srv.RenderOnly(InputTextBox)
	case "<C-u>":
		srv.inputTextBox.Append(string(rune(0x1F)))
		srv.RenderOnly(InputTextBox)
	case "<C-b>":
		srv.inputTextBox.Append(string(rune(0x02)))
		srv.RenderOnly(InputTextBox)
	case "<M-b>":
		srv.inputTextBox.CursorPrevWord()
		srv.RenderOnly(InputTextBox)
	case "<M-f>":
		srv.inputTextBox.CursorNextWord()
		srv.RenderOnly(InputTextBox)
	case "<Home>":
		srv.inputTextBox.CursorStartLine()
		srv.RenderOnly(InputTextBox)
	case "<End>":
		srv.inputTextBox.CursorEndLine()
		srv.RenderOnly(InputTextBox)
	case "<PageUp>":
		srv.mu.RLock()
		h := srv.pageSize - 2
		srv.mu.RUnlock()
		srv.windows.ScrollOffset(-h)
	case "<PageDown>":
		srv.mu.RLock()
		h := srv.pageSize - 2
		srv.mu.RUnlock()
		srv.windows.ScrollOffset(h)
	case "<M-<PageUp>>":
		srv.mu.Lock()
		h := srv.pageSize - 2
		srv.userListPane.SelectedRow -= h
		srv.mu.Unlock()
		srv.events.Emit("ui.DIRTY", nil)
	case "<M-<PageDown>>":
		srv.mu.Lock()
		h := srv.pageSize - 2
		if srv.userListPane.SelectedRow == 0 {
			srv.userListPane.SelectedRow = h
		}
		srv.userListPane.SelectedRow += h
		srv.mu.Unlock()
		srv.events.Emit("ui.DIRTY", nil)
	case "<Space>":
		srv.inputTextBox.Append(" ")
		srv.RenderOnly(InputTextBox)
	case "<Backspace>":
		srv.inputTextBox.Backspace()
		srv.RenderOnly(InputTextBox)
	case "<Delete>":
		srv.inputTextBox.DeleteNext()
		srv.RenderOnly(InputTextBox)
	case "<C-5>":
		srv.windows.SelectNext()
	case "<Escape>":
		srv.windows.SelectPrev()
	case "<Up>":
		win := srv.windows.Active()
		if win == nil {
			return
		}
		cur := srv.inputTextBox.Consume()
		if cur.Text != "" {
			srv.history.Insert(win, cur)
		}
		msg := srv.history.Previous(win)
		srv.inputTextBox.Set(msg)
		srv.RenderOnly(InputTextBox)
	case "<Down>":
		win := srv.windows.Active()
		if win == nil {
			return
		}
		cur := srv.inputTextBox.Consume()
		if cur.Text != "" {
			srv.history.Insert(win, cur)
		}
		msg := srv.history.Next(win)
		srv.inputTextBox.Set(msg)
		srv.RenderOnly(InputTextBox)
	case "<Left>":
		srv.inputTextBox.CursorPrev()
		srv.RenderOnly(InputTextBox)
	case "<Right>":
		srv.inputTextBox.CursorNext()
		srv.RenderOnly(InputTextBox)
	case "<Tab>":
		win := srv.windows.Active()
		if ch, ok := win.(*Channel); ok {
			var tabbed string
			if srv.tabber.Active() {
				tabbed = srv.tabber.Tab()
			} else {
				tabbed = srv.tabber.Reset(srv.inputTextBox.Peek(), ch)
			}
			srv.inputTextBox.Set(widget.ModedText{Kind: srv.inputTextBox.Mode(), Text: tabbed})
		}
		srv.RenderOnly(InputTextBox)
	case "<Enter>":
		in := srv.inputTextBox.Consume()
		active := srv.windows.ActiveIndex()
		channel := srv.windows.Active()
		if channel == nil {
			return
		}
		if len(in.Text) == 0 {
			// render anyway incase the textbox mode was changed
			srv.RenderOnly(MainWindow, InputTextBox)
			return
		}
		defer srv.RenderOnly(InputTextBox)
		defer srv.history.Append(channel, in)
		switch in.Kind {
		case widget.ModeCommand:
			args := strings.Split(in.Text, " ")
			c := args[0]
			if cmd, ok := builtIns[c]; ok {
				cmd(srv, args)
			} else {
				logrus.Warnln("no command named:", c)
			}
		case widget.ModeMessage:
			if active == 0 {
				// status window doesn't accept messages
				return
			}
			msgTarget(srv, []string{"msg", channel.Title(), in.Text})
		}

	default:
		if len(key) != 1 {
			logrus.Debugln("received unhandled keypress:", key)
			return
		}
		if key == "/" && srv.inputTextBox.Pos() == 0 {
			srv.inputTextBox.ToggleMode()
		} else {
			srv.inputTextBox.Append(key)
		}
		srv.RenderOnly(InputTextBox)
	}
}
