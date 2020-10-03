package squirssi

import (
	"math"
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
		srv.inputTextBox.Append(string(0x03))
		srv.RenderOnly(InputTextBox)
	case "<C-u>":
		srv.inputTextBox.Append(string(0x1F))
		srv.RenderOnly(InputTextBox)
	case "<C-b>":
		srv.inputTextBox.Append(string(0x02))
		srv.RenderOnly(InputTextBox)
	case "<PageUp>":
		srv.mu.RLock()
		h := srv.pageHeight - 2
		srv.mu.RUnlock()
		srv.wm.ScrollOffset(-h)
	case "<PageDown>":
		srv.mu.RLock()
		h := srv.pageHeight - 2
		srv.mu.RUnlock()
		srv.wm.ScrollOffset(h)
	case "<Home>":
		srv.wm.ScrollTo(0)
	case "<End>":
		srv.wm.ScrollTo(math.MaxInt32)
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
		srv.wm.SelectNext()
	case "<Escape>":
		srv.wm.SelectPrev()
	case "<Up>":
		win := srv.wm.Active()
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
		win := srv.wm.Active()
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
		win := srv.wm.Active()
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
		active := srv.wm.ActiveIndex()
		channel := srv.wm.Active()
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
		if key == "/" && srv.inputTextBox.Len() == 0 {
			srv.inputTextBox.ToggleMode()
		} else {
			srv.inputTextBox.Append(key)
		}
		srv.RenderOnly(InputTextBox)
	}
}
