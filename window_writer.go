package squirssi

import (
	"fmt"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/sirupsen/logrus"
)

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
	padding := padLeft("* ", 10)
	wins := wm.Windows()
	if nick.me {
		logrus.Infoln("%s[|](fg:grey) Quit:", message)
	}
	for _, win := range wins {
		if nick.me {
			if _, err := win.Write([]byte(fmt.Sprintf("%s[|](fg:grey) Quit: %s", padding, message))); err != nil {
				logrus.Warnln("failed to write user quit:", err)
			}
			continue
		}
		if win.Title() == nick.string {
			// direct message with nick, update title and print there
			if _, err := win.Write([]byte(fmt.Sprintf("%s[|](fg:grey) %s quit (%s)", padding, nick, message))); err != nil {
				logrus.Warnln("failed to write user quit:", err)
			}
		} else if ch, ok := win.(*Channel); ok {
			if ch.HasUser(nick.string) {
				if _, err := win.Write([]byte(fmt.Sprintf("%s[|](fg:grey) %s quit (%s)", padding, nick, message))); err != nil {
					logrus.Warnln("failed to write user quit:", err)
				}
			}
		}
	}
}

func WriteNick(wm *WindowManager, nick Nick, newNick string) {
	padding := padLeft("* ", 10)
	wins := wm.Windows()
	for _, win := range wins {
		if win.Title() == nick.string {
			// direct message with nick, update title and print there
			if dm, ok := win.(*DirectMessage); ok {
				dm.mu.Lock()
				dm.name = nick.string
				dm.mu.Unlock()
			}
			if _, err := win.Write([]byte(fmt.Sprintf("%s[|](fg:grey) %s is now known as %s", padding, nick.String(), newNick))); err != nil {
				logrus.Warnln("failed to write nick change:", err)
			}
		} else if ch, ok := win.(*Channel); ok {
			hasNick := false
			ch.mu.Lock()
			// todo: refactor to avoid locking the channel ourselves
			for i, u := range ch.users {
				if strings.ReplaceAll(strings.ReplaceAll(u, "@", ""), "+", "") == nick.string {
					ch.users[i] = newNick
					hasNick = true
					break
				}
			}
			ch.mu.Unlock()
			if hasNick {
				if _, err := win.Write([]byte(fmt.Sprintf("%s[|](fg:grey) %s is now known as %s", padding, nick, newNick))); err != nil {
					logrus.Warnln("failed to write nick change:", err)
				}
			}
		}
	}
}

func WriteJoin(win Window, nick Nick) {
	padding := padLeft("* ", 10)
	if _, err := win.Write([]byte(fmt.Sprintf("%s[|](fg:grey) %s joined %s", padding, nick.String(), win.Title()))); err != nil {
		logrus.Warnln("%s: failed to write join message:", err)
	}
}

func WritePart(win Window, nick Nick, message string) {
	padding := padLeft("* ", 10)
	title := win.Title()
	if title == message {
		message = ""
	} else {
		message = " (" + message + ")"
	}
	if _, err := win.Write([]byte(fmt.Sprintf("%s[|](fg:grey) %s left %s%s", padding, nick.String(), title, message))); err != nil {
		logrus.Warnln("%s: failed to write part message:", err)
	}
}

func WriteKick(win Window, nick Nick, message string) {
	padding := padLeft("* ", 10)
	if nick.string == message {
		message = ""
	} else {
		message = " (" + message + ")"
	}
	if _, err := win.Write([]byte(fmt.Sprintf("%s[|](fg:grey) %s got kicked from %s%s", padding, nick.String(), win.Title(), message))); err != nil {
		logrus.Warnln("failed to write kick message:", err)
	}
}

func padLeft(s string, padTo int) string {
	ml := len(s)
	if ml > padTo {
		return s[:9] + string(ui.ELLIPSES)
	} else if ml < padTo {
		return strings.Repeat(" ", padTo-ml) + s
	}
	return s
}

func postProcessMessage(nick *Nick, message *Message) {
	if message.refsMe {
		nick.me = true
	}
}

func WriteAction(win Window, nick Nick, message Message) {
	postProcessMessage(&nick, &message)
	m := padLeft("* ", 10) + "[|](fg:grey) " + nick.String() + " " + message.String()
	if _, err := win.Write([]byte(m)); err != nil {
		logrus.Warnln("failed to write action:", err)
	}
}

func WritePrivmsg(win Window, nick Nick, message Message) {
	postProcessMessage(&nick, &message)
	nick.string = padLeft(nick.string, 10)
	m := nick.String() + "[|](fg:grey) " + message.String()
	if _, err := win.Write([]byte(m)); err != nil {
		logrus.Warnln("failed to write message:", err)
	}
}
