package squirssi

import (
	"strings"
	"sync"
	"time"

	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	"github.com/sirupsen/logrus"
	irc2 "github.com/thoj/go-ircevent"
)

func bindIRCHandlers(srv *Server, events *event.Dispatcher) {
	events.Bind("irc.CONNECT", HandleIRCEvent(srv, onIRCConnect))
	events.Bind("irc.DISCONNECT", HandleIRCEvent(srv, onIRCDisconnect))
	events.Bind("irc.PRIVMSG", HandleIRCEvent(srv, onIRCPrivmsg))
	events.Bind("irc.NOTICE", HandleIRCEvent(srv, onIRCNotice))
	events.Bind("irc.CTCP_ACTION", HandleIRCEvent(srv, onIRCAction))
	events.Bind("irc.JOIN", HandleIRCEvent(srv, onIRCJoin))
	events.Bind("irc.PART", HandleIRCEvent(srv, onIRCPart))
	events.Bind("irc.KICK", HandleIRCEvent(srv, onIRCKick))
	events.Bind("irc.JOIN", HandleIRCEvent(srv, onIRCNames))
	events.Bind("irc.PART", HandleIRCEvent(srv, onIRCNames))
	events.Bind("irc.KICK", HandleIRCEvent(srv, onIRCNames))
	events.Bind("irc.NICK", HandleIRCEvent(srv, onIRCNick))
	events.Bind("irc.433", HandleIRCEvent(srv, onIRC433))
	events.Bind("irc.353", HandleIRCEvent(srv, onIRC353))
	events.Bind("irc.366", HandleIRCEvent(srv, onIRC366))
	events.Bind("irc.QUIT", HandleIRCEvent(srv, onIRCQuit))
	events.Bind("irc.MODE", HandleIRCEvent(srv, onIRCMode))
	events.Bind("irc.324", HandleIRCEvent(srv, onIRC324))
	events.Bind("irc.332", HandleIRCEvent(srv, onIRC332))
	events.Bind("irc.331", HandleIRCEvent(srv, onIRC331))
	events.Bind("irc.TOPIC", HandleIRCEvent(srv, onIRCTopic))
	errorCodes := []string{"irc.401", "irc.403", "irc.404", "irc.405", "irc.406", "irc.407", "irc.408", "irc.421"}
	for _, code := range errorCodes {
		events.Bind(code, HandleIRCEvent(srv, onIRCError))
	}
	whoisCodes := []string{"irc.311", "irc.312", "irc.313", "irc.317", "irc.318", "irc.319", "irc.314", "irc.369"}
	for _, code := range whoisCodes {
		events.Bind(code, HandleIRCEvent(srv, onIRCWhois))
	}
	miscCodes := []string{
		"irc.001", "irc.002", "irc.003", "irc.250", "irc.251",
		"irc.252", "irc.253", "irc.254", "irc.255", "irc.265",
		"irc.266", "irc.375", "irc.372", "irc.376",
	}
	for _, code := range miscCodes {
		events.Bind(code, HandleIRCEvent(srv, onIRCMessage))
	}
	events.Bind("debug.IRC", event.HandlerFunc(handleIRCDebugEvent))
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

func normalizeDebugEvent(ev *event.Event) *IRCEvent {
	if ev.Data == nil {
		return nil
	}
	v, ok := ev.Data["source"].(*irc2.Event)
	if !ok {
		return nil
	}
	return &IRCEvent{
		Code:    v.Code,
		Raw:     v.Raw,
		Nick:    v.Nick,
		Host:    v.Host,
		Source:  v.Source,
		User:    v.User,
		Target:  v.Arguments[0],
		Message: v.Message(),
		Args:    v.Arguments,
	}
}

var debugIgnore = map[string]struct{}{
	"CTCP_ACTION":     {},
	"CTCP_TIME":       {},
	"CTCP_VERSION":    {},
	"CTCP_USERINFO":   {},
	"CTCP_CLIENTINFO": {},
	"CTCP_PING":       {},

	"PRIVMSG": {},
	"NOTICE":  {},
	"TOPIC":   {},
	"MODE":    {},
	"KICK":    {},
	"NICK":    {},
	"QUIT":    {},
	"JOIN":    {},
	"PART":    {},

	"366": {},
	"353": {},
	"324": {},
	"331": {},
	"332": {},
	"001": {},
	"002": {},
	"003": {},
	"251": {},
	"252": {},
	"253": {},
	"254": {},
	"255": {},
	"250": {},
	"265": {},
	"266": {},
	"375": {},
	"372": {},
	"376": {},
	"433": {},
}

func handleIRCDebugEvent(ev *event.Event) {
	nev := normalizeDebugEvent(ev)
	if _, ok := debugIgnore[nev.Code]; ok {
		return
	}
	logrus.Debugf("irc.%s - T(%s) N(%s) => %s", nev.Code, nev.Target, nev.Nick, strings.Join(nev.Args[1:], " "))
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

func onIRC324(srv *Server, ev *IRCEvent) {
	modes := strings.Join(ev.Args[2:], " ")
	win := srv.wm.Named(ev.Args[1])
	blankBefore := false
	if ch, ok := win.(*Channel); ok {
		ch.mu.Lock()
		if ch.modes == "" {
			blankBefore = true
		}
		ch.modes = modes
		ch.mu.Unlock()
	}
	if win == nil {
		logrus.Warnln("received MODES for something without a window:", ev.Args[1], ev.Args[2:])
		return
	}
	// dont write the message the first time we get modes since that is done automatically on join
	if !blankBefore {
		WriteModes(win, modes)
	}
}

func onIRC331(srv *Server, ev *IRCEvent) {
	target := ev.Args[1]
	win := srv.wm.Named(target)
	if ch, ok := win.(*Channel); ok {
		ch.mu.Lock()
		ch.topic = ""
		ch.mu.Unlock()
		Write331(win)
	}
}

func onIRC332(srv *Server, ev *IRCEvent) {
	target := ev.Args[1]
	win := srv.wm.Named(target)
	if ch, ok := win.(*Channel); ok {
		topic := strings.Join(ev.Args[2:], " ")
		ch.mu.Lock()
		ch.topic = topic
		ch.mu.Unlock()
		Write332(win, topic)
	}
}

func onIRCConnect(srv *Server, _ *IRCEvent) {
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		srv.setCurrentNick(conn.GetNick())
		conn.AddCallback("*", func(ev *irc2.Event) {
			srv.events.Emit("debug.IRC", map[string]interface{}{
				"source": ev,
			})
		})
		return nil
	})
}

func onIRCDisconnect(srv *Server, _ *IRCEvent) {
	logrus.Infoln("*** Disconnected")
	srv.setCurrentNick("")
}

func onIRCMode(srv *Server, ev *IRCEvent) {
	target := ev.Target
	nick := SomeNick(ev.Nick)
	mode := strings.Join(ev.Args[1:], " ")
	currNick := srv.CurrentNick()
	if ev.Nick == currNick {
		nick.me = true
	} else if target == currNick {
		nick = MyNick(target)
	}
	win := srv.wm.Named(target)
	if win != nil {
		WriteMode(win, nick, mode)
	} else {
		WriteMode(srv.wm.Index(0), nick, mode)
	}
}

func onIRCTopic(srv *Server, ev *IRCEvent) {
	target := ev.Target
	nick := SomeNick(ev.Nick)
	topic := strings.Join(ev.Args[1:], " ")
	if ev.Nick == srv.CurrentNick() {
		nick.me = true
	}
	win := srv.wm.Named(target)
	if win != nil {
		WriteTopic(win, nick, topic)
	} else {
		logrus.Warnln("received topic with no channel window:", target, nick, topic)
	}
}

func onIRC433(srv *Server, _ *IRCEvent) {
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		srv.setCurrentNick(conn.GetNick())
		return nil
	})
}

func onIRCNick(srv *Server, ev *IRCEvent) {
	nick := SomeNick(ev.Nick)
	newNick := SomeNick(ev.Message)
	if ev.Nick == srv.CurrentNick() {
		nick.me = true
		newNick.me = true
		srv.setCurrentNick(newNick.string)
	}
	WriteNick(srv.wm, nick, newNick)
}

func onIRCKick(srv *Server, ev *IRCEvent) {
	channel := ev.Target
	kicker := SomeNick(ev.Nick)
	kicked := SomeNick(ev.Args[1])
	myNick := srv.CurrentNick()
	if kicked.string == myNick {
		kicked.me = true
	}
	if kicker.string == myNick {
		kicker.me = true
	}
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
	win := srv.wm.Named(channel)
	if win == nil {
		logrus.Errorln("received kick with no Window:", channel, ev.Message, ev.Nick)
		return
	}
	if ch, ok := win.(*Channel); ok {
		ch.DeleteUser(kicked.string)
	}
	WriteKick(win, kicker, kicked, ev.Message)
}

var namesCache = &struct {
	sync.Mutex
	values map[string][]string
}{values: make(map[string][]string)}

func onIRC353(srv *Server, ev *IRCEvent) {
	// NAMES
	chanName := ev.Args[2]
	nicks := strings.Split(ev.Args[3], " ")
	win := srv.wm.Named(chanName)
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
	win := srv.wm.Named(chanName)
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
	ch.SetUsers(namesCache.values[chanName])
	delete(namesCache.values, chanName)
	srv.wm.events.Emit("ui.DIRTY", nil)
}

func onIRCError(srv *Server, ev *IRCEvent) {
	var kind string
	if len(ev.Args) > 1 {
		kind = ev.Args[1]
	} else {
		kind = ev.Target
	}
	win := srv.wm.Named(kind)
	WriteError(win, kind, ev.Message)
}

func onIRCWhois(srv *Server, ev *IRCEvent) {
	nick := ev.Args[1]
	data := ev.Args[2:]
	win := srv.wm.Named(nick)
	WriteWhois(win, nick, data)
}

func onIRCMessage(srv *Server, ev *IRCEvent) {
	win := srv.wm.Index(0)
	WriteMessage(win, strings.Join(ev.Args[1:], " "))
}

func onIRCNames(srv *Server, ev *IRCEvent) {
	if ev.Code == "PART" || ev.Code == "KICK" {
		myNick := srv.CurrentNick()
		if ev.Nick == myNick {
			// dont bother trying to get names when we are the one leaving
			return
		}
	}
	target := ev.Target
	if strings.HasPrefix(target, "#") {
		srv.IRCDoAsync(func(conn *irc.Connection) error {
			conn.SendRawf("NAMES :%s", target)
			return nil
		})
	}
}

func onIRCJoin(srv *Server, ev *IRCEvent) {
	target := ev.Target
	win := srv.wm.Named(target)
	nick := SomeNick(ev.Nick)
	if ev.Nick == srv.CurrentNick() {
		nick.me = true
	}
	if win == nil {
		ch := &Channel{
			bufferedWindow: newBufferedWindow(target, srv.events),
			users:          []User{},
			usersIndexed:   make(map[string]int),
		}
		srv.wm.Append(ch)
		win = ch
		if nick.me {
			srv.wm.SelectIndex(srv.wm.Len() - 1)
			modeChange(srv, []string{"mode"})
		}
	}
	if ch, ok := win.(*Channel); ok {
		ch.AddUser(SomeUser(nick.string))
	}
	WriteJoin(win, nick)
}

func onIRCPart(srv *Server, ev *IRCEvent) {
	target := ev.Target
	nick := SomeNick(ev.Nick)
	win := srv.wm.Named(target)
	if ev.Nick == srv.CurrentNick() {
		nick.me = true
	}
	if win == nil {
		if !nick.me {
			// dont bother logging if we are the ones leaving
			logrus.Errorln("received message with no Window:", target, ev.Message, nick)
		}
		return
	}
	if ch, ok := win.(*Channel); ok {
		ch.DeleteUser(nick.string)
	}
	WritePart(win, nick, ev.Message)
}

func onIRCAction(srv *Server, ev *IRCEvent) {
	direct := false
	target := ev.Target
	nick := ev.Nick
	myNick := MyNick(srv.CurrentNick())
	if target == myNick.string {
		// its a direct message!
		direct = true
		target = nick
	}
	win := srv.wm.Named(target)
	if win == nil {
		if !direct {
			logrus.Warnln("received action message with no Window:", target, ev.Message, nick)
			return
		} else {
			ch := &DirectMessage{bufferedWindow: newBufferedWindow(target, srv.events)}
			srv.wm.Append(ch)
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
	myNick := MyNick(srv.CurrentNick())
	if target == myNick.string {
		// its a direct message!
		direct = true
		target = nick
	}
	win := srv.wm.Named(target)
	if win == nil {
		if !direct {
			logrus.Warnln("received message with no Window:", target, ev.Message, nick)
			return
		} else {
			ch := &DirectMessage{bufferedWindow: newBufferedWindow(target, srv.events)}
			srv.wm.Append(ch)
			win = ch
		}
	}
	msg := SomeMessage(ev.Message, myNick)
	WritePrivmsg(win, SomeNick(nick), msg)
}

func onIRCNotice(srv *Server, ev *IRCEvent) {
	me := srv.CurrentNick()
	target := SomeTarget(ev.Target, me)
	// "*" is used by at least Freenode when you don't yet have a nick.
	if target.string == "*" {
		target.me = true
	}
	if target.me {
		target = SomeTarget(ev.Nick, me)
	}
	win := srv.wm.Named(target.string)
	if win == nil {
		win = srv.wm.Index(0)
	}

	if strings.Contains(ev.Message, "\x01") {
		WriteCTCP(win, target, false, ev.Message)
		return
	}
	WriteNotice(win, target, false, ev.Message)
}

func onIRCQuit(srv *Server, ev *IRCEvent) {
	nick := SomeNick(ev.Nick)
	message := ev.Message
	if ev.Nick == srv.CurrentNick() {
		nick.me = true
	}
	WriteQuit(srv.wm, nick, message)
}
