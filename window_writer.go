package squirssi

import (
	"fmt"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/sirupsen/logrus"
)

func padLeft(s string, padTo int) string {
	ml := len(s)
	if ml > padTo {
		return s[:padTo-1] + string(ui.ELLIPSES)
	} else if ml < padTo {
		return strings.Repeat(" ", padTo-ml) + s
	}
	return s
}

func padRight(s string, padTo int) string {
	ml := len(s)
	if ml > padTo {
		return s[:padTo-1] + string(ui.ELLIPSES)
	} else if ml < padTo {
		return s + strings.Repeat(" ", padTo-ml)
	}
	return s
}

type Message struct {
	string
	mine   bool
	refsMe bool
}

func (m Message) String() string {
	if m.mine {
		return "[" + m.string + "](fg:gray100)"
	} else if m.refsMe {
		return "[" + m.string + "](mod:bold)"
	}
	return m.string
}

func MyMessage(m string) Message {
	return Message{m, true, false}
}

func SomeMessage(m string, myNick Nick) Message {
	if strings.Contains(m, myNick.string) {
		return Message{m, false, true}
	}
	return Message{m, false, false}
}

type Nick struct {
	string
	me bool
}

type Target struct {
	Nick
	Me Nick
}

func (n Target) IsChannel() bool {
	return len(n.string) > 0 && n.string[0] == '#'
}

func SomeTarget(name string, me string) Target {
	if me == name {
		return Target{MyNick(name), MyNick(me)}
	}
	return Target{SomeNick(name), MyNick(me)}
}

func (n Nick) String() string {
	if n.me {
		return "[" + n.string + "](mod:bold)"
	}
	return n.string
}

func MyNick(nick string) Nick {
	return Nick{nick, true}
}

func SomeNick(nick string) Nick {
	return Nick{nick, false}
}

func WriteQuit(wm *WindowManager, nick Nick, message string) {
	wins := wm.Windows()
	for _, win := range wins {
		padding := padLeft("* ", win.padding())
		if nick.me {
			if _, err := win.Write([]byte(fmt.Sprintf("%s[│](fg:grey) Quit: %s", padding, message))); err != nil {
				logrus.Warnln("failed to write user quit:", err)
			}
			continue
		}
		if win.Title() == nick.string {
			// direct message with nick, update title and print there
			if _, err := win.Write([]byte(fmt.Sprintf("%s[│](fg:grey) %s quit (%s)", padding, nick, message))); err != nil {
				logrus.Warnln("failed to write user quit:", err)
			}
		} else if ch, ok := win.(*Channel); ok {
			if ch.DeleteUser(nick.string) {
				if _, err := win.Write([]byte(fmt.Sprintf("%s[│](fg:grey) %s quit (%s)", padding, nick, message))); err != nil {
					logrus.Warnln("failed to write user quit:", err)
				}
			}
		}
	}
}

func WriteNick(wm *WindowManager, nick Nick, newNick Nick) {
	wins := wm.Windows()
	for _, win := range wins {
		padding := padLeft("* ", win.padding())
		if win.Title() == nick.string {
			// direct message with nick, update title and print there
			if dm, ok := win.(*DirectMessage); ok {
				dm.mu.Lock()
				dm.name = nick.string
				dm.mu.Unlock()
			}
			if _, err := win.Write([]byte(fmt.Sprintf("%s[│](fg:grey) %s is now known as %s", padding, nick.String(), newNick))); err != nil {
				logrus.Warnln("failed to write nick change:", err)
			}
		} else if ch, ok := win.(*Channel); ok {
			if ch.UpdateUser(nick.string, newNick.string) {
				if _, err := win.Write([]byte(fmt.Sprintf("%s[│](fg:grey) %s is now known as %s", padding, nick, newNick))); err != nil {
					logrus.Warnln("failed to write nick change:", err)
				}
			}
		} else {
			if _, err := win.Write([]byte(fmt.Sprintf("%s[│](fg:grey) You are now known as %s", padding, newNick))); err != nil {
				logrus.Warnln("failed to write nick change:", err)
			}
		}
	}
}

func WriteWhois(win Window, nick string, args []string) {
	m := strings.Join(args, " ")
	if win == nil {
		logrus.Infof("WHOIS %s => %s", nick, m)
		return
	}
	padding := padLeft("* ", win.padding())
	if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) WHOIS => %s", padding, m)); err != nil {
		logrus.Warnln("failed to write whois result to status:", err)
	}
}

func WriteError(win Window, name, message string) {
	if win == nil {
		logrus.Errorln("%s: %s", name, message)
		return
	}
	padding := padLeft("* ", win.padding())
	if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) %s: %s", padding, name, message)); err != nil {
		logrus.Warnln("%s: failed to write error message:", err)
	}
}

func WriteRaw(win Window, raw string) {
	padding := padLeft("RAW", win.padding())
	if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) %s", padding, raw)); err != nil {
		logrus.Warnln("%s: failed to write raw command:", err)
	}
}

func WriteMessage(win Window, message string) {
	padding := padLeft("* ", win.padding())
	if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) [%s](fg:red4)", padding, message)); err != nil {
		logrus.Warnln("%s: failed to write error message:", err)
	}
}

func Write331(win Window) {
	padding := padLeft("* ", win.padding())
	if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) No topic is set in %s", padding, win.Title())); err != nil {
		logrus.Warnln("%s: failed to write topic message:", err)
	}
}

func Write332(win Window, topic string) {
	padding := padLeft("* ", win.padding())
	if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) Topic for %s is: %s", padding, win.Title(), topic)); err != nil {
		logrus.Warnln("%s: failed to write topic message:", err)
	}
}

func WriteJoin(win Window, nick Nick) {
	padding := padLeft("* ", win.padding())
	if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) %s joined %s", padding, nick.String(), win.Title())); err != nil {
		logrus.Warnln("%s: failed to write join message:", err)
	}
}

func WriteModes(win Window, modes string) {
	padding := padLeft("* ", win.padding())
	if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) Modes for %s: %s", padding, win.Title(), modes)); err != nil {
		logrus.Warnln("%s: failed to write mode message:", err)
	}
}

func WriteMode(win Window, nick Nick, mode string) {
	if nick.string == "" {
		nick.string = "Server"
	}
	padding := padLeft("* ", win.padding())
	title := win.Title()
	if title == "status" {
		if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) Changed mode for %s (%s)", padding, nick.String(), mode)); err != nil {
			logrus.Warnln("%s: failed to write mode message:", err)
		}
		return
	}
	if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) %s changed mode on %s (%s)", padding, nick.String(), win.Title(), mode)); err != nil {
		logrus.Warnln("%s: failed to write mode message:", err)
	}
}

func WriteTopic(win Window, nick Nick, topic string) {
	if ch, ok := win.(*Channel); ok {
		ch.mu.Lock()
		ch.topic = topic
		ch.mu.Unlock()
	}
	padding := padLeft("* ", win.padding())
	if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) %s changed topic on %s to: %s", padding, nick.String(), win.Title(), topic)); err != nil {
		logrus.Warnln("%s: failed to write mode message:", err)
	}
}

func WritePart(win Window, nick Nick, message string) {
	padding := padLeft("* ", win.padding())
	title := win.Title()
	if title == message {
		message = ""
	} else {
		message = " (" + message + ")"
	}
	if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) %s left %s%s", padding, nick.String(), title, message)); err != nil {
		logrus.Warnln("%s: failed to write part message:", err)
	}
}

func WriteKick(win Window, nick Nick, message string) {
	padding := padLeft("* ", win.padding())
	if nick.string == message {
		message = ""
	} else {
		message = " (" + message + ")"
	}
	if nick.me {
		win.Notice()
	}
	if _, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) %s got kicked from %s%s", padding, nick.String(), win.Title(), message)); err != nil {
		logrus.Warnln("failed to write kick message:", err)
	}
}

func postProcessMessage(nick *Nick, message *Message) {
	if message.refsMe {
		nick.me = true
	}
}

func WriteAction(win Window, nick Nick, message Message) {
	postProcessMessage(&nick, &message)
	m := padLeft("* ", win.padding()) + "[│](fg:grey) " + nick.String() + " " + message.String()
	if message.refsMe || nick.string == win.Title() {
		win.Notice()
	}
	if _, err := win.WriteString(m); err != nil {
		logrus.Warnln("failed to write action:", err)
	}
}

func WritePrivmsg(win Window, nick Nick, message Message) {
	postProcessMessage(&nick, &message)
	if message.refsMe || nick.string == win.Title() {
		win.Notice()
	}
	nick.string = padLeft(nick.string, win.padding())
	m := nick.String() + "[│](fg:grey) " + message.String()
	if _, err := win.WriteString(m); err != nil {
		logrus.Warnln("failed to write message:", err)
	}
}

func WriteHelpGeneric(win Window, msg string) {
	padding := padLeft("HELP", win.padding())
	m := fmt.Sprintf("[%s](fg:yellow,mod:bold)[│](fg:grey) %s", padding, msg)
	if _, err := win.WriteString(m); err != nil {
		logrus.Warnln("failed to write message:", err)
	}
}

func WriteHelp(win Window, cmd string, desc string) {
	padding := padLeft("HELP", win.padding())
	cmd = padRight(cmd, 10)
	m := fmt.Sprintf("[%s](fg:yellow,mod:bold)[│](fg:grey)   [%s](mod:bold)  %s", padding, cmd, desc)
	if _, err := win.WriteString(m); err != nil {
		logrus.Warnln("failed to write message:", err)
	}
}

func WriteNotice(win Window, target Target, sent bool, message string) {
	win.Notice()
	if win.Title() == "status" {
		arrow := "->"
		if sent {
			arrow = "<-"
		}
		padding := padLeft("NOTICE", win.padding())
		m := fmt.Sprintf("[%s](fg:grey100,mod:bold)[│](fg:grey) %s %s %s", padding, target, arrow, message)
		if _, err := win.WriteString(m); err != nil {
			logrus.Warnln("failed to write message:", err)
		}
	} else {
		nick := target.Nick
		if sent {
			nick = target.Me
		}
		nick.string = padLeft(nick.string, win.padding())
		m := nick.String() + "[│](fg:grey) [NOTICE](fg:grey100,mod:bold): " + message
		if _, err := win.WriteString(m); err != nil {
			logrus.Warnln("failed to write message:", err)
		}
	}
}

func WriteCTCP(win Window, target Target, sent bool, message string) {
	win.Notice()
	if win.Title() == "status" {
		arrow := "->"
		if sent {
			arrow = "<-"
		}
		padding := padLeft("CTCP", win.padding())
		m := fmt.Sprintf("[%s](fg:grey100,mod:bold)[│](fg:grey) %s %s %s", padding, target, arrow, message)
		if _, err := win.WriteString(m); err != nil {
			logrus.Warnln("failed to write message:", err)
		}
	} else {
		nick := target.Nick
		if sent {
			nick = target.Me
		}
		nick.string = padLeft(nick.string, win.padding())
		m := nick.String() + "[│](fg:grey) [CTCP](fg:grey100,mod:bold) " + message
		if _, err := win.WriteString(m); err != nil {
			logrus.Warnln("failed to write message:", err)
		}
	}
}
