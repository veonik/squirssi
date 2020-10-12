package squirssi

import (
	"strconv"
	"strings"

	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	"github.com/sirupsen/logrus"
)

type Command func(*Server, []string)

func modeHandler(mode string) Command {
	return func(srv *Server, args []string) {
		args = append(append(append([]string{}, args[:1]...), mode), args[1:]...)
		modeChange(srv, args)
	}
}

var builtInsOrdered = []string{
	"exit",
	"connect",
	"disconnect",
	"w",
	"wc",
	"join",
	"part",
	"topic",
	"whois",
	"names",
	"nick",
	"me",
	"msg",
	"ctcp",
	"notice",
	"kick",
	"mode",
	"ban",
	"unban",
	"op",
	"deop",
	"voice",
	"devoice",
	"mute",
	"unmute",
	"echo",
	"raw",
	"help",
}

var builtIns = map[string]Command{
	"help":   helpCmd,
	"?":      helpCmd,
	"exit":   exitProgram,
	"w":      selectWindow,
	"wc":     closeWindow,
	"join":   joinChannel,
	"part":   partChannel,
	"topic":  topicChange,
	"whois":  whoisNick,
	"names":  namesChannel,
	"nick":   changeNick,
	"me":     actionTarget,
	"msg":    msgTarget,
	"ctcp":   ctcpTarget,
	"notice": noticeTarget,

	"kick":    kickTarget,
	"mode":    modeChange,
	"ban":     modeHandler("+b"),
	"unban":   modeHandler("-b"),
	"op":      modeHandler("+o"),
	"deop":    modeHandler("-o"),
	"voice":   modeHandler("+v"),
	"devoice": modeHandler("-v"),
	"mute":    modeHandler("+q"),
	"unmute":  modeHandler("-q"),

	"connect":    connectServer,
	"disconnect": disconnectServer,
	"quit":       disconnectServer,

	"echo": func(srv *Server, args []string) {
		win := srv.wm.Active()
		_, _ = win.WriteString(strings.Join(args[1:], " "))
	},
	"raw": func(srv *Server, args []string) {
		srv.IRCDoAsync(func(conn *irc.Connection) error {
			conn.SendRaw(strings.Join(args[1:], " "))
			win := srv.wm.Active()
			if win != nil {
				WriteRaw(win, "-> "+strings.Join(args[1:], " "))
			}
			return nil
		})
	},
}

var builtInDescriptions = map[string]string{
	"help":       "Prints this help text.",
	"exit":       "Exits squirssi.",
	"w":          "Switches to the given window by number.",
	"wc":         "Closes the given window by number, or the currently active window.",
	"join":       "Attempts to join the given channel.",
	"part":       "Parts the given channel.",
	"topic":      "Sets the topic for the given channel, or the currently active window.",
	"whois":      "Runs a WHOIS query on the given nickname.",
	"names":      "Runs a NAMES query on the given channel.",
	"nick":       "Changes the current nickname.",
	"me":         "Performs an action message in the current window.",
	"msg":        "Sends a message to the given target.",
	"ctcp":       "Sends a CTCP query to the given target.",
	"notice":     "Sends a NOTICE to the given target.",
	"kick":       "Kicks a user from the given channel.",
	"mode":       "Sets mode on a channel or the current user.",
	"ban":        "Bans (+b) a user from the given channel.",
	"unban":      "Unbans (-b) a user from the given channel.",
	"op":         "Ops (+o) a user on the given channel.",
	"deop":       "Deops (-o) a user on the given channel.",
	"voice":      "Voices (+v) a user on the given channel.",
	"devoice":    "Devoices (-v) a user on the given channel.",
	"mute":       "Mutes (+q) a user on the given channel.",
	"unmute":     "Unmutes (-q) a user on the given channel.",
	"connect":    "Connects to the configured IRC server.",
	"disconnect": "Disconnects from the connected IRC server.",
	"echo":       "Writes any arguments given to the currently active window.",
	"raw":        "Sends a raw IRC command.",
}

func helpCmd(srv *Server, args []string) {
	win := srv.wm.Active()
	if win == nil {
		return
	}
	if len(args) > 1 {
		if desc, ok := builtInDescriptions[args[1]]; ok {
			WriteHelpGeneric(win, "Help information for "+args[1])
			WriteHelp(win, args[1], desc)
		} else {
			WriteHelpGeneric(win, "Unknown command: "+args[1])
		}
		return
	}
	// print all help.
	WriteHelpGeneric(win, "[Available commands:](mod:bold)")
	for _, cmd := range builtInsOrdered {
		WriteHelp(win, cmd, builtInDescriptions[cmd])
	}
}

func connectServer(srv *Server, _ []string) {
	go func() {
		if err := srv.irc.Connect(); err != nil {
			logrus.Errorln("Unable to connect:", err)
		}
	}()
}

func disconnectServer(srv *Server, _ []string) {
	go func() {
		if err := srv.irc.Disconnect(); err != nil {
			logrus.Errorln("Unable to disconnect:", err)
		}
	}()
}

func exitProgram(srv *Server, _ []string) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.interrupt != nil {
		srv.interrupt()
	}
}

func topicChange(srv *Server, args []string) {
	if len(args) < 2 || !strings.HasPrefix("#", args[1]) {
		win := srv.wm.Active()
		if win == nil || !strings.HasPrefix(win.Title(), "#") {
			logrus.Warnln("topic: unable to determine current channel")
			return
		}
		args = append(append([]string{}, args[0], win.Title()), args[1:]...)
	}
	target := args[1]
	if len(args) == 2 {
		srv.IRCDoAsync(func(conn *irc.Connection) error {
			conn.SendRawf("TOPIC %s", target)
			return nil
		})
		return
	}
	topic := strings.Join(args[2:], " ")
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		conn.SendRawf("TOPIC %s :%s", target, topic)
		return nil
	})
}

func kickTarget(srv *Server, args []string) {
	if len(args) < 2 || !strings.HasPrefix(args[1], "#") {
		win := srv.wm.Active()
		t := ""
		if win == nil || win.Title() == "status" {
			t = srv.CurrentNick()
		} else {
			t = win.Title()
		}
		args = append(append([]string{}, args[0], t), args[1:]...)
	}
	target := args[1]
	if len(target) > 0 && target[0] != '#' {
		logrus.Warnln("kick: unable to determine current channel")
		return
	}
	nick := args[2]
	msg := strings.Join(args[3:], " ")
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		conn.Kick(nick, target, msg)
		return nil
	})
}

func modeChange(srv *Server, args []string) {
	if len(args) < 2 || strings.HasPrefix(args[1], "+") || strings.HasPrefix(args[1], "-") {
		win := srv.wm.Active()
		t := ""
		if win == nil || win.Title() == "status" {
			t = srv.CurrentNick()
		} else {
			t = win.Title()
		}
		args = append(append([]string{}, args[0], t), args[1:]...)
	}
	target := args[1]
	modes := args[2:]
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		conn.Mode(target, modes...)
		return nil
	})
}

func selectWindow(srv *Server, args []string) {
	if len(args) < 2 {
		logrus.Warnln("window: expected one argument")
		return
	}
	var err error
	ch, err := strconv.Atoi(args[1])
	if err != nil {
		logrus.Warnln("window: expected first argument to be an integer")
		return
	}
	srv.wm.SelectIndex(ch)
}

func closeWindow(srv *Server, args []string) {
	var ch int
	if len(args) < 2 {
		ch = srv.wm.ActiveIndex()
	} else {
		var err error
		ch, err = strconv.Atoi(args[1])
		if err != nil {
			logrus.Warnln("window_close: expected first argument to be an integer")
			return
		}
	}
	win := srv.wm.Index(ch)
	if ch, ok := win.(*Channel); ok {
		myNick := srv.CurrentNick()
		if ch.HasUser(myNick) {
			srv.IRCDoAsync(func(conn *irc.Connection) error {
				conn.Part(win.Title())
				return nil
			})
		}
	}
	srv.wm.CloseIndex(ch)
}

func joinChannel(srv *Server, args []string) {
	if len(args) < 2 {
		logrus.Warnln("join: expected one argument")
		return
	}
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		conn.Join(args[1])
		return nil
	})
}

func partChannel(srv *Server, args []string) {
	if len(args) < 2 {
		logrus.Warnln("part: expected one argument")
		return
	}
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		conn.Part(args[1])
		return nil
	})
}

func whoisNick(srv *Server, args []string) {
	if len(args) < 2 {
		logrus.Warnln("whois: expected one argument")
		return
	}
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		conn.SendRawf("WHOIS %s", args[1])
		return nil
	})
}

func namesChannel(srv *Server, args []string) {
	if len(args) < 2 {
		logrus.Warnln("names: expected one argument")
		return
	}
	channel := args[1]
	win := srv.wm.Named(channel)
	if win == nil {
		logrus.Warnln("names: no window named", channel)
		return
	}
	irc353Handler := event.HandlerFunc(func(ev *event.Event) {
		args := ev.Data["Args"].([]string)
		chanName := args[2]
		nicks := args[3]
		logrus.Infof("NAMES %s: %s", chanName, nicks)
	})
	var irc366Handler event.Handler
	irc366Handler = event.HandlerFunc(func(ev *event.Event) {
		args := ev.Data["Args"].([]string)
		chanName := args[1]
		logrus.Infof("END NAMES %s", chanName)
		srv.events.Unbind("irc.353", irc353Handler)
		srv.events.Unbind("irc.366", irc366Handler)
	})
	srv.events.Bind("irc.353", irc353Handler)
	srv.events.Bind("irc.366", irc366Handler)
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		conn.SendRawf("NAMES :%s", channel)
		return nil
	})
}

func changeNick(srv *Server, args []string) {
	if len(args) < 2 {
		logrus.Warnln("nick: expected one argument")
		return
	}
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		conn.Nick(args[1])
		return nil
	})
}

func actionTarget(srv *Server, args []string) {
	message := strings.Join(args[1:], " ")
	window := srv.wm.Active()
	if window == nil || window.Title() == "status" {
		return
	}
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		conn.Action(window.Title(), message)
		return nil
	})
	myNick := MyNick(srv.CurrentNick())
	WriteAction(window, myNick, MyMessage(message))
}

func msgTarget(srv *Server, args []string) {
	if len(args) < 3 {
		logrus.Warnln("msg: expected at least 2 arguments")
		return
	}
	target := args[1]
	if target == "status" {
		return
	}
	message := strings.Join(args[2:], " ")
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		conn.Privmsg(target, message)
		return nil
	})
	window := srv.wm.Named(target)
	if !strings.HasPrefix(target, "#") {
		// direct message!
		if window == nil {
			dm := &DirectMessage{
				newBufferedWindow(target, srv.events),
			}
			srv.wm.Append(dm)
			window = dm
		}
	}
	myNick := MyNick(srv.CurrentNick())
	if window == nil {
		// no window for this but we might still have sent the message, so write it to the status window
		window = srv.wm.Index(0)
		message = target + " -> " + message
	}
	WritePrivmsg(window, myNick, MyMessage(message))
}

func noticeTarget(srv *Server, args []string) {
	if len(args) < 3 {
		logrus.Warnln("notice: expected at least 2 arguments")
		return
	}
	target := args[1]
	if target == "status" {
		return
	}
	message := strings.Join(args[2:], " ")
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		conn.Notice(target, message)
		return nil
	})
	window := srv.wm.Named(target)
	if window == nil {
		// no window for this but we might still have sent the message, so write it to the status window
		window = srv.wm.Index(0)
	}
	WriteNotice(window, SomeTarget(target, srv.CurrentNick()), true, message)
}

func ctcpTarget(srv *Server, args []string) {
	if len(args) < 3 {
		logrus.Warnln("ctcp: expected at least 2 arguments")
		return
	}
	target := args[1]
	if target == "status" {
		return
	}
	message := strings.Join(args[2:], " ")
	srv.IRCDoAsync(func(conn *irc.Connection) error {
		conn.SendRawf("PRIVMSG %s :\x01%s\x01", target, message)
		return nil
	})
	window := srv.wm.Named(target)
	if window == nil {
		// no window for this but we might still have sent the message, so write it to the status window
		window = srv.wm.Index(0)
	}
	WriteCTCP(window, SomeTarget(target, srv.CurrentNick()), true, message)
}
