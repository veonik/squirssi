package squirssi

import (
	"fmt"

	"code.dopame.me/veonik/squircy3/event"
	"github.com/sirupsen/logrus"
)

func (srv *Server) onUIDirty(_ *event.Event) {
	srv.Update()
	srv.Render()
}

func (srv *Server) onIRCJoin(ev *event.Event) {
	var win Window
	target := ev.Data["Target"].(string)
	for _, w := range srv.windows {
		if w.Title() == target {
			win = w
			break
		}
	}
	if win == nil {
		ch := &Channel{
			bufferedWindow: newBufferedWindow(target, srv.events),
			users:          []string{"veonik"},
		}
		srv.windows = append(srv.windows, ch)
	}
}

func (srv *Server) onIRCPrivmsg(ev *event.Event) {
	var win Window
	target := ev.Data["Target"].(string)
	for _, w := range srv.windows {
		if w.Title() == target {
			win = w
			break
		}
	}
	if win == nil {
		logrus.Warnln("received message with no Window:", target, ev.Data["Message"], ev.Data["Nick"])
	} else {
		if v, ok := win.(*Channel); ok {
			if v.current == len(v.lines)-1 {
				v.current++
			}
			if _, err := v.Write([]byte(fmt.Sprintf("<%s> %s", ev.Data["Nick"], ev.Data["Message"]))); err != nil {
				logrus.Warnln("error writing to Channel:", err)
				return
			}
		}
	}
}
