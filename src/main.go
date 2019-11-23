/*
Joe Bot

2019 © Postgres.ai

Conversational UI bot for Postgres query optimization.
*/

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"./log"

	"./srv"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"
)

var opts struct {
	VerificationToken string `short:"v" long:"verification-token" description:"callback URL verification token" env:"VERIFICATION_TOKEN" required:"true"`
	DbPassword        string `short:"P" long:"password" description:"database password" env:"DB_PASSWORD" default:"postgres"`

	ShowHelp func() error `long:"help" description:"Show this help message"`
}

func main() {
	// Load CLI options.
	var _, err = parseArgs()

	if err != nil {
		if flags.WroteHelp(err) {
			return
		}

		log.Err("Args parse error", err)
		return
	}

	log.DEBUG = true

	srv.RunServer()
}

func parseArgs() ([]string, error) {
	var parser = flags.NewParser(&opts, flags.Default & ^flags.HelpFlag)

	// jessevdk/go-flags lib doesn't allow to use short flag -h because it's binded to usage help.
	// We need to hack it a bit to use -h for as a hostname option. See https://github.com/jessevdk/go-flags/issues/240
	opts.ShowHelp = func() error {
		var b bytes.Buffer

		parser.WriteHelp(&b)
		return &flags.Error{
			Type:    flags.ErrHelp,
			Message: b.String(),
		}
	}

	return parser.Parse()
}

func loadConfig(config interface{}, name string) error {
	b, err := ioutil.ReadFile(getConfigPath(name))
	if err != nil {
		return fmt.Errorf("Error loading %s config file", name)
	}

	err = yaml.Unmarshal(b, config)
	if err != nil {
		return fmt.Errorf("Error parsing %s config", name)
	}

	log.Dbg("Config loaded", name, config)
	return nil
}

func getConfigPath(name string) string {
	bindir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir, _ := filepath.Abs(filepath.Dir(bindir))
	path := dir + string(os.PathSeparator) + "config" + string(os.PathSeparator) + name
	return path
}
