package widget

import (
	"image"
	"strings"

	ui "github.com/gizak/termui/v3"
	rw "github.com/mattn/go-runewidth"
	"github.com/mitchellh/go-wordwrap"

	"code.dopame.me/veonik/squirssi/colors"
)

// A ChatPane contains the messages for the screen.
// This widget is based on the termui List widget.
// ChatPanes support both termui formatting as well as IRC formatting.
type ChatPane struct {
	ui.Block
	Rows        []string
	WrapText    bool
	TextStyle   ui.Style
	SelectedRow int
	LeftPadding int

	ModeText  string
	ModeStyle ui.Style

	SubTitle      string
	SubTitleStyle ui.Style
}

func NewChatPane() *ChatPane {
	return &ChatPane{
		Block:     *ui.NewBlock(),
		TextStyle: ui.Theme.List.Text,
	}
}

func WrapCellsPadded(cells []ui.Cell, width uint, leftPadding int) []ui.Cell {
	str := ui.CellsToString(cells)
	wrapped := wordwrap.WrapString(str, width)
	wrappedCells := []ui.Cell{}
	i := 0
	twoLines := false
	printPipe := false
loop:
	for x, _rune := range wrapped {
		if _rune == '│' {
			printPipe = true
		}
		if _rune == '\n' {
			wrappedCells = append(wrappedCells, ui.Cell{_rune, ui.StyleClear})
			for j := 0; j < leftPadding; j++ {
				wrappedCells = append(wrappedCells, ui.Cell{' ', ui.StyleClear})
			}
			if printPipe {
				wrappedCells = append(wrappedCells, ui.Cell{ui.VERTICAL_LINE, ui.NewStyle(colors.Grey35)})
			}
			wrappedCells = append(wrappedCells, ui.Cell{' ', ui.StyleClear})
			if !twoLines {
				// the first time we wrap, we use the full available width, but the
				// next lines are padded before they starts so that the text lines
				// up on all lines with the nick and timestamp in a "gutter".
				// so after wrapping the first time, recalculate the wrapping using the
				// padded width. this only needs to happen once.
				lPad := uint(leftPadding)
				if printPipe {
					lPad++
				}
				wrapped = wordwrap.WrapString(strings.ReplaceAll(wrapped[x+1:], "\n", " "), width-lPad)
				twoLines = true
				goto loop
			}
		} else {
			wrappedCells = append(wrappedCells, ui.Cell{_rune, cells[i].Style})
		}
		i++
	}
	return wrappedCells
}

func (cp *ChatPane) Draw(buf *ui.Buffer) {
	cp.Block.Draw(buf)

	tcells := ui.ParseStyles(cp.Title, cp.TitleStyle)
	if cp.ModeText != "" {
		tcells = append(tcells, ui.Cell{'(', cp.TitleStyle})
		tcells = append(tcells, ui.RunesToStyledCells([]rune(cp.ModeText), cp.ModeStyle)...)
		tcells = append(tcells, ui.Cell{')', cp.TitleStyle})
	}
	if cp.SubTitle != "" {
		tcells = append(tcells, ui.RunesToStyledCells([]rune{ui.HORIZONTAL_LINE, ui.HORIZONTAL_LINE}, ui.NewStyle(colors.Grey42))...)
		tcells = append(tcells, ui.ParseStyles(cp.SubTitle, cp.SubTitleStyle)...)
	}
	pt := image.Pt(cp.Min.X+2, cp.Min.Y)
	if cp.Max.X > 0 && len(tcells) >= cp.Max.X-5 {
		tcells = append(tcells[:cp.Max.X-5], ui.Cell{ui.ELLIPSES, cp.SubTitleStyle})
	}
	for i := 0; i < len(tcells); i++ {
		cc := tcells[i]
		buf.SetCell(cc, pt)
		pt.X++
	}

	point := cp.Inner.Min

	rows := make([]int, len(cp.Rows))
	actuals := [][]ui.Cell{}
	actualLen := 0
	for i, o := range cp.Rows {
		c := ui.ParseStyles(o, cp.TextStyle)
		c = ParseIRCStyles(c)
		if cp.WrapText {
			c = WrapCellsPadded(c, uint(cp.Inner.Dx()-1), cp.LeftPadding)
		}
		p := ui.SplitCells(c, '\n')
		l := len(p)
		e := actualLen + l
		actualLen = e
		rows[i] = e - 1
		for j := 0; j < l; j++ {
			actuals = append(actuals, p[j])
		}
	}

	// row that would actually be selected after text wrapping is done
	actualSelected := 0
	if len(rows) > cp.SelectedRow {
		actualSelected = rows[cp.SelectedRow]
	}
	topRow := 0

	// adjust starting row based on the bounding box and the actual selected row
	if actualSelected >= cp.Inner.Dy()+topRow {
		topRow = actualSelected - cp.Inner.Dy() + 1
	} else if actualSelected < topRow {
		topRow = actualSelected
	}

	// draw the already wrapped rows
	for row := topRow; row < len(actuals) && point.Y < cp.Inner.Max.Y; row++ {
		cells := actuals[row]
		for j := 0; j < len(cells) && point.Y < cp.Inner.Max.Y; j++ {
			style := cells[j].Style
			if cells[j].Rune == '\n' {
				point = image.Pt(cp.Inner.Min.X, point.Y+1)
			} else {
				if point.X+1 == cp.Inner.Max.X+1 && len(cells) > cp.Inner.Dx() {
					buf.SetCell(ui.NewCell(ui.ELLIPSES, style), point.Add(image.Pt(-1, 0)))
					break
				} else {
					buf.SetCell(ui.NewCell(cells[j].Rune, style), point)
					point = point.Add(image.Pt(rw.RuneWidth(cells[j].Rune), 0))
				}
			}
		}
		point = image.Pt(cp.Inner.Min.X, point.Y+1)
	}

	// draw UP_ARROW if needed
	if topRow > 0 {
		buf.SetCell(
			ui.NewCell(ui.UP_ARROW, ui.NewStyle(ui.ColorWhite)),
			image.Pt(cp.Inner.Max.X-1, cp.Inner.Min.Y),
		)
	}

	// draw DOWN_ARROW if needed
	if len(cp.Rows) > topRow+cp.Inner.Dy() {
		buf.SetCell(
			ui.NewCell(ui.DOWN_ARROW, ui.NewStyle(ui.ColorWhite)),
			image.Pt(cp.Inner.Max.X-1, cp.Inner.Max.Y-1),
		)
	}
}
