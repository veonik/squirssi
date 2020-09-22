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
	"w":     selectWindow,
	"wc":    closeWindow,
	"join":  joinChannel,
	"part":  partChannel,
	"whois": whoisNick,
	"names": namesChannel,
	"nick":  changeNick,
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
	srv.mu.Lock()
	if ch >= len(srv.windows) {
		logrus.Warnf("selectWindow: no window #%d", ch)
		srv.mu.Unlock()
		return
	}
	srv.statusBar.ActiveTabIndex = ch
	srv.mu.Unlock()
	srv.Update()
	srv.Render()
}

func closeWindow(srv *Server, args []string) {
	var ch int
	if len(args) < 2 {
		srv.mu.Lock()
		ch = srv.statusBar.ActiveTabIndex
		srv.mu.Unlock()
	} else {
		var err error
		ch, err = strconv.Atoi(args[1])
		if err != nil {
			logrus.Warnln("selectWindow: expected first argument to be an integer")
			return
		}
	}
	win := srv.windows[ch]
	if strings.HasPrefix(win.Title(), "#") {
		if err := srv.irc.Do(func(conn *irc.Connection) error {
			conn.Part(win.Title())
			return nil
		}); err != nil {
			logrus.Warnln("closeWindow: failed to part channel before closing window")
		}
	}
	srv.CloseWindow(ch)
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
	win := srv.WindowNamed(channel)
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
