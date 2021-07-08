// +build !linked_plugins

package main

import (
	"code.dopame.me/veonik/squircy3/plugin"

	"code.dopame.me/veonik/squirssi"
)

var linkedPlugins = []plugin.Initializer{
	plugin.InitializerFunc(squirssi.Initialize)}
