// import "code.dopame.me/veonik/squirssi"
package squirssi

import (
	"fmt"
	"sync"

	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squirssi/colors"
)

// A Server handles user interaction and displaying screen elements.
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

	currentNick string

	windows []Window
	status  *Status

	mu sync.Mutex
}

type logFormatter struct{}

func (f *logFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	lvl := ""
	switch entry.Level {
	case logrus.InfoLevel:
		lvl = "[INFO ](fg:blue)"
	case logrus.DebugLevel:
		lvl = "[DEBUG](fg:white)"
	case logrus.WarnLevel:
		lvl = "[WARN ](fg:yellow)"
	case logrus.ErrorLevel:
		lvl = "[ERROR](fg:red)"
	case logrus.FatalLevel:
		lvl = "[FATAL](fg:white,bg:red)"
	case logrus.TraceLevel:
		lvl = "[TRACE](fg:white)"
	case logrus.PanicLevel:
		lvl = "[PANIC](fg:white,bg:red)"
	}
	return []byte(fmt.Sprintf("%s: %s", lvl, entry.Message)), nil
}

// NewServer creates a new server.
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

		status: &Status{bufferedWindow: newBufferedWindow("status", ev)},
	}
	srv.windows = []Window{srv.status}
	srv.Logger.SetOutput(srv.status)
	srv.Logger.SetFormatter(&logFormatter{})
	srv.initUI()
	return srv, nil
}

// WindowNamed returns the window with the given name, if it exists.
func (srv *Server) WindowNamed(name string) Window {
	var win Window
	srv.mu.Lock()
	defer srv.mu.Unlock()
	for _, w := range srv.windows {
		if w.Title() == name {
			win = w
			break
		}
	}
	return win
}

// CloseWindow closes a window denoted by tab index.
func (srv *Server) CloseWindow(ch int) {
	if ch == 0 {
		logrus.Warnln("CloseWindow: cant close status window")
		return
	}
	srv.mu.Lock()
	if ch >= len(srv.windows) {
		logrus.Warnf("CloseWindow: no window #%d", ch)
		srv.mu.Unlock()
		return
	}
	srv.windows = append(srv.windows[:ch], srv.windows[ch+1:]...)
	if ch >= srv.statusBar.ActiveTabIndex {
		srv.statusBar.ActiveTabIndex = ch - 1
	}
	srv.mu.Unlock()
	srv.Update()
	srv.Render()
}

// ScrollPageUp scrolls one opage up in the current window.
func (srv *Server) ScrollPageUp() {
	srv.mu.Lock()
	srv.chatPane.ScrollPageUp()
	row := srv.chatPane.SelectedRow
	channel := srv.windows[srv.statusBar.ActiveTabIndex]
	srv.mu.Unlock()
	if channel == nil {
		return
	}
	channel.ScrollTo(row)
	srv.Render()
}

// ScrollPageDown scrolls one page down in the current window.
func (srv *Server) ScrollPageDown() {
	srv.mu.Lock()
	srv.chatPane.ScrollPageDown()
	row := srv.chatPane.SelectedRow
	channel := srv.windows[srv.statusBar.ActiveTabIndex]
	srv.mu.Unlock()
	if channel == nil {
		return
	}
	channel.ScrollTo(row)
	srv.Render()
}

// ScrollTop scrolls to the top of the current window.
func (srv *Server) ScrollTop() {
	srv.mu.Lock()
	srv.chatPane.ScrollTop()
	row := srv.chatPane.SelectedRow
	channel := srv.windows[srv.statusBar.ActiveTabIndex]
	srv.mu.Unlock()
	if channel == nil {
		return
	}
	channel.ScrollTo(row)
	srv.Render()
}

// ScrollBottom scrolls to the end of the current window.
func (srv *Server) ScrollBottom() {
	srv.mu.Lock()
	srv.chatPane.ScrollBottom()
	channel := srv.windows[srv.statusBar.ActiveTabIndex]
	srv.mu.Unlock()
	if channel == nil {
		return
	}
	channel.ScrollTo(-1)
	srv.Render()
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
	srv.chatPane.WrapText = true

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
}

// Close ends the UI session, returning control of stdout.
func (srv *Server) Close() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	ui.Close()
}

func tabNames(windows []Window, active int) ([]string, map[int]struct{}) {
	res := make([]string, len(windows))
	activity := make(map[int]struct{})
	for i := 0; i < len(windows); i++ {
		win := windows[i]
		if win.HasActivity() {
			activity[i] = struct{}{}
		}
		if active == i {
			res[i] = fmt.Sprintf(" %s ", win.Title())
		} else {
			res[i] = fmt.Sprintf(" %d ", i)
		}
	}
	return res, activity
}

// Update refreshes the state of the UI but stops short of rendering.
func (srv *Server) Update() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	win := srv.windows[srv.statusBar.ActiveTabIndex]
	if win == nil {
		return
	}
	win.Touch()
	srv.statusBar.TabNames, srv.statusBar.TabsWithActivity = tabNames(srv.windows, srv.statusBar.ActiveTabIndex)
	srv.chatPane.Rows = win.Lines()
	srv.chatPane.Title = win.Title()
	srv.chatPane.SelectedRow = win.CurrentLine()
	srv.mainWindow.Items = nil
	var rows [][]string
	if v, ok := win.(WindowWithUserList); ok {
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

type screenElement int

const (
	InputTextBox screenElement = iota
	StatusBar
	MainWindow
)

func (srv *Server) preRender() {
	if len(srv.chatPane.Rows) == 0 {
		srv.chatPane.SelectedRow = 0
	} else if srv.chatPane.SelectedRow < 0 {
		srv.chatPane.ScrollBottom()
	}
}

// RenderOnly renders select screen elements rather than the whole screen.
func (srv *Server) RenderOnly(items ...screenElement) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	var its []ui.Drawable
	for _, it := range items {
		switch it {
		case InputTextBox:
			its = append(its, srv.inputTextBox)
		case StatusBar:
			its = append(its, srv.statusBar)
		case MainWindow:
			srv.preRender()
			its = append(its, srv.mainWindow)
		}
	}
	ui.Render(its...)
}

// Render renders the current state to the screen.
func (srv *Server) Render() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.preRender()
	ui.Render(srv.mainWindow, srv.statusBar, srv.inputTextBox)
}

func (srv *Server) resize() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.statusBar.SetRect(0, srv.ScreenHeight-3, srv.ScreenWidth, srv.ScreenHeight)
	srv.inputTextBox.SetRect(0, srv.ScreenHeight-srv.statusBar.Dy()-1, srv.ScreenWidth, srv.ScreenHeight-srv.statusBar.Dy())
	srv.mainWindow.SetRect(0, 0, srv.ScreenWidth, srv.ScreenHeight-srv.statusBar.Dy()-srv.inputTextBox.Dy())
}

// Start begins the UI event loop and does the initial render.
func (srv *Server) Start() {
	srv.bind()
	srv.inputTextBox.Reset()
	srv.resize()
	srv.Update()
	srv.Render()

	uiEvents := ui.PollEvents()

	for {
		e := <-uiEvents
		switch e.Type {
		case ui.KeyboardEvent:
			// handle keyboard input outside of the event emitter to avoid
			// too long a delay between keypress and the UI reacting.
			srv.onUIKeyPress(e.ID)
			srv.RenderOnly(InputTextBox)
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
			srv.mu.Lock()
			srv.ScreenHeight = resize.Height
			srv.ScreenWidth = resize.Width
			srv.mu.Unlock()
			srv.resize()
			srv.Update()
			srv.Render()
			srv.events.Emit("ui.RESIZE", map[string]interface{}{
				"width":  resize.Width,
				"height": resize.Height,
			})
		}
	}
}
