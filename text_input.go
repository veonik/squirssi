package squirssi

import (
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

// Consume returns and clears the current input in the TextInput.
func (i *TextInput) Consume() string {
	defer i.Reset()
	if i.Len() < 1 {
		return ""
	}
	return i.Text[i.prefixLen : len(i.Text)-i.cursorLen]
}

// Len returns the length of the contents of the TextInput.
func (i *TextInput) Len() int {
	return len(i.Text) - i.cursorLen - i.prefixLen
}

// Reset the contents of the TextInput.
func (i *TextInput) Reset() {
	prefix := ""
	if i.Prefix != nil {
		prefix = i.Prefix()
	}
	i.prefixLen = len(prefix)
	i.Text = prefix + i.cursor
}

// Append adds the given string to the end of the editable content.
func (i *TextInput) Append(in string) {
	i.Text = i.Text[0:len(i.Text)-i.cursorLen] + in + i.cursor
}

// Remove the last character from the end of the editable content.
func (i *TextInput) Backspace() {
	if len(i.Paragraph.Text) > i.prefixLen+i.cursorLen {
		i.Text = (i.Text)[0:len(i.Text)-i.cursorLen-1] + i.cursor
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
	return i.mode
}

// ToggleMode switches between the editing modes.
func (i *ModedTextInput) ToggleMode() {
	if i.mode == ModeMessage {
		i.mode = ModeCommand
	} else {
		i.mode = ModeMessage
	}
	i.Reset()
}

// ModedText is some text with an editing mode specified.
type ModedText struct {
	Kind InputMode
	Text string
}

// Consume returns and clears the ModedText in the ModedTextInput.
func (i *ModedTextInput) Consume() ModedText {
	return ModedText{
		Kind: i.mode,
		Text: i.TextInput.Consume(),
	}
}
