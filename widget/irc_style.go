package widget

import (
	"strconv"
	"unicode"

	ui "github.com/gizak/termui/v3"

	"code.dopame.me/veonik/squirssi/colors"
)

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
			for j := i + 1; j-i < 5 && j < len(c); j++ {
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
			if fg != "" {
				p, _ := strconv.Atoi(fg)
				if p > 15 {
					// fg is too big, only use the first number
					fg = fg[0:1]
					eat -= len(bg) + 1
					bg = ""
				}
			}
			if bg != "" {
				p, _ := strconv.Atoi(bg)
				if p > 15 {
					// bg is too big, only use the first number
					bg = bg[0:1]
					eat -= 1
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
