// import "code.dopame.me/veonik/squirssi"
package squirssi

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squirssi/colors"
)

type Server struct {
	*logrus.Logger

	ScreenWidth, ScreenHeight int

	mainWindow *ui.Grid
	statusBar  *ActivityTabPane

	inputTextBox *ModedTextInput
	chatPane     *widgets.List
	userListPane *widgets.Table

	events *event.Dispatcher
	irc    *irc.Manager

	windows []Window
	status  *Status

	mu sync.Mutex
}

func NewServer(ev *event.Dispatcher, irc *irc.Manager) (*Server, error) {
	if err := ui.Init(); err != nil {
		return nil, err
	}
	w, h := ui.TerminalDimensions()
	srv := &Server{
		Logger: logrus.StandardLogger(),

		ScreenWidth:  w,
		ScreenHeight: h,

		events: ev,
		irc:    irc,
	}
	srv.initUI()
	return srv, nil
}

func (srv *Server) initUI() {
	srv.userListPane = widgets.NewTable()
	srv.userListPane.Rows = [][]string{}
	srv.userListPane.Border = false
	srv.userListPane.BorderStyle.Fg = ui.ColorBlack
	srv.userListPane.RowSeparator = false
	srv.userListPane.Title = "Users"
	srv.userListPane.TextAlignment = ui.AlignRight
	srv.userListPane.PaddingRight = 1

	srv.chatPane = widgets.NewList()
	srv.chatPane.Rows = []string{}
	srv.chatPane.BorderStyle.Fg = colors.DodgerBlue1
	srv.chatPane.Border = true
	srv.chatPane.PaddingLeft = 1
	srv.chatPane.PaddingRight = 1

	srv.statusBar = &ActivityTabPane{
		TabPane:       widgets.NewTabPane(" 0 "),
		ActivityStyle: ui.NewStyle(ui.ColorBlack, ui.ColorWhite),
	}
	srv.statusBar.SetRect(0, srv.ScreenHeight-3, srv.ScreenWidth, srv.ScreenHeight)
	srv.statusBar.ActiveTabStyle.Fg = colors.DodgerBlue1
	srv.statusBar.Border = true
	srv.statusBar.BorderTop = true
	srv.statusBar.BorderLeft = false
	srv.statusBar.BorderRight = false
	srv.statusBar.BorderBottom = false
	srv.statusBar.BorderStyle.Fg = colors.DodgerBlue1

	srv.inputTextBox = NewModedTextInput(CursorFullBlock)
	srv.inputTextBox.Border = false

	srv.mainWindow = ui.NewGrid()

	srv.status = &Status{bufferedWindow: newBufferedWindow("status", srv.events)}
	srv.windows = []Window{srv.status}
}

func (srv *Server) Close() {
	ui.Close()
}

type screenElement int

const (
	InputTextBox screenElement = iota
	StatusBar
	MainWindow
)

func (srv *Server) RenderOnly(items ...screenElement) {
	var its []ui.Drawable
	for _, it := range items {
		switch it {
		case InputTextBox:
			its = append(its, srv.inputTextBox)
		case StatusBar:
			its = append(its, srv.statusBar)
		case MainWindow:
			its = append(its, srv.mainWindow)
		}
	}
	ui.Render(its...)
}

func tabNames(windows []Window) ([]string, map[int]struct{}) {
	res := make([]string, len(windows))
	activity := make(map[int]struct{})
	for i := 0; i < len(windows); i++ {
		win := windows[i]
		if win.HasActivity() {
			activity[i] = struct{}{}
		}
		res[i] = fmt.Sprintf(" %d ", i)
	}
	return res, activity
}

func (srv *Server) Update() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	channel := srv.windows[srv.statusBar.ActiveTabIndex]
	if channel == nil {
		return
	}
	srv.statusBar.TabNames, srv.statusBar.TabsWithActivity = tabNames(srv.windows)
	srv.chatPane.Rows = channel.Lines()
	srv.chatPane.Title = channel.Title()
	srv.chatPane.SelectedRow = channel.CurrentLine()
	srv.mainWindow.Items = nil
	var rows [][]string
	if v, ok := channel.(WindowWithUserList); ok {
		for _, nick := range v.Users() {
			rows = append(rows, []string{nick})
		}
		srv.mainWindow.Set(
			ui.NewCol(.9, srv.chatPane),
			ui.NewCol(.1, srv.userListPane),
		)
	} else {
		srv.mainWindow.Set(
			ui.NewCol(1, srv.chatPane),
		)
	}
	srv.userListPane.Rows = rows
}

func (srv *Server) Render() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	ui.Render(srv.mainWindow, srv.statusBar, srv.inputTextBox)
}

func (srv *Server) handleKey(e ui.Event) {
	switch e.ID {
	case "<C-c>":
		srv.Close()
		os.Exit(0)
		return
	case "<PageUp>":
		srv.chatPane.ScrollPageUp()
		srv.Render()
	case "<PageDown>":
		srv.chatPane.ScrollPageDown()
		srv.Render()
	case "<Space>":
		srv.inputTextBox.Append(" ")
		srv.RenderOnly(InputTextBox)
	case "<Backspace>":
		srv.inputTextBox.Backspace()
		srv.RenderOnly(InputTextBox)
	case "<C-5>":
		srv.statusBar.FocusRight()
		srv.Update()
		srv.Render()
	case "<Escape>":
		srv.statusBar.FocusLeft()
		srv.Update()
		srv.Render()
	case "<Tab>":
	case "<Enter>":
		in := srv.inputTextBox.Consume()
		if srv.inputTextBox.Mode() == ModeCommand {
			srv.inputTextBox.ToggleMode()
		}
		channel := srv.windows[srv.statusBar.ActiveTabIndex]
		if channel == nil {
			return
		}
		if len(in.Text) == 0 {
			return
		}
		switch in.Kind {
		case ModeCommand:
			args := strings.Split(in.Text, " ")
			c := args[0]
			if cmd, ok := builtIns[c]; ok {
				cmd(srv, args)
			}
		case ModeMessage:
			if err := srv.irc.Do(func(c *irc.Connection) error {
				c.Privmsg(channel.Title(), in.Text)
				_, err := channel.Write([]byte("<veonik> " + in.Text))
				return err
			}); err != nil {
				logrus.Warnln("failed to send message:", err)
			}
		}

	default:
		if len(e.ID) != 1 {
			// a single key resulted in more than one character, probably not a regular char
			return
		}
		if e.ID == "/" && srv.inputTextBox.Len() == 0 {
			srv.inputTextBox.ToggleMode()
		} else {
			srv.inputTextBox.Append(e.ID)
		}
		srv.RenderOnly(InputTextBox)
	}
}

func (srv *Server) resize() {
	srv.statusBar.SetRect(0, srv.ScreenHeight-3, srv.ScreenWidth, srv.ScreenHeight)
	srv.inputTextBox.SetRect(0, srv.ScreenHeight-srv.statusBar.Dy()-1, srv.ScreenWidth, srv.ScreenHeight-srv.statusBar.Dy())
	srv.mainWindow.SetRect(0, 0, srv.ScreenWidth, srv.ScreenHeight-srv.statusBar.Dy()-srv.inputTextBox.Dy())
}

func (srv *Server) bind() {
	srv.events.Bind("ui.DIRTY", event.HandlerFunc(srv.onUIDirty))
	srv.events.Bind("irc.JOIN", event.HandlerFunc(srv.onIRCJoin))
	srv.events.Bind("irc.PRIVMSG", event.HandlerFunc(srv.onIRCPrivmsg))
}

func (srv *Server) Start() {
	srv.bind()
	srv.inputTextBox.Reset()
	srv.resize()
	srv.Update()
	srv.Render()
	srv.Logger.SetOutput(srv.status)

	uiEvents := ui.PollEvents()

	for {
		e := <-uiEvents
		switch e.Type {
		case ui.KeyboardEvent:
			srv.handleKey(e)
			srv.events.Emit("ui.KEYPRESS", map[string]interface{}{
				"key": e.ID,
			})
		case ui.MouseEvent:
			mouse, ok := e.Payload.(ui.Mouse)
			if !ok {
				panic(fmt.Sprintf("received termui Mouse event but Payload was unexpected type %T", e.Payload))
			}
			srv.events.Emit("ui.MOUSE", map[string]interface{}{
				"kind": e.ID,
				"x":    mouse.X,
				"y":    mouse.Y,
				"drag": mouse.Drag,
			})
		case ui.ResizeEvent:
			resize, ok := e.Payload.(ui.Resize)
			if !ok {
				panic(fmt.Sprintf("received termui Resize event but Payload was unexpected type %T", e.Payload))
			}
			srv.ScreenHeight = resize.Height
			srv.ScreenWidth = resize.Width
			srv.resize()
			srv.events.Emit("ui.RESIZE", map[string]interface{}{
				"width":  resize.Width,
				"height": resize.Height,
			})
			srv.Render()
		}
	}
}
