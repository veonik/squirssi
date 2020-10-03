package widget

import (
	"image"
	"strings"

	ui "github.com/gizak/termui/v3"
	tb "github.com/nsf/termbox-go"
)

// A TextInput is a widget with editable contents.
// This widget is based on the termui Paragraph widget.
type TextInput struct {
	ui.Block
	Text      string
	TextStyle ui.Style
	WrapText  bool

	// Actual input stored in the TextInput.
	input string
	// Current cursor position within input.
	cursorPos int

	// Current prefix rendered before the actual input.
	// Set only when resetting.
	prefix string

	// If specified, called during Reset and the result is used
	// as the initial text in the text box.
	Prefix func() string
}

func NewTextInput() *TextInput {
	return &TextInput{
		Block:     *ui.NewBlock(),
		TextStyle: ui.Theme.Paragraph.Text,
		WrapText:  true,
	}
}

func (i *TextInput) Draw(buf *ui.Buffer) {
	i.Block.Draw(buf)

	cells := ui.ParseStyles(i.Text, i.TextStyle)
	cells = ParseIRCStyles(cells)
	if i.WrapText {
		cells = ui.WrapCells(cells, uint(i.Inner.Dx()))
	}

	rows := ui.SplitCells(cells, '\n')

	for y, row := range rows {
		if y+i.Inner.Min.Y >= i.Inner.Max.Y {
			break
		}
		row = ui.TrimCells(row, i.Inner.Dx())
		for _, cx := range ui.BuildCellWithXArray(row) {
			x, cell := cx.X, cx.Cell
			buf.SetCell(cell, image.Pt(x, y).Add(i.Inner.Min))
		}
	}
	tb.SetCursor(i.Inner.Min.X+len(i.prefix)+i.cursorPos, i.Inner.Min.Y)
}

func (i *TextInput) update() {
	t := i.input
	t = strings.Replace(t, string(0x1F), string(0x016)+"U"+string(0x016), -1)
	t = strings.Replace(t, string(0x03), string(0x016)+"C"+string(0x016), -1)
	t = strings.Replace(t, string(0x02), string(0x016)+"B"+string(0x016), -1)
	i.Text = i.prefix + t
}

func (i *TextInput) CursorPrev() {
	i.Lock()
	defer i.Unlock()
	if i.cursorPos <= 0 {
		return
	}
	i.cursorPos--
	i.update()
}

func (i *TextInput) CursorNext() {
	i.Lock()
	defer i.Unlock()
	if i.cursorPos >= len(i.input) {
		return
	}
	i.cursorPos++
	i.update()
}

// Peek returns the current input in the TextInput without clearing.
func (i *TextInput) Peek() string {
	i.Lock()
	defer i.Unlock()
	return i.input
}

// Consume returns and clears the current input in the TextInput.
func (i *TextInput) Consume() string {
	defer i.Reset()
	if i.Len() < 1 {
		return ""
	}
	i.Lock()
	defer i.Unlock()
	return i.input
}

// Len returns the length of the contents of the TextInput.
func (i *TextInput) Len() int {
	i.Lock()
	defer i.Unlock()
	return len(i.input)
}

// Reset the contents of the TextInput.
func (i *TextInput) Reset() {
	i.Lock()
	defer i.Unlock()
	i.cursorPos = 0
	i.input = ""
	i.prefix = i.Prefix()
	i.update()
}

// Append adds the given string to the end of the editable content.
func (i *TextInput) Append(in string) {
	i.Lock()
	defer i.Unlock()
	i.input = i.input[:i.cursorPos] + in + i.input[i.cursorPos:]
	i.cursorPos += len(in)
	i.update()
}

// Remove the last character from the end of the editable content.
func (i *TextInput) Backspace() {
	i.Lock()
	defer i.Unlock()
	if i.cursorPos > 0 {
		i.input = i.input[:i.cursorPos-1] + i.input[i.cursorPos:]
		i.cursorPos--
		i.update()
	}
}

// Remove the last character from the end of the editable content.
func (i *TextInput) DeleteNext() {
	i.Lock()
	defer i.Unlock()
	if i.cursorPos < len(i.input) {
		i.input = i.input[:i.cursorPos] + i.input[i.cursorPos+1:]
		i.update()
	}
}

// InputMode defines different kinds of input handled by a ModedTextInput.
type InputMode int

const (
	// Regular text.
	ModeMessage InputMode = iota
	// A command and arguments separated by spaces.
	ModeCommand
)

// A ModedTextInput tracks the current editing mode of a TextInput.
type ModedTextInput struct {
	TextInput

	mode InputMode
}

// NewModedTextInput creates a new ModedTextInput.
func NewModedTextInput() *ModedTextInput {
	i := &ModedTextInput{
		TextInput: *NewTextInput(),
		mode:      ModeMessage,
	}
	i.Prefix = func() string {
		if i.mode == ModeCommand {
			return "/ "
		}
		return "> "
	}
	return i
}

// Mode returns the current editing mode.
func (i *ModedTextInput) Mode() InputMode {
	i.Lock()
	defer i.Unlock()
	return i.mode
}

func (i *ModedTextInput) Set(in ModedText) {
	i.Lock()
	i.mode = in.Kind
	i.Unlock()
	i.Reset()
	i.Append(in.Text)
}

// ToggleMode switches between the editing modes.
func (i *ModedTextInput) ToggleMode() {
	i.Lock()
	if i.mode == ModeMessage {
		i.mode = ModeCommand
	} else {
		i.mode = ModeMessage
	}
	i.Unlock()
	i.Reset()
}

// ModedText is some text with an editing mode specified.
type ModedText struct {
	Kind InputMode
	Text string
}

// Consume returns and clears the ModedText in the ModedTextInput.
func (i *ModedTextInput) Consume() ModedText {
	i.Lock()
	mode := i.mode
	i.mode = ModeMessage
	i.Unlock()
	txt := i.TextInput.Consume()
	return ModedText{
		Kind: mode,
		Text: txt,
	}
}

func (i *ModedTextInput) Backspace() {
	if i.Len() == 0 {
		if i.Mode() != ModeMessage {
			i.ToggleMode()
		}
		return
	}
	i.TextInput.Backspace()
}
