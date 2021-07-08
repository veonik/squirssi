package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	_ "gopkg.in/mattes/go-expand-tilde.v1"

	"code.dopame.me/veonik/squircy3/cli"
	"code.dopame.me/veonik/squircy3/irc"
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squirssi"
)

var SquirssiVersion = "SNAPSHOT"
var Squircy3Version = "SNAPSHOT"

type Manager struct {
	*cli.Manager
}

func (m *Manager) Start() (err error) {
	if err := m.Manager.Start(); err != nil {
		return err
	}
	logrus.Infof("Starting squirssi (version %s, built with squircy3-%s)", SquirssiVersion, Squircy3Version)
	plugins := m.Plugins()
	printPluginsLoaded(plugins)
	ircp, err := irc.FromPlugins(plugins)
	if err != nil {
		return err
	}
	ircp.SetVersionString(fmt.Sprintf("squirssi %s", SquirssiVersion))
	srv, err := squirssi.FromPlugins(plugins)
	if err != nil {
		return err
	}
	srv.OnInterrupt(m.Stop)
	return srv.Start()
}

func NewManager() (*Manager, error) {
	cm, err := cli.NewManager()
	if err != nil {
		return nil, err
	}
	cm.LinkedPlugins = append(cm.LinkedPlugins, linkedPlugins...)
	return &Manager{cm}, nil
}

func init() {
	printVersion := false
	cli.CoreFlags(flag.CommandLine, "~/.squirssi")
	cli.IRCFlags(flag.CommandLine)
	cli.VMFlags(flag.CommandLine)
	flag.BoolVar(&printVersion, "version", false, "print version information")

	flag.Usage = func() {
		fmt.Println("Usage: ", os.Args[0], "[options]")
		fmt.Println()
		fmt.Println("squirssi is a proper IRC client.")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if printVersion {
		fmt.Printf("squirssi (version %s, built with squircy3-%s)\n", SquirssiVersion, Squircy3Version)
		os.Exit(0)
	}
}

func printPluginsLoaded(plugins *plugin.Manager) {
	pls := plugins.Loaded()
	sort.Strings(pls)
	logrus.Infoln("Loaded plugins:", strings.Join(pls, ", "))
}

func unboxAll(rootDir string) (modified bool, err error) {
	if _, err = os.Stat(filepath.Join(rootDir, "config.toml")); err == nil {
		// root directory already exists, don't muck with it
		return false, nil
	}
	if err = os.MkdirAll(rootDir, 0755); err != nil {
		return false, errors.Wrap(err, "failed to create root directory")
	}
	box := packr.New("defconf", "./defconf")
	for _, f := range box.List() {
		dst := filepath.Join(rootDir, f)
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return true, errors.Wrap(err, "failed to recreate directory")
			}
			logrus.Infof("Creating default %s", dst)
			d, err := box.Find(f)
			if err != nil {
				return true, errors.Wrapf(err, "failed to get contents of boxed %s", f)
			}
			if err := ioutil.WriteFile(dst, d, 0644); err != nil {
				return true, errors.Wrapf(err, "failed to write unboxed file %s", f)
			}
		}
	}
	return true, nil
}

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	m, err := NewManager()
	if err != nil {
		logrus.Fatalln("core: error initializing squirssi:", err)
	}
	logrus.SetLevel(m.LogLevel)
	if modified, err := unboxAll(m.RootDir); err != nil {
		logrus.Fatalln("core: failed to unbox defaults:", err)
	} else if modified {
		// re-initialize the manager so that plugins load
		m, err = NewManager()
		if err != nil {
			logrus.Fatalln("core: error initializing squirssi:", err)
		}
		logrus.SetLevel(m.LogLevel)
	}
	if err := m.Start(); err != nil {
		logrus.Fatalln("core: error starting squirssi:", err)
	}
	if err = m.Loop(); err != nil {
		logrus.Fatalln("core: exiting main loop with error:", err)
	}
}
