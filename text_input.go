package squirssi

import (
	"strings"

	"github.com/gizak/termui/v3/widgets"
)

const CursorFullBlock = "█"

// A TextInput is a Paragraph widget with editable contents.
// A cursor block character is printed at the end of the contents and
// transparently handled when updating those contents.
type TextInput struct {
	*widgets.Paragraph

	// Character used to indicate the TextInput is focused and
	// awaiting input.
	cursor string
	// Length of the cursor character at the end of the input.
	cursorLen int

	// Length of the current prefix in the text.
	// Calculated only when resetting.
	prefixLen int

	// If specified, called during Reset and the result is used
	// as the initial text in the text box.
	Prefix func() string
}

func NewTextInput(cursor string) *TextInput {
	return &TextInput{
		Paragraph: widgets.NewParagraph(),
		cursor:    cursor,
		cursorLen: len(cursor),
	}
}

// Peek returns the current input in the TextInput without clearing.
func (i *TextInput) Peek() string {
	if i.Len() < 1 {
		return ""
	}
	i.Lock()
	defer i.Unlock()
	i.Text = strings.Replace(i.Text, "[C](mod:reverse)", string(0x03),-1)
	i.Text = strings.Replace(i.Text, "[B](mod:reverse)", string(0x02),-1)
	i.Text = strings.Replace(i.Text, "[U](mod:reverse)", string(0x1F),-1)
	t := i.Text[i.prefixLen : len(i.Text)-i.cursorLen]
	i.Text = strings.Replace(i.Text, string(0x03), "[C](mod:reverse)", -1)
	i.Text = strings.Replace(i.Text, string(0x02), "[B](mod:reverse)", -1)
	i.Text = strings.Replace(i.Text, string(0x1F), "[U](mod:reverse)", -1)
	return t
}

// Consume returns and clears the current input in the TextInput.
func (i *TextInput) Consume() string {
	defer i.Reset()
	if i.Len() < 1 {
		return ""
	}
	i.Lock()
	defer i.Unlock()
	t := i.Text[i.prefixLen : len(i.Text)-i.cursorLen]
	t = strings.Replace(t, "[C](mod:reverse)", string(0x03),-1)
	t = strings.Replace(t, "[B](mod:reverse)", string(0x02),-1)
	t = strings.Replace(t, "[U](mod:reverse)", string(0x1F),-1)
	return t
}

// Len returns the length of the contents of the TextInput.
func (i *TextInput) Len() int {
	i.Lock()
	defer i.Unlock()
	i.Text = strings.Replace(i.Text, "[C](mod:reverse)", string(0x03),-1)
	i.Text = strings.Replace(i.Text, "[B](mod:reverse)", string(0x02),-1)
	i.Text = strings.Replace(i.Text, "[U](mod:reverse)", string(0x1F),-1)
	l := len(i.Text) - i.cursorLen - i.prefixLen
	i.Text = strings.Replace(i.Text, string(0x03), "[C](mod:reverse)", -1)
	i.Text = strings.Replace(i.Text, string(0x02), "[B](mod:reverse)", -1)
	i.Text = strings.Replace(i.Text, string(0x1F), "[U](mod:reverse)", -1)
	return l
}

// Reset the contents of the TextInput.
func (i *TextInput) Reset() {
	i.Lock()
	defer i.Unlock()
	prefix := ""
	if i.Prefix != nil {
		prefix = i.Prefix()
	}
	i.prefixLen = len(prefix)
	i.Text = prefix + i.cursor
}

// Append adds the given string to the end of the editable content.
func (i *TextInput) Append(in string) {
	i.Lock()
	defer i.Unlock()
	in = strings.Replace(in, string(0x03), "[C](mod:reverse)", -1)
	in = strings.Replace(in, string(0x02), "[B](mod:reverse)", -1)
	in = strings.Replace(in, string(0x1F), "[U](mod:reverse)", -1)
	i.Text = i.Text[0:len(i.Text)-i.cursorLen] + in + i.cursor
}

// Remove the last character from the end of the editable content.
func (i *TextInput) Backspace() {
	i.Lock()
	defer i.Unlock()
	i.Text = strings.Replace(i.Text, "[C](mod:reverse)", string(0x03),-1)
	i.Text = strings.Replace(i.Text, "[B](mod:reverse)", string(0x02),-1)
	i.Text = strings.Replace(i.Text, "[U](mod:reverse)", string(0x1F),-1)
	if len(i.Text) > i.prefixLen+i.cursorLen {
		i.Text = (i.Text)[0:len(i.Text)-i.cursorLen-1] + i.cursor
	}
	i.Text = strings.Replace(i.Text, string(0x03), "[C](mod:reverse)", -1)
	i.Text = strings.Replace(i.Text, string(0x02), "[B](mod:reverse)", -1)
	i.Text = strings.Replace(i.Text, string(0x1F), "[U](mod:reverse)", -1)
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
func NewModedTextInput(cursor string) *ModedTextInput {
	i := &ModedTextInput{
		TextInput: *NewTextInput(cursor),
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
