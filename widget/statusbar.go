package widget

import (
	"image"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type ActivityType int

const TabHasActivity ActivityType = 0
const TabHasNotice ActivityType = 1

// StatusBarPane contains the tabs for available windows.
// This widget compounds a termui TabPane widget with highlighting of tabs
// in two additional ways: notice and activity. Notice is intended for when
// a tab wants extra attention (ie. user was mentioned) vs activity where
// there are just some new lines since last touched.
type StatusBarPane struct {
	*widgets.TabPane

	TabsWithActivity map[int]ActivityType
	NoticeStyle      ui.Style
	ActivityStyle    ui.Style
}

func NewStatusBarPane() *StatusBarPane {
	return &StatusBarPane{
		TabPane: widgets.NewTabPane(" 0 "),
	}
}

func (sb *StatusBarPane) Draw(buf *ui.Buffer) {
	sb.Block.Draw(buf)

	xCoordinate := sb.Inner.Min.X
	for i, name := range sb.TabNames {
		ColorPair := sb.InactiveTabStyle
		if k, ok := sb.TabsWithActivity[i]; ok {
			switch k {
			case TabHasActivity:
				ColorPair = sb.ActivityStyle
			case TabHasNotice:
				ColorPair = sb.NoticeStyle
			}
		}
		if i == sb.ActiveTabIndex {
			ColorPair = sb.ActiveTabStyle
		}
		buf.SetString(
			ui.TrimString(name, sb.Inner.Max.X-xCoordinate),
			ColorPair,
			image.Pt(xCoordinate, sb.Inner.Min.Y),
		)

		xCoordinate += 1 + len(name)

		if i < len(sb.TabNames)-1 && xCoordinate < sb.Inner.Max.X {
			buf.SetCell(
				ui.NewCell(ui.VERTICAL_LINE, ui.NewStyle(ui.ColorWhite)),
				image.Pt(xCoordinate, sb.Inner.Min.Y),
			)
		}

		xCoordinate += 2
	}
}
