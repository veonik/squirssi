package squirssi

type Window interface {
	Title() string
	Lines() []string
	HasActivity() bool
	CurrentLine() int
}

type WindowWithUserList interface {
	Window
	Users() []string
}

type bufferedWindow struct {
	lines   []string
	current int

	hasUnseen bool
}

func (c *bufferedWindow) HasActivity() bool {
	return c.hasUnseen
}

func (c *bufferedWindow) Lines() []string {
	return c.lines
}

func (c *bufferedWindow) CurrentLine() int {
	return c.current
}

type Status struct {
	bufferedWindow
}

func (c *Status) Title() string {
	return "status"
}

type Channel struct {
	bufferedWindow

	name  string
	topic string
	modes string
	users []string
}

func (c *Channel) Title() string {
	return c.name
}

func (c *Channel) Users() []string {
	return c.users
}

type DirectMessage struct {
	bufferedWindow

	user string
}

func (c *DirectMessage) Title() string {
	return c.user
}
