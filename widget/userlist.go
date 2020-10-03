package widget

import (
	"image"

	ui "github.com/gizak/termui/v3"
)

// A UserList contains a list of users on a channel.
// This widget is based on the termui Table widget.
type UserList struct {
	ui.Block
	Rows      []string
	TextStyle ui.Style
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

	// draw rows
	for i := 0; i < len(ul.Rows) && yCoordinate < ul.Inner.Max.Y; i++ {
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
}
