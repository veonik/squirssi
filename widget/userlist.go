package widget

import (
	"image"

	ui "github.com/gizak/termui/v3"

	"code.dopame.me/veonik/squirssi/colors"
)

// A UserList contains a list of users on a channel.
// This widget is based on the termui Table widget.
type UserList struct {
	ui.Block
	Rows        []string
	TextStyle   ui.Style
	SelectedRow int
}

func NewUserList() *UserList {
	return &UserList{
		Block:     *ui.NewBlock(),
		TextStyle: ui.Theme.Table.Text,
	}
}

func (ul *UserList) Draw(buf *ui.Buffer) {
	ul.Block.Draw(buf)

	columnWidth := ul.Inner.Dx()

	yCoordinate := ul.Inner.Min.Y

	if ul.SelectedRow < 0 {
		ul.SelectedRow = 0
	} else if ul.SelectedRow >= len(ul.Rows) {
		ul.SelectedRow = len(ul.Rows) - 1
	}

	topRow := 0

	// adjust starting row based on the bounding box and the actual selected row
	if ul.SelectedRow >= ul.Inner.Dy()+topRow {
		topRow = ul.SelectedRow - ul.Inner.Dy() + 1
	} else if ul.SelectedRow < topRow {
		topRow = ul.SelectedRow
	}

	if topRow < 0 {
		topRow = 0
	}

	// draw rows
	for i := topRow; i < len(ul.Rows) && yCoordinate < ul.Inner.Max.Y; i++ {
		row := ul.Rows[i]
		colXCoordinate := ul.Inner.Min.X

		rowStyle := ul.TextStyle

		col := ui.ParseStyles(row, rowStyle)
		// draw row cell
		if len(col) > columnWidth {
			for _, cx := range ui.BuildCellWithXArray(col) {
				k, cell := cx.X, cx.Cell
				if k == columnWidth || colXCoordinate+k == ul.Inner.Max.X {
					cell.Rune = ui.ELLIPSES
					buf.SetCell(cell, image.Pt(colXCoordinate+k-1, yCoordinate))
					break
				} else {
					buf.SetCell(cell, image.Pt(colXCoordinate+k, yCoordinate))
				}
			}
		} else {
			stringXCoordinate := ui.MinInt(colXCoordinate+columnWidth, ul.Inner.Max.X) - len(col)
			for _, cx := range ui.BuildCellWithXArray(col) {
				k, cell := cx.X, cx.Cell
				buf.SetCell(cell, image.Pt(stringXCoordinate+k, yCoordinate))
			}
		}

		yCoordinate++
	}

	// draw UP_ARROW if needed
	if topRow > 0 {
		buf.SetCell(
			ui.NewCell(ui.UP_ARROW, ui.NewStyle(colors.Grey42)),
			image.Pt(ul.Inner.Min.X+1, ul.Inner.Min.Y),
		)
	}

	// draw DOWN_ARROW if needed
	if len(ul.Rows) > topRow+ul.Inner.Dy() {
		buf.SetCell(
			ui.NewCell(ui.DOWN_ARROW, ui.NewStyle(colors.Grey42)),
			image.Pt(ul.Inner.Min.X+1, ul.Inner.Max.Y-1),
		)
	}
}
