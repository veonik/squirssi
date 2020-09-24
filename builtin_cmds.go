package squirssi

import (
	"strconv"
	"strings"

	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	"github.com/sirupsen/logrus"
)

type Command func(*Server, []string)

var builtIns = map[string]Command{
	"exit":  exitProgram,
	"w":     selectWindow,
	"wc":    closeWindow,
	"join":  joinChannel,
	"part":  partChannel,
	"whois": whoisNick,
	"names": namesChannel,
	"nick":  changeNick,
	"me":    actionMessage,
	"msg":   msgTarget,
}

func exitProgram(srv *Server, _ []string) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.interrupt != nil {
		srv.interrupt()
	}
}

func selectWindow(srv *Server, args []string) {
	if len(args) < 2 {
		logrus.Warnln("selectWindow: expected one argument")
		return
	}
	var err error
	ch, err := strconv.Atoi(args[1])
	if err != nil {
		logrus.Warnln("selectWindow: expected first argument to be an integer")
		return
	}
	srv.WindowManager.SelectIndex(ch)
}

func closeWindow(srv *Server, args []string) {
	var ch int
	if len(args) < 2 {
		ch = srv.WindowManager.ActiveIndex()
	} else {
		var err error
		ch, err = strconv.Atoi(args[1])
		if err != nil {
			logrus.Warnln("selectWindow: expected first argument to be an integer")
			return
		}
	}
	win := srv.WindowManager.Index(ch)
	if strings.HasPrefix(win.Title(), "#") {
		if err := srv.irc.Do(func(conn *irc.Connection) error {
			conn.Part(win.Title())
			return nil
		}); err != nil {
			logrus.Warnln("closeWindow: failed to part channel before closing window")
		}
	}
	srv.WindowManager.CloseIndex(ch)
}

func joinChannel(srv *Server, args []string) {
	if len(args) < 2 {
		logrus.Warnln("joinChannel: expected one argument")
		return
	}
	if err := srv.irc.Do(func(conn *irc.Connection) error {
		conn.Join(args[1])
		return nil
	}); err != nil {
		logrus.Warnln("joinChannel: error joining channel:", err)
	}
}

func partChannel(srv *Server, args []string) {
	if len(args) < 2 {
		logrus.Warnln("partChannel: expected one argument")
		return
	}
	if err := srv.irc.Do(func(conn *irc.Connection) error {
		conn.Part(args[1])
		return nil
	}); err != nil {
		logrus.Warnln("partChannel: error joining channel:", err)
	}
}

func whoisNick(srv *Server, args []string) {
	if len(args) < 2 {
		logrus.Warnln("whoisNick: expected one argument")
		return
	}
	if err := srv.irc.Do(func(conn *irc.Connection) error {
		conn.SendRawf("WHOIS %s", args[1])
		return nil
	}); err != nil {
		logrus.Warnln("whoisNick: error getting whois info:", err)
	}
}

func namesChannel(srv *Server, args []string) {
	if len(args) < 2 {
		logrus.Warnln("namesChannel: expected one argument")
		return
	}
	channel := args[1]
	win := srv.WindowManager.Named(channel)
	if win == nil {
		logrus.Warnln("namesChannel: no window named", channel)
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
	if err := srv.irc.Do(func(conn *irc.Connection) error {
		conn.SendRawf("NAMES :%s", channel)
		return nil
	}); err != nil {
		logrus.Warnln("namesChannel: error getting names:", err)
	}
}

func changeNick(srv *Server, args []string) {
	if len(args) < 2 {
		logrus.Warnln("changeNick: expected one argument")
		return
	}
	if err := srv.irc.Do(func(conn *irc.Connection) error {
		conn.Nick(args[1])
		return nil
	}); err != nil {
		logrus.Warnln("changeNick: error changing nick:", err)
	}
}

func actionMessage(srv *Server, args []string) {
	message := strings.Join(args[1:], " ")
	window := srv.WindowManager.Active()
	if window == nil || window.Title() == "status" {
		return
	}
	if err := srv.irc.Do(func(conn *irc.Connection) error {
		conn.Action(window.Title(), message)
		return nil
	}); err != nil {
		logrus.Warnln("actionMessage: error sending message:", err)
	}
	srv.mu.Lock()
	myNick := MyNick(srv.currentNick)
	srv.mu.Unlock()
	WriteAction(window, myNick, MyMessage(message))
}

func msgTarget(srv *Server, args []string) {
	if len(args) < 3 {
		logrus.Warnln("msgTarget: expects at least 2 arguments")
		return
	}
	target := args[1]
	if target == "status" {
		return
	}
	message := strings.Join(args[2:], " ")
	if err := srv.irc.Do(func(conn *irc.Connection) error {
		conn.Privmsg(target, message)
		return nil
	}); err != nil {
		logrus.Warnln("msgTarget: error sending message:", err)
	}
	window := srv.WindowManager.Named(target)
	if !strings.HasPrefix(target, "#") {
		// direct message!
		if window == nil {
			dm := &DirectMessage{
				newBufferedWindow(target, srv.events),
			}
			srv.WindowManager.Append(dm)
			window = dm
		}
	}
	srv.mu.Lock()
	myNick := MyNick(srv.currentNick)
	srv.mu.Unlock()
	if window == nil {
		// no window for this but we might still have sent the message, so write it to the status window
		window = srv.WindowManager.Index(0)
		message = target + " -> " + message
	}
	WritePrivmsg(window, myNick, MyMessage(message))
}
