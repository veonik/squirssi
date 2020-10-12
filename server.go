// import "code.dopame.me/veonik/squirssi"
package squirssi

import (
	"fmt"
	"os"
	"sync"
	"time"

	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	ui "github.com/gizak/termui/v3"
	tb "github.com/nsf/termbox-go"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squirssi/colors"
	"code.dopame.me/veonik/squirssi/widget"
)

var Version = "SNAPSHOT"

// A Server handles user interaction and displaying screen elements.
type Server struct {
	*logrus.Logger
	outputLogHook *logFileWriterHook

	screenWidth, screenHeight int
	pageSize                  int

	mainWindow   *ui.Grid
	statusBar    *widget.StatusBarPane
	inputTextBox *widget.ModedTextInput
	chatPane     *widget.ChatPane
	userListPane *widget.UserList

	events *event.Dispatcher
	irc    *irc.Manager

	currentNick string

	wm      *WindowManager
	history *HistoryManager
	tabber  *TabCompleter

	mu   sync.RWMutex
	done chan struct{}

	interrupt Interrupter

	debounce bool
}

// NewServer creates a new server.
func NewServer(ev *event.Dispatcher, irc *irc.Manager) (*Server, error) {
	srv := &Server{
		Logger:        logrus.StandardLogger(),
		outputLogHook: newLogFileWriterHook(),

		events: ev,
		irc:    irc,

		wm:      NewWindowManager(ev),
		history: NewHistoryManager(),
		tabber:  NewTabCompleter(),

		done: make(chan struct{}),
	}
	srv.initUI()
	srv.Logger.SetOutput(srv.wm.Index(0))
	srv.Logger.SetFormatter(&statusFormatter{})
	srv.Logger.AddHook(srv.outputLogHook)
	return srv, nil
}

type Interrupter func()

func (srv *Server) OnInterrupt(fn Interrupter) *Server {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.interrupt = fn
	return srv
}

func (srv *Server) IRCDoAsync(fn func(conn *irc.Connection) error) {
	go func() {
		err := srv.irc.Do(fn)
		if err != nil {
			logrus.Errorln("irc command failed:", err)
		}
	}()
}

func (srv *Server) CurrentNick() string {
	srv.mu.RLock()
	defer srv.mu.RUnlock()
	return srv.currentNick
}

func (srv *Server) setCurrentNick(newNick string) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.currentNick = newNick
}

func (srv *Server) initUI() {
	ui.StyleParserColorMap["gray"] = colors.Grey35
	ui.StyleParserColorMap["grey"] = colors.Grey35
	ui.StyleParserColorMap["gray82"] = colors.Grey82
	ui.StyleParserColorMap["grey82"] = colors.Grey82
	ui.StyleParserColorMap["gray100"] = colors.Grey100
	ui.StyleParserColorMap["grey100"] = colors.Grey100
	ui.StyleParserColorMap["red4"] = colors.Red4

	srv.userListPane = widget.NewUserList()
	srv.userListPane.Rows = []string{}
	srv.userListPane.Border = true
	srv.userListPane.BorderRight = false
	srv.userListPane.BorderLeft = false
	srv.userListPane.BorderTop = true
	srv.userListPane.BorderBottom = true
	srv.userListPane.BorderStyle.Fg = colors.Grey42
	srv.userListPane.Title = "Users"
	srv.userListPane.PaddingRight = 0
	srv.userListPane.TitleStyle.Fg = colors.Grey100

	srv.chatPane = widget.NewChatPane()
	srv.chatPane.Rows = []string{}
	srv.chatPane.BorderStyle.Fg = colors.DodgerBlue1
	srv.chatPane.Border = true
	srv.chatPane.PaddingLeft = 1
	srv.chatPane.PaddingRight = 1
	srv.chatPane.WrapText = true
	srv.chatPane.TitleStyle.Fg = colors.Grey100
	srv.chatPane.SubTitleStyle.Fg = colors.White
	srv.chatPane.ModeStyle.Fg = colors.Grey42

	srv.statusBar = widget.NewStatusBarPane()
	srv.statusBar.ActiveTabStyle.Fg = colors.DodgerBlue1
	srv.statusBar.NoticeStyle = ui.NewStyle(colors.White, colors.DodgerBlue1)
	srv.statusBar.ActivityStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite)
	srv.statusBar.Border = true
	srv.statusBar.BorderTop = true
	srv.statusBar.BorderLeft = false
	srv.statusBar.BorderRight = false
	srv.statusBar.BorderBottom = false
	srv.statusBar.BorderStyle.Fg = colors.DodgerBlue1

	srv.inputTextBox = widget.NewModedTextInput()
	srv.inputTextBox.Border = false

	srv.mainWindow = ui.NewGrid()
}

// Close ends the UI session, returning control of stdout.
func (srv *Server) Close() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	select {
	case <-srv.done:
		// already closing
		return
	default:
		ui.Close()
		close(srv.done)
	}
}

// Update refreshes the state of the UI but stops short of rendering.
func (srv *Server) Update() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.statusBar.ActiveTabIndex = srv.wm.ActiveIndex()
	win := srv.wm.Active()
	if win == nil {
		return
	}
	win.Touch()
	srv.statusBar.TabNames, srv.statusBar.TabsWithActivity = srv.wm.TabNames()
	srv.chatPane.SelectedRow = win.CurrentLine()
	srv.chatPane.Rows = win.Lines()
	srv.chatPane.Title = win.Title()

	if ch, ok := win.(*Channel); ok {
		srv.chatPane.SubTitle = ch.Topic()
		srv.chatPane.ModeText = ch.Modes()
	} else {
		srv.chatPane.SubTitle = ""
		srv.chatPane.ModeText = ""
	}
	srv.chatPane.LeftPadding = win.padding() + 7
	if srv.statusBar.ActiveTabIndex == 0 {
		srv.chatPane.ModeText = srv.currentNick
	}
	srv.mainWindow.Items = nil
	if v, ok := win.(WindowWithUserList); ok {
		srv.userListPane.Rows = v.UserList()
		suff := "s"
		if len(srv.userListPane.Rows) == 1 {
			suff = ""
		}
		srv.userListPane.Title = fmt.Sprintf("%d user%s", len(srv.userListPane.Rows), suff)
		srv.mainWindow.Set(
			ui.NewCol(.85, srv.chatPane),
			ui.NewCol(.15, srv.userListPane),
		)
	} else {
		srv.mainWindow.Set(
			ui.NewCol(1, srv.chatPane),
		)
	}
}

type screenElement int

const (
	InputTextBox screenElement = iota
	StatusBar
	MainWindow
)

// RenderOnly renders select screen elements rather than the whole screen.
func (srv *Server) RenderOnly(items ...screenElement) {
	srv.mu.RLock()
	defer srv.mu.RUnlock()
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

// Render renders the current state to the screen.
func (srv *Server) Render() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.debounce {
		return
	}
	srv.debounce = true
	go func() {
		<-time.After(1 * time.Millisecond)
		srv.doRender(false)
	}()
}

func (srv *Server) doRender(force bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	ui.Render(srv.mainWindow, srv.statusBar, srv.inputTextBox)
	srv.pageSize = srv.chatPane.Inner.Dy()
	if !force {
		srv.debounce = false
	}
}

func (srv *Server) resize(w, h int) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.screenHeight = h
	srv.screenWidth = w
	// guess the page size for now, will be corrected after the first render.
	srv.pageSize = h - 8
	srv.statusBar.SetRect(0, srv.screenHeight-2, srv.screenWidth, srv.screenHeight)
	srv.inputTextBox.SetRect(0, srv.screenHeight-srv.statusBar.Dy()-1, srv.screenWidth, srv.screenHeight-srv.statusBar.Dy())
	srv.mainWindow.SetRect(0, 0, srv.screenWidth, srv.screenHeight-srv.statusBar.Dy()-srv.inputTextBox.Dy())
}

// From https://github.com/gizak/termui/issues/255
func DisableMouseInput() {
	tb.SetInputMode(tb.InputAlt)
}

// Start begins the UI event loop and does the initial render.
func (srv *Server) Start() error {
	if err := ui.Init(); err != nil {
		return err
	}
	srv.outputLogHook.Start()
	DisableMouseInput()
	w, h := ui.TerminalDimensions()
	bindUIHandlers(srv, srv.events)
	bindIRCHandlers(srv, srv.events)
	srv.inputTextBox.Reset()
	srv.resize(w, h)
	srv.Update()
	srv.Render()

	go srv.startUIEventLoop()

	return nil
}

func (srv *Server) startUIEventLoop() {
	uiEvents := ui.PollEvents()

	for {
		select {
		case <-srv.done:
			// srv.Close() was called, no need to continue
			return
		case e := <-uiEvents:
			switch e.Type {
			case ui.KeyboardEvent:
				// handle keyboard input outside of the event emitter to avoid
				// too long a delay between keypress and the UI reacting.
				onUIKeyPress(srv, e.ID)
				srv.events.Emit("ui.KEYPRESS", map[string]interface{}{
					"key": e.ID,
				})
			case ui.ResizeEvent:
				resize, ok := e.Payload.(ui.Resize)
				if !ok {
					panic(fmt.Sprintf("received termui Resize event but Payload was unexpected type %T", e.Payload))
				}
				srv.resize(resize.Width, resize.Height)
				fmt.Fprintf(os.Stderr, "resize event: new size %dx%d\n", resize.Width, resize.Height)
				// logrus.Debugf("resize event: new size %dx%d", resize.Width, resize.Height)
				srv.events.Emit("ui.RESIZE", map[string]interface{}{
					"width":  resize.Width,
					"height": resize.Height,
				})
				srv.doRender(true)
				srv.events.Emit("ui.DIRTY", nil)
			}
		}
	}
}
