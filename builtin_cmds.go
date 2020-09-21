package squirssi

import (
	"strconv"
)

type Command func(*Server, []string)

var builtIns = map[string]Command{
	"w":  selectWindow,
	"wc": closeWindow,
}

func selectWindow(srv *Server, args []string) {
	if len(args) < 2 {
		return
	}
	ch, err := strconv.Atoi(args[1])
	if err != nil {
		return
	}
	if ch < len(srv.statusBar.TabNames) {
		srv.statusBar.ActiveTabIndex = ch
		srv.Update()
		srv.Render()
	}
}

func closeWindow(srv *Server, args []string) {

}
