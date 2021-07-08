package squirssi

import (
	"fmt"
	"strings"
	"time"

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

func padLeftStyled(s StyledString, padTo int) string {
	ml := len(s.string)
	res := s.string
	if ml > padTo {
		res = s.string[:padTo-1] + string(ui.ELLIPSES)
	} else if ml < padTo {
		res = strings.Repeat(" ", padTo-ml) + s.string
	}
	if s.Style != "" {
		return "[" + res + "](" + s.Style + ")"
	}
	return res
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
	return "[" + n.string + "](mod:none)"
}

func (n Nick) Styled() StyledString {
	if n.me {
		return StyledString{string: n.string, Style: "mod:bold"}
	}
	return StyledString{string: n.string, Style: "mod:none"}
}

func MyNick(nick string) Nick {
	return Nick{nick, true}
}

func SomeNick(nick string) Nick {
	return Nick{nick, false}
}

type StyledString struct {
	string
	Style string
}

func Unstyled(s string) StyledString {
	return StyledString{string: s}
}

func Styled(s, style string) StyledString {
	return StyledString{string: s, Style: style}
}

func (s StyledString) String() string {
	if s.Style != "" {
		return "[" + s.string + "](" + s.Style + ")"
	}
	return s.string
}

var basePrefix = Unstyled("* ")

func WritePrefixed(win Window, prefix StyledString, message string) error {
	padding := padLeftStyled(prefix, win.padding())
	_, err := win.WriteString(fmt.Sprintf("%s[│](fg:grey) %s", padding, message))
	return err
}

func WriteQuit(wm *WindowManager, nick Nick, message string) {
	wins := wm.Windows()
	for _, win := range wins {
		if nick.me {
			if err := WritePrefixed(win, basePrefix, fmt.Sprintf("Quit: %s", message)); err != nil {
				logrus.Warnf("%s: failed to write user quit: %s", win.Title(), err)
			}
			continue
		}
		if win.Title() == nick.string {
			// direct message with nick, update title and print there
			if err := WritePrefixed(win, basePrefix, fmt.Sprintf("%s quit (%s)", nick, message)); err != nil {
				logrus.Warnf("%s: failed to write user quit: %s", win.Title(), err)
			}
		} else if ch, ok := win.(*Channel); ok {
			if ch.DeleteUser(nick.string) {
				if err := WritePrefixed(win, basePrefix, fmt.Sprintf("%s quit (%s)", nick, message)); err != nil {
					logrus.Warnf("%s: failed to write user quit: %s", win.Title(), err)
				}
			}
		}
	}
}

func WriteNick(wm *WindowManager, nick Nick, newNick Nick) {
	wins := wm.Windows()
	for _, win := range wins {
		if win.Title() == nick.string {
			// direct message with nick, update title and print there
			if dm, ok := win.(*DirectMessage); ok {
				dm.mu.Lock()
				dm.name = newNick.string
				dm.mu.Unlock()
			}
			if err := WritePrefixed(win, basePrefix, fmt.Sprintf("%s is now known as %s", nick.String(), newNick)); err != nil {
				logrus.Warnf("%s: failed to write nick change: %s", win.Title(), err)
			}
		} else if ch, ok := win.(*Channel); ok {
			if ch.UpdateUser(nick.string, newNick.string) {
				if err := WritePrefixed(win, basePrefix, fmt.Sprintf("%s is now known as %s", nick, newNick)); err != nil {
					logrus.Warnf("%s: failed to write nick change: %s", win.Title(), err)
				}
			}
		} else if win.Title() == "status" && nick.me {
			if err := WritePrefixed(win, basePrefix, fmt.Sprintf("You are now known as %s", newNick)); err != nil {
				logrus.Warnf("%s: failed to write nick change: %s", win.Title(), err)
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
	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("WHOIS => %s", m)); err != nil {
		logrus.Warnf("%s: failed to write whois result message: %s", win.Title(), err)
	}
}

func WriteError(win Window, name, message string) {
	if win == nil {
		logrus.Errorf("%s: %s", name, message)
		return
	}
	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("%s: %s", name, message)); err != nil {
		logrus.Warnf("%s: failed to write error message: %s", win.Title(), err)
	}
}

func WriteRaw(win Window, raw string) {
	if err := WritePrefixed(win, Unstyled("RAW"), raw); err != nil {
		logrus.Warnf("%s: failed to write raw command: %s", win.Title(), err)
	}
}

func WriteMessage(win Window, message string) {
	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("[%s](fg:red4)", message)); err != nil {
		logrus.Warnf("%s: failed to write message: %s", win.Title(), err)
	}
}

func WriteEval(win Window, script string) {
	if err := WritePrefixed(win, Styled("EVAL", "fg:orange"), fmt.Sprintf("[>](fg:grey100,mod:bold) %s", script)); err != nil {
		logrus.Warnf("%s: failed to write eval command: %s", win.Title(), err)
	}
}

func WriteEvalResult(win Window, script string) {
	if err := WritePrefixed(win, Styled("EVAL", "fg:orange"), fmt.Sprintf("[=](fg:grey100,mod:bold) %s", script)); err != nil {
		logrus.Warnf("%s: failed to write eval result: %s", win.Title(), err)
	}
}

func WriteEvalError(win Window, script string) {
	if err := WritePrefixed(win, Styled("EVAL", "fg:orange"), fmt.Sprintf("[!](fg:red,mod:bold) %s", script)); err != nil {
		logrus.Warnf("%s: failed to write eval error: %s", win.Title(), err)
	}
}
func Write329(win Window, created time.Time) {
	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("Channel created at [%s](mod:bold)", created.String())); err != nil {
		logrus.Warnf("%s: failed to write created at message: %s", win.Title(), err)
	}
}
func Write331(win Window) {
	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("No topic is set in [%s](mod:bold)", win.Title())); err != nil {
		logrus.Warnf("%s: failed to write topic message: %s", win.Title(), err)
	}
}

func Write332(win Window, topic string) {
	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("Topic for [%s](mod:bold) is: %s", win.Title(), topic)); err != nil {
		logrus.Warnf("%s: failed to write topic message: %s", win.Title(), err)
	}
}

func WriteJoin(win Window, nick Nick) {
	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("%s joined [%s](mod:bold)", nick.String(), win.Title())); err != nil {
		logrus.Warnf("%s: failed to write join message: %s", win.Title(), err)
	}
}

func WriteModes(win Window, modes string) {
	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("Modes for [%s](mod:bold): %s", win.Title(), modes)); err != nil {
		logrus.Warnf("%s: failed to write modes: %s", win.Title(), err)
	}
}

func WriteMode(win Window, nick Nick, mode string) {
	if nick.string == "" {
		nick.string = "Server"
	}
	title := win.Title()
	if title == "status" {
		if err := WritePrefixed(win, basePrefix, fmt.Sprintf("Changed mode for %s (%s)", nick.String(), mode)); err != nil {
			logrus.Warnf("%s: failed to write mode message: %s", win.Title(), err)
		}
		return
	}

	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("%s changed mode on [%s](mod:bold) (%s)", nick.String(), win.Title(), mode)); err != nil {
		logrus.Warnf("%s: failed to write error message: %s", win.Title(), err)
	}
}

func WriteTopic(win Window, nick Nick, topic string) {
	if ch, ok := win.(*Channel); ok {
		ch.mu.Lock()
		ch.topic = topic
		ch.mu.Unlock()
	}
	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("%s changed topic on [%s](mod:bold) to: %s", nick.String(), win.Title(), topic)); err != nil {
		logrus.Warnf("%s: failed to write topic message: %s", win.Title(), err)
	}
}

func WritePart(win Window, nick Nick, message string) {
	title := win.Title()
	if title == message {
		message = ""
	} else {
		message = " (" + message + ")"
	}
	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("%s left [%s](mod:bold)%s", nick.String(), title, message)); err != nil {
		logrus.Warnf("%s: failed to write part message: %s", win.Title(), err)
	}
}

func WriteKick(win Window, kicker Nick, kicked Nick, message string) {
	if kicked.string == message {
		message = ""
	} else {
		message = " (" + message + ")"
	}
	if kicked.me {
		win.Notice()
	}
	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("%s kicked %s from [%s](mod:bold)%s", kicker.String(), kicked.String(), win.Title(), message)); err != nil {
		logrus.Warnf("%s: failed to write kick message: %s", win.Title(), err)
	}
}

func postProcessMessage(nick *Nick, message *Message) {
	if message.refsMe {
		nick.me = true
	}
}

func WriteAction(win Window, nick Nick, message Message) {
	postProcessMessage(&nick, &message)
	if message.refsMe || nick.string == win.Title() {
		win.Notice()
	}
	if err := WritePrefixed(win, basePrefix, fmt.Sprintf("%s %s", nick.String(), message.String())); err != nil {
		logrus.Warnf("%s: failed to write action message: %s", win.Title(), err)
	}
}

func WritePrivmsg(win Window, nick Nick, message Message) {
	postProcessMessage(&nick, &message)
	if message.refsMe || nick.string == win.Title() {
		win.Notice()
	}
	if err := WritePrefixed(win, nick.Styled(), message.String()); err != nil {
		logrus.Warnf("%s: failed to write privmsg: %s", win.Title(), err)
	}
}

func WriteHelpGeneric(win Window, msg string) {
	prefix := Styled("HELP", "fg:yellow,mod:bold")
	if err := WritePrefixed(win, prefix, msg); err != nil {
		logrus.Warnf("%s: failed to write help message: %s", win.Title(), err)
	}
}

func WriteHelp(win Window, cmd string, desc string) {
	cmd = padRight(cmd, 10)
	prefix := Styled("HELP", "fg:yellow,mod:bold")
	if err := WritePrefixed(win, prefix, fmt.Sprintf("[%s](mod:bold)  %s", cmd, desc)); err != nil {
		logrus.Warnf("%s: failed to write help message: %s", win.Title(), err)
	}
}

func WriteNotice(win Window, target Target, sent bool, message string) {
	writeNotice(win, target, "NOTICE", sent, message)
}

func WriteCTCP(win Window, target Target, sent bool, message string) {
	writeNotice(win, target, "CTCP", sent, message)
}

func writeNotice(win Window, target Target, kind string, sent bool, message string) {
	win.Notice()
	if win.Title() == "status" {
		arrow := "->"
		if sent {
			arrow = "<-"
		}
		prefix := Styled(kind, "fg:grey100,mod:bold")
		m := fmt.Sprintf("%s %s %s", target, arrow, message)
		if err := WritePrefixed(win, prefix, m); err != nil {
			logrus.Warnf("%s: failed to write %s message: %s", win.Title(), strings.ToLower(kind), err)
		}
	} else {
		nick := target.Nick
		if sent {
			nick = target.Me
		}
		m := "[" + kind + "](fg:grey100,mod:bold) " + message
		if err := WritePrefixed(win, nick.Styled(), m); err != nil {
			logrus.Warnf("%s: failed to write %s message: %s", win.Title(), strings.ToLower(kind), err)
		}
	}
}
