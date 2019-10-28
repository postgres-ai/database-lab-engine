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

	"./bot"
	"./chatapi"
	"./ec2ctrl"
	"./log"
	"./pgexplain"
	"./provision"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"
)

var opts struct {
	// Chat API.
	AccessToken       string `short:"t" long:"token" description:"\"Bot User OAuth Access Token\" which starts with \"xoxb-\"" env:"CHAT_TOKEN" required:"true"`
	VerificationToken string `short:"v" long:"verification-token" description:"callback URL verification token" env:"CHAT_VERIFICATION_TOKEN" required:"true"`

	// Database.
	DbHost     string `short:"h" long:"host" description:"database server host" env:"DB_HOST" default:"localhost"`
	DbPort     uint   `short:"p" long:"port" description:"database server port" env:"DB_PORT" default:"5432"`
	DbUser     string `short:"U" long:"username" description:"database user name" env:"DB_USER" default:"postgres"`
	DbPassword string `short:"P" long:"password" description:"database password" env:"DB_PASSWORD" default:"postgres"`
	DbName     string `short:"d" long:"dbname" description:"database name to connect to" env:"DB_NAME" default:"db"`

	// HTTP Server.
	ServerPort uint `short:"s" long:"http-port" description:"HTTP server port" env:"SERVER_PORT" default:"3000"`

	QuotaLimit    uint `long:"quota-limit" description:"limit request rates to up to 2x of this number" env:"QUOTA_LIMIT" default:"10"`
	QuotaInterval uint `long:"quota-interval" description:"an time interval (in seconds) to apply a quota-limit" env:"QUOTA_INTERVAL" default:"60"`

	IdleInterval uint `long:"idle-interval" description:"an time interval (in seconds) before user session can be stoped due to idle" env:"IDLE_INTERVAL" default:"3600"`

	// Platform.
	ApiUrl         string `long:"api-url" description:"Postgres.ai platform API base URL" env:"API_URL" default:"https://postgres.ai/api/general"`
	ApiToken       string `long:"api-token" description:"Postgres.ai platform API token" env:"API_TOKEN"`
	ApiProject     string `long:"api-project" description:"Postgres.ai platform project to assign user sessions" env:"API_PROJECT"`
	HistoryEnabled bool   `long:"history-enabled" description:"send command and queries history to Postgres.ai platform for collaboration and visualization" env:"HISTORY_ENABLED"`

	// Dev.
	DevGitCommitHash string `long:"git-commit-hash" env:"GIT_COMMIT_HASH" default:""`
	DevGitBranch     string `long:"git-branch" env:"GIT_BRANCH" default:""`
	DevGitModified   bool   `long:"git-modified" env:"GIT_MODIFIED"`

	ShowHelp func() error `long:"help" description:"Show this help message"`
}

// TODO(anatoly): Refactor configs and envs.
type ProvisionConfig struct {
	Mode    string                  `yaml:"mode"`
	Aws     provision.AwsConfig     `yaml:"aws"`
	Local   provision.LocalConfig   `yaml:"local"`
	MuLocal provision.MuLocalConfig `yaml:"mulocal"`
	Debug   bool                    `yaml:"debug"`

	ZfsPool         string `yaml:"zfsPool"`
	InitialSnapshot string `yaml:"initialSnapshot"`

	PgVersion    string `yaml:"pgVersion"`
	PgBindir     string `yaml:"pgBindir"`
	PgDataSubdir string `yaml:"pgDataSubdir"`
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

	// Load and validate configuration files.
	explainConfig, err := loadExplainConfig()
	if err != nil {
		log.Err("Unable to load explain config", err)
		return
	}
	provisionConfig, err := loadProvisionConfig()
	if err != nil {
		log.Err("Unable to load provision config", err)
		return
	}
	log.DEBUG = provisionConfig.Debug
	provConf := provision.Config{
		Mode:    provisionConfig.Mode,
		Aws:     provisionConfig.Aws,
		Local:   provisionConfig.Local,
		MuLocal: provisionConfig.MuLocal,
		Debug:   provisionConfig.Debug,

		// ZFS.
		ZfsPool:         provisionConfig.ZfsPool,
		InitialSnapshot: provisionConfig.InitialSnapshot,

		// TODO(anatoly): Use opts.DbPort, opts.DbHost for local and direct mode.
		DbHost:     opts.DbHost,
		DbUsername: opts.DbUser,
		DbPassword: opts.DbPassword,
		DbName:     opts.DbName,

		PgVersion:    provisionConfig.PgVersion,
		PgBindir:     provisionConfig.PgBindir,
		PgDataSubdir: provisionConfig.PgDataSubdir,
	}
	if !provision.IsValidConfig(provConf) {
		log.Err("Wrong configuration format.")
		os.Exit(1)
	}

	// Initialize provisioning.
	prov, err := provision.NewProvision(provConf)
	if err != nil {
		log.Fatal("Provision constuct failed", err)
	}

	err = prov.Init()
	if err != nil {
		log.Fatal("Provision init error", err)
	}
	log.Dbg("Provision init ok", err)

	log.Dbg("git: ", opts.DevGitCommitHash, opts.DevGitBranch, opts.DevGitModified)

	version := formatBotVersion(opts.DevGitCommitHash, opts.DevGitBranch,
		opts.DevGitModified)

	config := bot.Config{
		Port:          opts.ServerPort,
		Explain:       explainConfig,
		QuotaLimit:    opts.QuotaLimit,
		QuotaInterval: opts.QuotaInterval,
		IdleInterval:  opts.IdleInterval,

		DbName: opts.DbName,

		ApiUrl:         opts.ApiUrl,
		ApiToken:       opts.ApiToken,
		ApiProject:     opts.ApiProject,
		HistoryEnabled: opts.HistoryEnabled,

		Version: version,
	}

	var chat = chatapi.NewChat(opts.AccessToken, opts.VerificationToken)

	joeBot := bot.NewBot(config, chat, prov)
	joeBot.RunServer()
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

func loadExplainConfig() (pgexplain.ExplainConfig, error) {
	var config pgexplain.ExplainConfig

	err := loadConfig(&config, "explain.yaml")
	if err != nil {
		return config, err
	}

	return config, nil
}

func loadProvisionConfig() (ProvisionConfig, error) {
	var config = ProvisionConfig{
		Aws: provision.AwsConfig{
			Ec2: ec2ctrl.Ec2Configuration{
				AwsInstanceType: "r4.large",
				AwsRegion:       "us-east-1",
				AwsZone:         "a",
			},
		},
		Mode:            "aws",
		Debug:           true,
		PgVersion:       "9.6",
		ZfsPool:         "zpool",
		InitialSnapshot: "db_state_1",
	}

	err := loadConfig(&config, "provisioning.yaml")
	if err != nil {
		return config, err
	}

	return config, nil
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

func formatBotVersion(commit string, branch string, modified bool) string {
	if len(commit) < 7 {
		return "unknown"
	}

	modifiedStr := ""
	if modified {
		modifiedStr = " (modified)"
	}

	commitShort := commit[:7]

	return fmt.Sprintf("%s@%s%s", commitShort, branch, modifiedStr)
}
