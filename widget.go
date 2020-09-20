package squirssi

import (
	"image"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type ActivityTabPane struct {
	*widgets.TabPane

	TabsWithActivity map[int]struct{}
	ActivityStyle    ui.Style
}

func (self *ActivityTabPane) Draw(buf *ui.Buffer) {
	self.Block.Draw(buf)

	xCoordinate := self.Inner.Min.X
	for i, name := range self.TabNames {
		ColorPair := self.InactiveTabStyle
		if _, ok := self.TabsWithActivity[i]; ok {
			ColorPair = self.ActivityStyle
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
