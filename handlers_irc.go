package squirssi

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	"github.com/sirupsen/logrus"
)

func bindIRCHandlers(srv *Server, events *event.Dispatcher) {
	events.Bind("irc.CONNECT", HandleIRCEvent(srv, onIRCConnect))
	events.Bind("irc.DISCONNECT", HandleIRCEvent(srv, onIRCDisconnect))
	events.Bind("irc.PRIVMSG", HandleIRCEvent(srv, onIRCPrivmsg))
	events.Bind("irc.CTCP_ACTION", HandleIRCEvent(srv, onIRCAction))
	events.Bind("irc.JOIN", HandleIRCEvent(srv, onIRCJoin))
	events.Bind("irc.PART", HandleIRCEvent(srv, onIRCPart))
	events.Bind("irc.KICK", HandleIRCEvent(srv, onIRCKick))
	events.Bind("irc.JOIN", HandleIRCEvent(srv, onIRCNames))
	events.Bind("irc.PART", HandleIRCEvent(srv, onIRCNames))
	events.Bind("irc.KICK", HandleIRCEvent(srv, onIRCNames))
	events.Bind("irc.NICK", HandleIRCEvent(srv, onIRCNick))
	events.Bind("irc.353", HandleIRCEvent(srv, onIRC353))
	events.Bind("irc.366", HandleIRCEvent(srv, onIRC366))
	events.Bind("irc.QUIT", HandleIRCEvent(srv, onIRCQuit))
	errorCodes := []string{"irc.401", "irc.403", "irc.404", "irc.405", "irc.406", "irc.407", "irc.408", "irc.421"}
	for _, code := range errorCodes {
		events.Bind(code, HandleIRCEvent(srv, onIRCError))
	}
	whoisCodes := []string{"irc.311", "irc.312", "irc.313", "irc.317", "irc.318", "irc.319", "irc.314", "irc.369"}
	for _, code := range whoisCodes {
		events.Bind(code, HandleIRCEvent(srv, onIRCWhois))
	}
}

type IRCEvent struct {
	Code    string
	Raw     string
	Nick    string // <nick>
	Host    string // <nick>!<usr>@<host>
	Source  string // <host>
	User    string // <usr>
	Target  string
	Message string
	Args    []string
}

func NormalizeIRCEvent(ev *event.Event) *IRCEvent {
	if ev.Data == nil {
		return nil
	}
	return &IRCEvent{
		Code:    ev.Data["Code"].(string),
		Raw:     ev.Data["Raw"].(string),
		Nick:    ev.Data["Nick"].(string),
		Host:    ev.Data["Host"].(string),
		Source:  ev.Data["Source"].(string),
		User:    ev.Data["User"].(string),
		Target:  ev.Data["Target"].(string),
		Message: ev.Data["Message"].(string),
		Args:    ev.Data["Args"].([]string),
	}
}

type IRCEventHandler func(srv *Server, ev *IRCEvent)

func HandleIRCEvent(srv *Server, h IRCEventHandler) event.Handler {
	return event.HandlerFunc(func(ev *event.Event) {
		nev := NormalizeIRCEvent(ev)
		h(srv, nev)
	})
}

func onIRCConnect(srv *Server, _ *IRCEvent) {
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

func onIRCDisconnect(srv *Server, _ *IRCEvent) {
	logrus.Infoln("*** Disconnected")
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.currentNick = ""
}

func onIRCNick(srv *Server, ev *IRCEvent) {
	nick := SomeNick(ev.Nick)
	newNick := ev.Message
	srv.mu.Lock()
	if ev.Nick == srv.currentNick {
		nick = MyNick(srv.currentNick)
		srv.currentNick = newNick
	}
	srv.mu.Unlock()
	WriteNick(srv.WindowManager, nick, newNick)
}

func onIRCKick(srv *Server, ev *IRCEvent) {
	channel := ev.Target
	kicked := SomeNick(ev.Args[1])
	srv.mu.RLock()
	if kicked.string == srv.currentNick {
		kicked = MyNick(srv.currentNick)
	}
	srv.mu.RUnlock()
	if kicked.me {
		go func() {
			<-time.After(2 * time.Second)
			if err := srv.irc.Do(func(conn *irc.Connection) error {
				conn.Join(channel)
				return nil
			}); err != nil {
				logrus.Warnln("failed to rejoin after kick:", err)
			}
		}()
	}
	win := srv.WindowManager.Named(channel)
	if win == nil {
		logrus.Errorln("received kick with no Window:", channel, ev.Message, ev.Nick)
		return
	}
	if kicked.me {
		win.Notice()
	}
	WriteKick(win, kicked, ev.Message)
}

var namesCache = &struct {
	sync.Mutex
	values map[string][]string
}{values: make(map[string][]string)}

func onIRC353(srv *Server, ev *IRCEvent) {
	// NAMES
	chanName := ev.Args[2]
	nicks := strings.Split(ev.Args[3], " ")
	win := srv.WindowManager.Named(chanName)
	if win == nil {
		logrus.Warnln("received NAMES for channel with no window:", chanName)
		return
	}
	namesCache.Lock()
	defer namesCache.Unlock()
	namesCache.values[chanName] = append(namesCache.values[chanName], nicks...)
}

func onIRC366(srv *Server, ev *IRCEvent) {
	// END NAMES
	chanName := ev.Args[1]
	win := srv.WindowManager.Named(chanName)
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

func onIRCError(_ *Server, ev *IRCEvent) {
	var kind string
	if len(ev.Args) > 1 {
		kind = ev.Args[1]
	} else {
		kind = ev.Target
	}
	logrus.Errorf("%s: %s", kind, ev.Message)
}

func onIRCWhois(srv *Server, ev *IRCEvent) {
	kind := ev.Args[1]
	data := ev.Args[2:]
	if _, err := srv.WindowManager.status.Write([]byte(fmt.Sprintf("WHOIS %s => %s", kind, strings.Join(data, " ")))); err != nil {
		logrus.Warnln("failed to write whois result to status:", err)
	}
}

func onIRCNames(srv *Server, ev *IRCEvent) {
	if ev.Code == "PART" || ev.Code == "KICK" {
		srv.mu.RLock()
		myNick := srv.currentNick
		srv.mu.RUnlock()
		if ev.Nick == myNick {
			// dont bother trying to get names when we are the one leaving
			return
		}
	}
	target := ev.Target
	if strings.HasPrefix(target, "#") {
		if err := srv.irc.Do(func(conn *irc.Connection) error {
			conn.SendRawf("NAMES :%s", target)
			return nil
		}); err != nil {
			logrus.Warnln("failed to run NAMES on user change:", err)
		}
	}
}

func onIRCJoin(srv *Server, ev *IRCEvent) {
	target := ev.Target
	win := srv.WindowManager.Named(target)
	srv.mu.RLock()
	myNick := srv.currentNick
	srv.mu.RUnlock()
	if win == nil {
		ch := &Channel{
			bufferedWindow: newBufferedWindow(target, srv.events),
			users:          []string{},
		}
		srv.WindowManager.Append(ch)
		win = ch
	}
	WriteJoin(win, Nick{ev.Nick, ev.Nick == myNick})
}

func onIRCPart(srv *Server, ev *IRCEvent) {
	target := ev.Target
	nick := SomeNick(ev.Nick)
	win := srv.WindowManager.Named(target)
	srv.mu.RLock()
	if ev.Nick == srv.currentNick {
		nick = MyNick(srv.currentNick)
	}
	srv.mu.RUnlock()
	if win == nil {
		if !nick.me {
			// dont bother logging if we are the ones leaving
			logrus.Errorln("received message with no Window:", target, ev.Message, nick)
		}
		return
	}
	WritePart(win, nick, ev.Message)
}

func onIRCAction(srv *Server, ev *IRCEvent) {
	direct := false
	target := ev.Target
	nick := ev.Nick
	srv.mu.RLock()
	myNick := MyNick(srv.currentNick)
	srv.mu.RUnlock()
	if target == myNick.string {
		// its a direct message!
		direct = true
		target = nick
	}
	win := srv.WindowManager.Named(target)
	if win == nil {
		if !direct {
			logrus.Warnln("received action message with no Window:", target, ev.Message, nick)
			return
		} else {
			ch := &DirectMessage{bufferedWindow: newBufferedWindow(target, srv.events)}
			srv.WindowManager.Append(ch)
			win = ch
		}
	}
	msg := SomeMessage(ev.Message, myNick)
	WriteAction(win, SomeNick(nick), msg)
}

func onIRCPrivmsg(srv *Server, ev *IRCEvent) {
	direct := false
	target := ev.Target
	nick := ev.Nick
	srv.mu.RLock()
	myNick := MyNick(srv.currentNick)
	srv.mu.RUnlock()
	if target == myNick.string {
		// its a direct message!
		direct = true
		target = nick
	}
	win := srv.WindowManager.Named(target)
	if win == nil {
		if !direct {
			logrus.Warnln("received message with no Window:", target, ev.Message, nick)
			return
		} else {
			ch := &DirectMessage{bufferedWindow: newBufferedWindow(target, srv.events)}
			srv.WindowManager.Append(ch)
			win = ch
		}
	}
	msg := SomeMessage(ev.Message, myNick)
	WritePrivmsg(win, SomeNick(nick), msg)
}

func onIRCQuit(srv *Server, ev *IRCEvent) {
	nick := SomeNick(ev.Nick)
	message := ev.Message
	srv.mu.RLock()
	if ev.Nick == srv.currentNick {
		nick = MyNick(srv.currentNick)
	}
	srv.mu.RUnlock()
	WriteQuit(srv.WindowManager, nick, message)
}
