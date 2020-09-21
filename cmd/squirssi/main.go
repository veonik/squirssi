package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"code.dopame.me/veonik/squircy3/cli"
	"code.dopame.me/veonik/squircy3/plugin"
	"github.com/pkg/errors"
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

var rootDir string
var extraPlugins stringsFlag
var logLevel = stringLevel(logrus.DebugLevel)

func init() {
	flag.StringVar(&rootDir, "root", "~/.squirssi", "path to folder containing squirssi data")
	flag.Var(&logLevel, "log-level", "controls verbosity of logging output")
	flag.Var(&extraPlugins, "plugin", "path to shared plugin .so file, multiple plugins may be given")

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

func main() {
	logrus.SetLevel(logrus.Level(logLevel))
	m, err := cli.NewManager(rootDir, extraPlugins...)
	if err != nil {
		logrus.Fatalln("error initializing squirssi:", err)
	}
	if err := m.Start(); err != nil {
		logrus.Fatalln("error starting squirssi:", err)
	}
	plugins := m.Plugins()
	plugins.RegisterFunc(squirssi.Initialize)
	if err := configure(plugins); err != nil {
		logrus.Fatalln("error starting squirssi:", err)
	}
	wait := make(chan struct{})
	go func() {
		defer close(wait)
		if err = m.Loop(); err != nil {
			logrus.Fatalln("exiting main loop with error:", err)
		}
	}()
	srv, err := squirssi.FromPlugins(plugins)
	if err != nil {
		logrus.Fatalln("error starting squirssi:", err)
	}
	defer srv.Close()
	srv.Start()
}

func configure(m *plugin.Manager) error {
	errs := m.Configure()
	if errs != nil && len(errs) > 0 {
		if len(errs) > 1 {
			return errors.WithMessage(errs[0], fmt.Sprintf("(and %d more...)", len(errs)-1))
		}
		return errs[0]
	}
	return nil
}
