package colors

import (
	ui "github.com/gizak/termui/v3"
)

type IRC int

const (
	IRCWhite IRC = iota
	IRCBlack
	IRCBlue
	IRCGreen
	IRCRed
	IRCBrown
	IRCMagenta
	IRCOrange
	IRCYellow
	IRCLightGreen
	IRCCyan
	IRCLightCyan
	IRCLightBlue
	IRCPink
	IRCGrey
	IRCLightGrey

	IRCGray      = IRCGrey
	IRCLightGray = IRCLightGrey
	IRCDefault   = 99
)

var ircToUIMap = map[IRC]ui.Color{
	IRCWhite:      White,
	IRCBlack:      Black,
	IRCBlue:       Blue,
	IRCGreen:      Green,
	IRCRed:        Red,
	IRCBrown:      Red4,
	IRCMagenta:    Magenta1,
	IRCOrange:     Orange1,
	IRCYellow:     Yellow,
	IRCLightGreen: LightGreen,
	IRCCyan:       Cyan1,
	IRCLightCyan:  LightCyan1,
	IRCLightBlue:  LightSkyBlue1,
	IRCPink:       Pink1,
	IRCGrey:       Grey,
	IRCLightGrey:  Grey30,
	IRCDefault:    Clear,
}

func IRCToUI(irc IRC) ui.Color {
	if v, ok := ircToUIMap[irc]; ok {
		return v
	}
	return IRCDefault
}
