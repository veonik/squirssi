package squirssi

import (
	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/plugin"
	"github.com/pkg/errors"
)

const pluginName = "squirssi"

func FromPlugins(m *plugin.Manager) (*Server, error) {
	plg, err := m.Lookup(pluginName)
	if err != nil {
		return nil, err
	}
	mplg, ok := plg.(*squirssiPlugin)
	if !ok {
		return nil, errors.Errorf("event: received unexpected plugin type")
	}
	return mplg.server, nil
}

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	ev, err := event.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: missing required dependency (event)", pluginName)
	}
	srv, err := NewServer(ev)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: failed to initialize Server", pluginName)
	}
	p := &squirssiPlugin{srv}
	return p, nil
}

type squirssiPlugin struct {
	server *Server
}

func (p *squirssiPlugin) Name() string {
	return "squirssi"
}
