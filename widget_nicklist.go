package squirssi

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

func (self *UserList) Draw(buf *ui.Buffer) {
	self.Block.Draw(buf)

	columnWidth := self.Inner.Dx()

	yCoordinate := self.Inner.Min.Y

	// draw rows
	for i := 0; i < len(self.Rows) && yCoordinate < self.Inner.Max.Y; i++ {
		row := self.Rows[i]
		colXCoordinate := self.Inner.Min.X

		rowStyle := self.TextStyle

		col := ui.ParseStyles(row, rowStyle)
		// draw row cell
		if len(col) > columnWidth {
			for _, cx := range ui.BuildCellWithXArray(col) {
				k, cell := cx.X, cx.Cell
				if k == columnWidth || colXCoordinate+k == self.Inner.Max.X {
					cell.Rune = ui.ELLIPSES
					buf.SetCell(cell, image.Pt(colXCoordinate+k-1, yCoordinate))
					break
				} else {
					buf.SetCell(cell, image.Pt(colXCoordinate+k, yCoordinate))
				}
			}
		} else {
			stringXCoordinate := ui.MinInt(colXCoordinate+columnWidth, self.Inner.Max.X) - len(col)
			for _, cx := range ui.BuildCellWithXArray(col) {
				k, cell := cx.X, cx.Cell
				buf.SetCell(cell, image.Pt(stringXCoordinate+k, yCoordinate))
			}
		}

		yCoordinate++
	}
}
