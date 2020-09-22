package squirssi

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	"github.com/sirupsen/logrus"
)

func (srv *Server) bind() {
	srv.events.Bind("ui.DIRTY", event.HandlerFunc(srv.onUIDirty))

	srv.events.Bind("irc.CONNECT", event.HandlerFunc(srv.onIRCConnect))
	srv.events.Bind("irc.DISCONNECT", event.HandlerFunc(srv.onIRCDisconnect))
	srv.events.Bind("irc.PRIVMSG", event.HandlerFunc(srv.onIRCPrivmsg))
	srv.events.Bind("irc.JOIN", event.HandlerFunc(srv.onIRCJoin))
	srv.events.Bind("irc.PART", event.HandlerFunc(srv.onIRCPart))
	srv.events.Bind("irc.KICK", event.HandlerFunc(srv.onIRCKick))
	srv.events.Bind("irc.JOIN", event.HandlerFunc(srv.onIRCNames))
	srv.events.Bind("irc.PART", event.HandlerFunc(srv.onIRCNames))
	srv.events.Bind("irc.KICK", event.HandlerFunc(srv.onIRCNames))
	srv.events.Bind("irc.NICK", event.HandlerFunc(srv.onIRCNick))
	srv.events.Bind("irc.353", event.HandlerFunc(srv.onIRC353))
	srv.events.Bind("irc.366", event.HandlerFunc(srv.onIRC366))
	errorCodes := []string{"irc.401", "irc.403", "irc.404", "irc.405", "irc.406", "irc.407", "irc.408", "irc.421"}
	for _, code := range errorCodes {
		srv.events.Bind(code, event.HandlerFunc(srv.onIRCError))
	}
	whoisCodes := []string{"irc.311", "irc.312", "irc.313", "irc.317", "irc.318", "irc.319", "irc.314", "irc.369"}
	for _, code := range whoisCodes {
		srv.events.Bind(code, event.HandlerFunc(srv.onIRCWhois))
	}
}

func (srv *Server) onUIDirty(_ *event.Event) {
	srv.Update()
	srv.Render()
}

// onUIKeyPress handles keyboard input from termui.
// Not a regular event handler but instead called before the actual
// ui.KEYPRESS event is emitted. This is done to avoid extra lag between
// pressing a key and seeing the UI react.
func (srv *Server) onUIKeyPress(key string) {
	switch key {
	case "<C-c>":
		srv.Close()
		os.Exit(0)
		return
	case "<PageUp>":
		srv.ScrollPageUp()
	case "<PageDown>":
		srv.ScrollPageDown()
	case "<Home>":
		srv.ScrollTop()
	case "<End>":
		srv.ScrollBottom()
	case "<Space>":
		srv.mu.Lock()
		srv.inputTextBox.Append(" ")
		srv.mu.Unlock()
		srv.RenderOnly(InputTextBox)
	case "<Backspace>":
		srv.mu.Lock()
		srv.inputTextBox.Backspace()
		srv.mu.Unlock()
		srv.RenderOnly(InputTextBox)
	case "<C-5>":
		srv.mu.Lock()
		srv.statusBar.FocusRight()
		srv.mu.Unlock()
		srv.Update()
		srv.Render()
	case "<Escape>":
		srv.mu.Lock()
		srv.statusBar.FocusLeft()
		srv.mu.Unlock()
		srv.Update()
		srv.Render()
	case "<Tab>":
		srv.mu.Lock()
		channel := srv.windows[srv.statusBar.ActiveTabIndex]
		srv.mu.Unlock()
		if ch, ok := channel.(WindowWithUserList); ok {
			msg := srv.inputTextBox.Peek()
			parts := strings.Split(msg, " ")
			match := parts[len(parts)-1]
			extra := ""
			if match == parts[0] {
				extra = ": "
			}
			for _, u := range ch.Users() {
				nick := strings.ReplaceAll(strings.ReplaceAll(u, "@", ""), "+", "")
				if strings.HasPrefix(nick, match) {
					srv.inputTextBox.Append(strings.Replace(nick, match, "", 1) + extra)
					break
				}
			}
		}
		srv.RenderOnly(InputTextBox)
	case "<Enter>":
		srv.mu.Lock()
		in := srv.inputTextBox.Consume()
		if srv.inputTextBox.Mode() == ModeCommand {
			srv.inputTextBox.ToggleMode()
		}
		active := srv.statusBar.ActiveTabIndex
		channel := srv.windows[active]
		myNick := srv.currentNick
		srv.mu.Unlock()
		if channel == nil {
			return
		}
		if len(in.Text) == 0 {
			// render anyway incase the textbox mode was changed
			srv.RenderOnly(MainWindow)
			return
		}
		switch in.Kind {
		case ModeCommand:
			args := strings.Split(in.Text, " ")
			c := args[0]
			if cmd, ok := builtIns[c]; ok {
				cmd(srv, args)
			}
		case ModeMessage:
			if active == 0 {
				// status window doesn't accept messages
				return
			}
			if err := srv.irc.Do(func(c *irc.Connection) error {
				c.Privmsg(channel.Title(), in.Text)
				return nil
			}); err != nil {
				logrus.Warnln("failed to send message:", err)
			}
			if _, err := channel.Write([]byte("<" + myNick + "> " + in.Text)); err != nil {
				logrus.Warnln("failed to write message:", err)
			}
		}

	default:
		if len(key) != 1 {
			// a single key resulted in more than one character, probably not a regular char
			return
		}
		srv.mu.Lock()
		if key == "/" && srv.inputTextBox.Len() == 0 {
			srv.inputTextBox.ToggleMode()
		} else {
			srv.inputTextBox.Append(key)
		}
		srv.mu.Unlock()
		srv.RenderOnly(InputTextBox)
	}
}

func (srv *Server) onIRCConnect(_ *event.Event) {
	err := srv.irc.Do(func(conn *irc.Connection) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		srv.currentNick = conn.GetNick()
		return nil
	})
	if err != nil {
		logrus.Warnln("failed to set current nick:", err)
	}
}

func (srv *Server) onIRCDisconnect(_ *event.Event) {
	logrus.Infoln("*** Disconnected")
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.currentNick = ""
}

func (srv *Server) onIRCNick(ev *event.Event) {
	logrus.Infoln("irc.NICK:", ev.Data)
	nick := ev.Data["Nick"].(string)
	newNick := ev.Data["Message"].(string)
	srv.mu.Lock()
	if nick == srv.currentNick {
		srv.currentNick = newNick
	}
	wins := make([]Window, len(srv.windows))
	copy(wins, srv.windows)
	srv.mu.Unlock()
	for _, win := range wins {
		if win.Title() == nick {
			// direct message with nick, update title and print there
			if dm, ok := win.(*DirectMessage); ok {
				dm.mu.Lock()
				dm.name = nick
				dm.mu.Unlock()
			}
			if _, err := win.Write([]byte(fmt.Sprintf("*** %s is now known as %s", nick, newNick))); err != nil {
				logrus.Warnln("failed to write nick change:", err)
			}
		} else if ch, ok := win.(*Channel); ok {
			hasNick := false
			ch.mu.Lock()
			for i, u := range ch.users {
				if strings.ReplaceAll(strings.ReplaceAll(u, "@", ""), "+", "") == nick {
					ch.users[i] = newNick
					hasNick = true
					break
				}
			}
			ch.mu.Unlock()
			if hasNick {
				if _, err := win.Write([]byte(fmt.Sprintf("*** %s is now known as %s", nick, newNick))); err != nil {
					logrus.Warnln("failed to write nick change:", err)
				}
			}
		}
	}
}

func (srv *Server) onIRCKick(ev *event.Event) {
	args := ev.Data["Args"].([]string)
	channel := ev.Data["Target"].(string)
	kicked := args[1]
	srv.mu.Lock()
	myNick := srv.currentNick
	srv.mu.Unlock()
	if kicked == myNick {
		go func() {
			<-time.After(2 * time.Second)
			if err := srv.irc.Do(func(conn *irc.Connection) error {
				conn.Join(channel)
				return nil
			}); err != nil {
				logrus.Warnln("failed to rejoin after kick:", err)
			}
		}()
		return
	}
	win := srv.WindowNamed(channel)
	if win == nil {
		logrus.Errorln("received kick with no Window:", channel, ev.Data["Message"], ev.Data["Nick"])
		return
	}
	if _, err := win.Write([]byte(fmt.Sprintf("*** %s got kicked from %s (%s)", kicked, channel, ev.Data["Message"].(string)))); err != nil {
		logrus.Warnln("failed to write message:", err)
	}
}

var namesCache = &struct {
	sync.Mutex
	values map[string][]string
}{values: make(map[string][]string)}

func (srv *Server) onIRC353(ev *event.Event) {
	// NAMES
	args := ev.Data["Args"].([]string)
	chanName := args[2]
	nicks := strings.Split(args[3], " ")
	win := srv.WindowNamed(chanName)
	if win == nil {
		logrus.Warnln("received NAMES for channel with no window:", chanName)
		return
	}
	namesCache.Lock()
	defer namesCache.Unlock()
	namesCache.values[chanName] = append(namesCache.values[chanName], nicks...)
}

func (srv *Server) onIRC366(ev *event.Event) {
	// END NAMES
	args := ev.Data["Args"].([]string)
	chanName := args[1]
	win := srv.WindowNamed(chanName)
	if win == nil {
		logrus.Warnln("received END NAMES for channel with no window:", chanName)
		return
	}
	ch, ok := win.(*Channel)
	if !ok {
		logrus.Warnln("received END NAMES for a non channel:", chanName)
		return
	}
	namesCache.Lock()
	defer namesCache.Unlock()
	ch.mu.Lock()
	ch.users = namesCache.values[chanName]
	ch.mu.Unlock()
	delete(namesCache.values, chanName)
}

func (srv *Server) onIRCError(ev *event.Event) {
	args := ev.Data["Args"].([]string)
	var kind string
	if len(args) > 1 {
		kind = args[1]
	} else {
		kind = ev.Data["Target"].(string)
	}
	logrus.Errorf("%s: %s", kind, ev.Data["Message"])
}

func (srv *Server) onIRCWhois(ev *event.Event) {
	args := ev.Data["Args"].([]string)
	kind := args[1]
	data := args[2:]
	if _, err := srv.status.Write([]byte(fmt.Sprintf("WHOIS %s => %s", kind, strings.Join(data, " ")))); err != nil {
		logrus.Warnln("failed to write whois result to status:", err)
	}
}

func (srv *Server) onIRCNames(ev *event.Event) {
	if ev.Name == "irc.PART" || ev.Name == "irc.KICK" {
		srv.mu.Lock()
		myNick := srv.currentNick
		srv.mu.Unlock()
		nick := ev.Data["Nick"].(string)
		if nick == myNick {
			// dont bother trying to get names when we are the one leaving
			return
		}
	}
	target := ev.Data["Target"].(string)
	if strings.HasPrefix(target, "#") {
		if err := srv.irc.Do(func(conn *irc.Connection) error {
			conn.SendRawf("NAMES :%s", target)
			return nil
		}); err != nil {
			logrus.Warnln("failed to run NAMES on user change:", err)
		}
	}
}

func (srv *Server) onIRCJoin(ev *event.Event) {
	target := ev.Data["Target"].(string)
	win := srv.WindowNamed(target)
	if win == nil {
		srv.mu.Lock()
		ch := &Channel{
			bufferedWindow: newBufferedWindow(target, srv.events),
			users:          []string{srv.currentNick},
		}
		srv.windows = append(srv.windows, ch)
		srv.mu.Unlock()
		win = ch
	}
	if _, err := win.Write([]byte(fmt.Sprintf("*** %s joined %s", ev.Data["Nick"], win.Title()))); err != nil {
		logrus.Warnln("%s: failed to write join message:", err)
	}
}

func (srv *Server) onIRCPart(ev *event.Event) {
	var win Window
	var ch int
	target := ev.Data["Target"].(string)
	nick := ev.Data["Nick"].(string)
	srv.mu.Lock()
	for i, w := range srv.windows {
		if w.Title() == target {
			win = w
			ch = i
			break
		}
	}
	myNick := srv.currentNick
	srv.mu.Unlock()
	if win == nil {
		if nick != myNick {
			// dont bother logging if we are the ones leaving
			logrus.Errorln("received message with no Window:", target, ev.Data["Message"], ev.Data["Nick"])
		}
		return
	}
	if nick == myNick {
		// its me!
		srv.CloseWindow(ch)
	}
	if _, err := win.Write([]byte(fmt.Sprintf("*** %s parted %s", nick, win.Title()))); err != nil {
		logrus.Warnln("%s: failed to write part message:", err)
	}
}

func (srv *Server) onIRCPrivmsg(ev *event.Event) {
	var win Window
	direct := false
	target := ev.Data["Target"].(string)
	nick := ev.Data["Nick"].(string)
	srv.mu.Lock()
	if target == srv.currentNick {
		// its a direct message!
		direct = true
		target = nick
	}
	for _, w := range srv.windows {
		if w.Title() == target {
			win = w
			break
		}
	}
	srv.mu.Unlock()
	if win == nil {
		if !direct {
			logrus.Warnln("received message with no Window:", target, ev.Data["Message"], nick)
			return
		} else {
			srv.mu.Lock()
			ch := &DirectMessage{bufferedWindow: newBufferedWindow(target, srv.events)}
			srv.windows = append(srv.windows, ch)
			win = ch
			srv.mu.Unlock()
		}
	}
	if _, err := win.Write([]byte(fmt.Sprintf("<%s> %s", nick, ev.Data["Message"]))); err != nil {
		logrus.Warnln("error writing to Window:", err)
	}
}
