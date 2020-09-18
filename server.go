// import "code.dopame.me/veonik/squirssi"
package squirssi

import (
	"fmt"
	"os"

	"code.dopame.me/veonik/squircy3/event"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"

	"code.dopame.me/veonik/squirssi/colors"
)

type Window interface {
	Title() string
	Lines() []string
	HasActivity() bool
}

type WindowWithNicklist interface {
	Window
	Nicklist() []string
}

type Channel struct {
	name    string
	topic   string
	modes   string
	users   []string
	lines   []string
	current int

	hasUnseen bool
}

func (c *Channel) HasActivity() bool {
	return c.hasUnseen
}

func (c *Channel) Title() string {
	return c.name
}

func (c *Channel) Lines() []string {
	return c.lines
}

func (c *Channel) Nicklist() []string {
	return c.users
}

type DirectMessage struct {
	user    string
	lines   []string
	current int

	hasUnseen bool
}

func (c *DirectMessage) Title() string {
	return c.user
}

func (c *DirectMessage) Lines() []string {
	return c.lines
}

func (c *DirectMessage) HasActivity() bool {
	return c.hasUnseen
}

type Server struct {
	ScreenWidth, ScreenHeight int

	mainWindow *ui.Grid
	statusBar  *widgets.TabPane

	inputTextBox *widgets.Paragraph
	chatPane     *widgets.List
	nicklistPane *widgets.Table

	input *string

	events *event.Dispatcher

	windows []Window
}

func NewServer(ev *event.Dispatcher) (*Server, error) {
	if err := ui.Init(); err != nil {
		return nil, err
	}
	w, h := ui.TerminalDimensions()
	return &Server{
		ScreenWidth:  w,
		ScreenHeight: h,
		events:       ev,
	}, nil
}

func (srv *Server) Close() {
	ui.Close()
}

type ScreenItem int

const (
	InputTextBox ScreenItem = iota
	StatusBar
	MainWindow
)

func (srv *Server) RenderOnly(items ...ScreenItem) {
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

func (srv *Server) Render() {
	channel := srv.windows[srv.statusBar.ActiveTabIndex]
	if channel == nil {
		return
	}
	srv.chatPane.Rows = channel.Lines()
	var rows [][]string
	if v, ok := channel.(WindowWithNicklist); ok {
		for _, nick := range v.Nicklist() {
			rows = append(rows, []string{nick})
		}
	}
	srv.nicklistPane.Rows = rows
	srv.chatPane.Title = channel.Title()
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
		srv.appendInput(" ")
		srv.RenderOnly(InputTextBox)
	case "<Backspace>":
		srv.backspaceInput()
	case "<C-5>":
		srv.statusBar.FocusRight()
		srv.Render()
	case "<Escape>":
		srv.statusBar.FocusLeft()
		srv.Render()
	case "<Enter>":
		channel := srv.windows[srv.statusBar.ActiveTabIndex]
		if channel == nil {
			return
		}
		if c, ok := channel.(*Channel); ok {
			c.lines = append(c.lines, "<veonik> "+srv.consumeInput())
			srv.Render()
		}
	default:
		if len(e.ID) != 1 {
			// a single key resulted in more than one character, probably not a regular char
			return
		}
		srv.appendInput(e.ID)
		srv.RenderOnly(InputTextBox)
	}
}

func (srv *Server) consumeInput() string {
	defer srv.resetInput()
	if len(*srv.input) < 2 {
		return ""
	}
	return (*srv.input)[2:]
}

func (srv *Server) resetInput() {
	*srv.input = "> "
}

func (srv *Server) appendInput(in string) {
	*srv.input = *srv.input + in
}

func (srv *Server) backspaceInput() {
	if len(srv.inputTextBox.Text) > 2 {
		*srv.input = (*srv.input)[0 : len(*srv.input)-1]
		srv.RenderOnly(InputTextBox)
	}
}

func (srv *Server) resize() {
	srv.statusBar.SetRect(0, srv.ScreenHeight-3, srv.ScreenWidth, srv.ScreenHeight)
	srv.inputTextBox.SetRect(0, srv.ScreenHeight-srv.statusBar.Dy()-1, srv.ScreenWidth, srv.ScreenHeight-srv.statusBar.Dy())
	srv.mainWindow.SetRect(0, 0, srv.ScreenWidth, srv.ScreenHeight-srv.statusBar.Dy()-srv.inputTextBox.Dy())
}

func (srv *Server) Start() {
	nicklist := widgets.NewTable()
	nicklist.Rows = [][]string{}
	nicklist.Border = false
	nicklist.BorderStyle.Fg = ui.ColorBlack
	nicklist.RowSeparator = false
	nicklist.Title = "Users"
	nicklist.TextAlignment = ui.AlignRight
	nicklist.PaddingRight = 1

	chat := widgets.NewList()
	chat.Rows = []string{}
	chat.BorderStyle.Fg = colors.DodgerBlue1
	chat.Border = true
	chat.PaddingLeft = 1
	chat.PaddingRight = 1

	statusbar := widgets.NewTabPane("  0 ", " 1 ", " 2 ")
	statusbar.SetRect(0, srv.ScreenHeight-3, srv.ScreenWidth, srv.ScreenHeight)
	statusbar.ActiveTabStyle.Fg = colors.DodgerBlue1
	statusbar.Border = true
	statusbar.BorderTop = true
	statusbar.BorderLeft = false
	statusbar.BorderRight = false
	statusbar.BorderBottom = false
	statusbar.BorderStyle.Fg = colors.DodgerBlue1

	input := widgets.NewParagraph()

	input.Border = false
	input.Text = "> "

	window := ui.NewGrid()

	window.Set(
		ui.NewCol(.9, chat),
		ui.NewCol(.1, nicklist),
	)

	chan0 := Channel{
		name:  "##somechan",
		lines: []string{"<veonik> this is a test", "<squishyj> this is only a test"},
		users: []string{"@veonik", "+squishyj"},
	}
	chan1 := Channel{
		name:  "#uwot",
		lines: []string{"<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi", "<trub0> u wot m8", "<veonik> hi"},
		users: []string{"trub0", "veonik"},
	}
	chan2 := Channel{
		name:  "#squishyslab",
		lines: []string{"<angrywombat> i dont think thats right", "<veonik> you are right"},
		users: []string{"angrywombat", "veonik"},
	}

	srv.windows = []Window{&chan0, &chan1, &chan2}
	srv.statusBar = statusbar
	srv.mainWindow = window
	srv.inputTextBox = input
	srv.chatPane = chat
	srv.nicklistPane = nicklist

	srv.input = &srv.inputTextBox.Text

	srv.resize()
	srv.Render()

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
				"x": mouse.X,
				"y": mouse.Y,
				"drag": mouse.Drag,
			})
		case ui.ResizeEvent:
			resize, ok := e.Payload.(ui.Resize)
			if !ok {
				panic(fmt.Sprintf("received termui Resize event but Payload was unexpected type %T", e.Payload))
			}
			srv.ScreenHeight = resize.Height
			srv.ScreenWidth	= resize.Width
			srv.resize()
			srv.events.Emit("ui.RESIZE", map[string]interface{}{
				"width": resize.Width,
				"height": resize.Height,
			})
			srv.Render()
		}
	}
}
