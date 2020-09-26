// import "code.dopame.me/veonik/squirssi"
package squirssi

import (
	"fmt"
	"strings"
	"sync"

	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	ui "github.com/gizak/termui/v3"
	tb "github.com/nsf/termbox-go"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squirssi/colors"
)

type logFormatter struct{}

func (f *logFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	lvl := ""
	switch entry.Level {
	case logrus.InfoLevel:
		lvl = "[INFO ](fg:blue)"
	case logrus.DebugLevel:
		lvl = "[DEBUG](fg:white,bg:blue)"
	case logrus.WarnLevel:
		lvl = "[WARN ](fg:yellow)"
	case logrus.ErrorLevel:
		lvl = "[ERROR](fg:red)"
	case logrus.FatalLevel:
		lvl = "[FATAL](fg:white,bg:red,mod:bold)"
	case logrus.TraceLevel:
		lvl = "[TRACE](fg:white,mod:bold)"
	case logrus.PanicLevel:
		lvl = "[PANIC](fg:white,bg:red,mod:bold)"
	}
	return []byte(fmt.Sprintf("%s[│](fg:grey) [%s](fg:gray100)", lvl, entry.Message)), nil
}

type HistoryManager struct {
	histories map[Window][]ModedText
	cursors   map[Window]int

	mu sync.Mutex
}

func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		histories: make(map[Window][]ModedText),
		cursors:   make(map[Window]int),
	}
}

func (hm *HistoryManager) Append(win Window, input ModedText) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.cursors[win] = len(hm.histories[win])
	hm.append(win, input)
	hm.cursors[win] = len(hm.histories[win])
}

func (hm *HistoryManager) Insert(win Window, input ModedText) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	if hm.current(win) == input {
		return
	}
	hm.append(win, input)
}

func (hm *HistoryManager) append(win Window, input ModedText) {
	hm.histories[win] = append(append(append([]ModedText{}, hm.histories[win][:hm.cursors[win]]...), input), hm.histories[win][hm.cursors[win]:]...)
}

func (hm *HistoryManager) current(win Window) ModedText {
	if hm.cursors[win] < 0 {
		hm.cursors[win] = 0
	}
	if hm.cursors[win] >= len(hm.histories[win]) {
		hm.cursors[win] = len(hm.histories[win])
		return ModedText{}
	}
	return hm.histories[win][hm.cursors[win]]
}

func (hm *HistoryManager) Current(win Window) ModedText {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	return hm.current(win)
}

func (hm *HistoryManager) Previous(win Window) ModedText {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.cursors[win] -= 1
	res := hm.current(win)
	return res
}

func (hm *HistoryManager) Next(win Window) ModedText {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.cursors[win] += 1
	res := hm.current(win)
	return res
}

type Tabber struct {
	active bool

	input   string
	match   string
	matches []string
	extra   string
	pos     int

	mu sync.Mutex
}

func NewTabber() *Tabber {
	return &Tabber{}
}

func (t *Tabber) Active() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.active
}

func (t *Tabber) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.active = false
}

func (t *Tabber) Reset(input string, channel *Channel) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	parts := strings.Split(input, " ")
	t.match = parts[len(parts)-1]
	t.extra = ""
	if t.match == parts[0] {
		t.extra = ": "
	}
	var m []string
	for _, v := range channel.Users() {
		if strings.HasPrefix(v, t.match) {
			m = append(m, v+t.extra)
		}
	}
	// put the match on the end of the stack so we can tab back to it.
	m = append(m, t.match)
	t.input = input
	t.matches = m
	t.pos = 0
	t.active = true
	return strings.Replace(input, t.match, t.matches[t.pos], 1)
}

func (t *Tabber) Tab() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.active {
		return ""
	}
	t.pos++
	if t.pos >= len(t.matches) {
		t.pos = 0
	}
	return strings.Replace(t.input, t.match, t.matches[t.pos], 1)
}

// A Server handles user interaction and displaying screen elements.
type Server struct {
	*logrus.Logger

	screenWidth, screenHeight int
	pageWidth, pageHeight     int

	mainWindow *ui.Grid
	statusBar  *ActivityTabPane

	inputTextBox *ModedTextInput
	chatPane     *ChatPane
	userListPane *UserList

	tabber *Tabber

	events *event.Dispatcher
	irc    *irc.Manager

	currentNick string

	WindowManager  *WindowManager
	HistoryManager *HistoryManager

	mu   sync.RWMutex
	done chan struct{}

	interrupt Interrupter
}

// NewServer creates a new server.
func NewServer(ev *event.Dispatcher, irc *irc.Manager) (*Server, error) {
	srv := &Server{
		Logger: logrus.StandardLogger(),

		events: ev,
		irc:    irc,

		tabber: NewTabber(),

		done: make(chan struct{}),
	}
	srv.initUI()
	srv.HistoryManager = NewHistoryManager()
	srv.WindowManager = NewWindowManager(ev)
	srv.Logger.SetOutput(srv.WindowManager.status)
	srv.Logger.SetFormatter(&logFormatter{})
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

func (srv *Server) initUI() {
	ui.StyleParserColorMap["gray"] = colors.Grey35
	ui.StyleParserColorMap["grey"] = colors.Grey35
	ui.StyleParserColorMap["gray82"] = colors.Grey82
	ui.StyleParserColorMap["grey82"] = colors.Grey82
	ui.StyleParserColorMap["gray100"] = colors.Grey100
	ui.StyleParserColorMap["grey100"] = colors.Grey100

	srv.userListPane = NewUserList()
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

	srv.chatPane = NewChatPane()
	srv.chatPane.Rows = []string{}
	srv.chatPane.BorderStyle.Fg = colors.DodgerBlue1
	srv.chatPane.Border = true
	srv.chatPane.PaddingLeft = 1
	srv.chatPane.PaddingRight = 1
	srv.chatPane.WrapText = true
	srv.chatPane.TitleStyle.Fg = colors.Grey100
	srv.chatPane.SubTitleStyle.Fg = colors.White
	srv.chatPane.ModeStyle.Fg = colors.Grey42

	srv.statusBar = NewActivityTabPane()
	srv.statusBar.ActiveTabStyle.Fg = colors.DodgerBlue1
	srv.statusBar.NoticeStyle = ui.NewStyle(colors.White, colors.DodgerBlue1)
	srv.statusBar.ActivityStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite)
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
	srv.statusBar.ActiveTabIndex = srv.WindowManager.ActiveIndex()
	win := srv.WindowManager.Active()
	if win == nil {
		return
	}
	win.Touch()
	srv.statusBar.TabNames, srv.statusBar.TabsWithActivity = srv.WindowManager.tabNames()
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
	srv.chatPane.LeftPadding = 12
	if srv.statusBar.ActiveTabIndex != 0 {
		srv.chatPane.LeftPadding = 17
	}
	srv.mainWindow.Items = nil
	if v, ok := win.(WindowWithUserList); ok {
		srv.userListPane.Rows = v.UserList()
		srv.userListPane.Title = fmt.Sprintf("%d users", len(srv.userListPane.Rows))
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
	ui.Render(srv.mainWindow, srv.statusBar, srv.inputTextBox)
	srv.pageHeight = srv.chatPane.Inner.Dy()
	srv.pageWidth = srv.chatPane.Inner.Dx()
}

func (srv *Server) resize(w, h int) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.screenHeight = h
	srv.screenWidth = w
	// guess the page height and width based on screen size
	// the actual size will be updated after rendering occurs
	srv.pageHeight = h - 8
	srv.pageWidth = int(float64(w)*.9) - 8
	srv.statusBar.SetRect(0, srv.screenHeight-2, srv.screenWidth, srv.screenHeight)
	srv.inputTextBox.SetRect(0, srv.screenHeight-srv.statusBar.Dy()-1, srv.screenWidth, srv.screenHeight-srv.statusBar.Dy())
	srv.mainWindow.SetRect(0, 0, srv.screenWidth, srv.screenHeight-srv.statusBar.Dy()-srv.inputTextBox.Dy())
}

// From https://github.com/gizak/termui/issues/255
func DisableMouseInput() {
	tb.SetInputMode(tb.InputEsc)
}

// Start begins the UI event loop and does the initial render.
func (srv *Server) Start() error {
	if err := ui.Init(); err != nil {
		return err
	}
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
				srv.Update()
				srv.Render()
				srv.events.Emit("ui.RESIZE", map[string]interface{}{
					"width":  resize.Width,
					"height": resize.Height,
				})
			}
		}
	}
}
