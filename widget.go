package squirssi

import (
	"image"
	"strconv"
	"strings"
	"unicode"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	rw "github.com/mattn/go-runewidth"
	"github.com/mitchellh/go-wordwrap"

	"code.dopame.me/veonik/squirssi/colors"
)

type ChatPane struct {
	ui.Block
	Rows             []string
	WrapText         bool
	TextStyle        ui.Style
	SelectedRow      int
	LeftPadding      int
}

func NewChatPane() *ChatPane {
	return &ChatPane{
		Block:            *ui.NewBlock(),
		TextStyle:        ui.Theme.List.Text,
	}
}

func ParseIRCStyles(c []ui.Cell) []ui.Cell {
	var style ui.Style
	var initial ui.Style
	var cells []ui.Cell
	changed := false
	for i := 0; i < len(c); i++ {
		cc := c[i]
		if !changed {
			initial = cc.Style
			style = cc.Style
		} else {
			cc.Style = style
		}
		switch cc.Rune {
		case 0x1E:
			// strikethrough, not supported
		case 0x1F:
			changed = true
			style.Modifier ^= ui.ModifierUnderline
		case 0x1D:
			// italics, not supported
		case 0x02:
			changed = true
			style.Modifier ^= ui.ModifierBold
		case 0x016:
			changed = true
			style.Modifier ^= ui.ModifierReverse
		case 0x0F:
			changed = false
			style = initial
		case 0x04:
			// hex color, not supported
		case 0x03:
			// color
			changed = true
			fgdone := false
			fg := ""
			bg := ""
			eat := 0
			for j := i+1; j-i < 5 && j < len(c); j++ {
				cx := c[j]
				if unicode.IsDigit(cx.Rune) {
					eat++
					if !fgdone {
						fg = fg + string(cx.Rune)
					} else {
						bg = bg + string(cx.Rune)
					}
				} else if cx.Rune == ',' {
					if fg == "" {
						break
					}
					eat++
					fgdone = true
				} else {
					break
				}
			}
			i += eat
			if fg == "" && bg == "" {
				style.Fg = initial.Fg
				style.Bg = initial.Bg
			} else {
				fgi, _ := strconv.Atoi(fg)
				style.Fg = colors.IRCToUI(colors.IRC(fgi))
				if bg != "" {
					bgi, _ := strconv.Atoi(bg)
					style.Bg = colors.IRCToUI(colors.IRC(bgi))
				}
			}
			continue

		default:
			cells = append(cells, cc)
		}
	}
	return cells
}

func WrapCellsPadded(cells []ui.Cell, width uint, leftPadding int) []ui.Cell {
	str := ui.CellsToString(cells)
	wrapped := wordwrap.WrapString(str, width)
	wrappedCells := []ui.Cell{}
	i := 0
	twoLines := false
	loop:
	for x, _rune := range wrapped {
		if _rune == '\n' {
			wrappedCells = append(wrappedCells, ui.Cell{_rune, ui.StyleClear})
			for j := 0; j < leftPadding; j++ {
				wrappedCells = append(wrappedCells, ui.Cell{' ', ui.StyleClear})
			}
			wrappedCells = append(wrappedCells, ui.Cell{'|', ui.NewStyle(colors.Grey62)}, ui.Cell{' ', ui.StyleClear})
			v := []rune(wrapped)
			if !twoLines {
				wrapped = wordwrap.WrapString(strings.ReplaceAll(string(v[x+1:]), "\n", ""), width-uint(leftPadding))
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

func (self *ChatPane) Draw(buf *ui.Buffer) {
	self.Block.Draw(buf)

	point := self.Inner.Min

	rows := make([]int, len(self.Rows))
	actuals := [][]ui.Cell{}
	actualLen := 0
	for i, o := range self.Rows {
		c := ui.ParseStyles(o, self.TextStyle)
		c = ParseIRCStyles(c)
		if self.WrapText {
			c = WrapCellsPadded(c, uint(self.Inner.Dx()-1), self.LeftPadding)
		}
		p := ui.SplitCells(c, '\n')
		l := len(p)
		e := actualLen + l
		actualLen = e
		rows[i] = e-1
		for j := 0; j < l; j++ {
			actuals = append(actuals, p[j])
		}
	}

	// row that would actually be selected after text wrapping is done
	actualSelected := rows[self.SelectedRow]
	topRow := 0

	// adjust starting row based on the bounding box and the actual selected row
	if actualSelected >= self.Inner.Dy()+topRow {
		topRow = actualSelected - self.Inner.Dy() + 1
	} else if actualSelected < topRow {
		topRow = actualSelected
	}

	// draw the already wrapped rows
	for row := topRow; row < len(actuals) && point.Y < self.Inner.Max.Y; row++ {
		cells := actuals[row]
		for j := 0; j < len(cells) && point.Y < self.Inner.Max.Y; j++ {
			style := cells[j].Style
			if cells[j].Rune == '\n' {
				point = image.Pt(self.Inner.Min.X, point.Y+1)
			} else {
				if point.X+1 == self.Inner.Max.X+1 && len(cells) > self.Inner.Dx() {
					buf.SetCell(ui.NewCell(ui.ELLIPSES, style), point.Add(image.Pt(-1, 0)))
					break
				} else {
					buf.SetCell(ui.NewCell(cells[j].Rune, style), point)
					point = point.Add(image.Pt(rw.RuneWidth(cells[j].Rune), 0))
				}
			}
		}
		point = image.Pt(self.Inner.Min.X, point.Y+1)
	}

	// draw UP_ARROW if needed
	if topRow > 0 {
		buf.SetCell(
			ui.NewCell(ui.UP_ARROW, ui.NewStyle(ui.ColorWhite)),
			image.Pt(self.Inner.Max.X-1, self.Inner.Min.Y),
		)
	}

	// draw DOWN_ARROW if needed
	if len(self.Rows) > topRow+self.Inner.Dy() {
		buf.SetCell(
			ui.NewCell(ui.DOWN_ARROW, ui.NewStyle(ui.ColorWhite)),
			image.Pt(self.Inner.Max.X-1, self.Inner.Max.Y-1),
		)
	}
}

type ActivityTabPane struct {
	*widgets.TabPane

	TabsWithActivity map[int]activityType
	NoticeStyle      ui.Style
	ActivityStyle    ui.Style
}

func NewActivityTabPane() *ActivityTabPane {
	return &ActivityTabPane{
		TabPane:       widgets.NewTabPane(" 0 "),
	}
}

func (self *ActivityTabPane) Draw(buf *ui.Buffer) {
	self.Block.Draw(buf)

	xCoordinate := self.Inner.Min.X
	for i, name := range self.TabNames {
		ColorPair := self.InactiveTabStyle
		if k, ok := self.TabsWithActivity[i]; ok {
			switch k {
			case TabHasActivity:
				ColorPair = self.ActivityStyle
			case TabHasNotice:
				ColorPair = self.NoticeStyle
			}
		}
		if i == self.ActiveTabIndex {
			ColorPair = self.ActiveTabStyle
		}
		buf.SetString(
			ui.TrimString(name, self.Inner.Max.X-xCoordinate),
			ColorPair,
			image.Pt(xCoordinate, self.Inner.Min.Y),
		)

		xCoordinate += 1 + len(name)

		if i < len(self.TabNames)-1 && xCoordinate < self.Inner.Max.X {
			buf.SetCell(
				ui.NewCell(ui.VERTICAL_LINE, ui.NewStyle(ui.ColorWhite)),
				image.Pt(xCoordinate, self.Inner.Min.Y),
			)
		}

		xCoordinate += 2
	}
}
