package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"code.dopame.me/veonik/squircy3/cli"
	"code.dopame.me/veonik/squircy3/plugin"
	"github.com/sirupsen/logrus"
	tilde "gopkg.in/mattes/go-expand-tilde.v1"

	"code.dopame.me/veonik/squirssi"
)

type stringsFlag []string

func (s stringsFlag) String() string {
	return strings.Join(s, "")
}
func (s *stringsFlag) Set(str string) error {
	*s = append(*s, str)
	return nil
}

type stringLevel logrus.Level

func (s stringLevel) String() string {
	return logrus.Level(s).String()
}
func (s *stringLevel) Set(str string) error {
	l, err := logrus.ParseLevel(str)
	if err != nil {
		return err
	}
	*s = stringLevel(l)
	return nil
}

type pluginOptsFlag map[string]interface{}

func (s pluginOptsFlag) String() string {
	var res []string
	for k, v := range s {
		res = append(res, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(res, " ")
}

func (s pluginOptsFlag) Set(str string) error {
	p := strings.SplitN(str, "=", 2)
	if len(p) == 1 {
		p = append(p, "true")
	}
	var v interface{}
	if p[1] == "true" {
		v = true
	} else if p[1] == "false" {
		v = false
	} else {
		v = p[1]
	}
	s[p[0]] = v
	return nil
}

var rootDir string
var extraPlugins stringsFlag
var pluginOptions pluginOptsFlag
var logLevel = stringLevel(logrus.InfoLevel)

var Squircy3Version = "SNAPSHOT"

func init() {
	flag.StringVar(&rootDir, "root", "~/.squirssi", "path to folder containing squirssi data")
	flag.Var(&logLevel, "log-level", "controls verbosity of logging output")
	flag.Var(&extraPlugins, "plugin", "path to shared plugin .so file, multiple plugins may be given")
	flag.Var(&pluginOptions, "plugin-option", "specify extra plugin configuration option, format: key=value")

	flag.Usage = func() {
		fmt.Println("Usage: ", os.Args[0], "[options]")
		fmt.Println()
		fmt.Println("squirssi is a proper IRC client.")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
	}
	flag.Parse()
	bp, err := tilde.Expand(rootDir)
	if err != nil {
		logrus.Fatalln(err)
	}
	err = os.MkdirAll(bp, 0644)
	if err != nil {
		logrus.Fatalln(err)
	}
	rootDir = bp
}

func printPluginsLoaded(plugins *plugin.Manager) {
	pls := plugins.Loaded()
	sort.Strings(pls)
	logrus.Infoln("Loaded plugins:", strings.Join(pls, ", "))
}

type Manager struct {
	*cli.Manager
}

func (m *Manager) Start() (err error) {
	if err := m.Manager.Start(); err != nil {
		return err
	}
	logrus.Infof("Starting squirssi (version %s, built with squircy3-%s)", squirssi.Version, Squircy3Version)
	plugins := m.Plugins()
	printPluginsLoaded(plugins)
	srv, err := squirssi.FromPlugins(plugins)
	if err != nil {
		return err
	}
	srv.OnInterrupt(m.Stop)
	return srv.Start()
}

func NewManager(rootDir string, extraPlugins ...string) (*Manager, error) {
	cm, err := cli.NewManager(rootDir, pluginOptions, extraPlugins...)
	if err != nil {
		return nil, err
	}
	cm.LinkedPlugins = append(cm.LinkedPlugins, plugin.InitializerFunc(squirssi.Initialize))
	return &Manager{cm}, nil
}

func main() {
	logrus.SetLevel(logrus.Level(logLevel))
	m, err := NewManager(rootDir, extraPlugins...)
	if err != nil {
		logrus.Fatalln("error initializing squirssi:", err)
	}
	if err := m.Start(); err != nil {
		logrus.Fatalln("error starting squirssi:", err)
	}
	if err = m.Loop(); err != nil {
		logrus.Fatalln("exiting main loop with error:", err)
	}
}
