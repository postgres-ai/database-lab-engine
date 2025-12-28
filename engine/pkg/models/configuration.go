package models

// ConnectionTest defines a connection test model.
type ConnectionTest struct {
	Host     string   `json:"host"`
	Port     string   `json:"port"`
	DBName   string   `json:"dbname"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	DBList   []string `json:"db_list"`
}

// ConfigProjection is a projection of the configuration.
type ConfigProjection struct {
	// Global settings
	Debug            *bool   `proj:"global.debug"`
	GlobalDBUsername *string `proj:"global.database.username"`
	GlobalDBName     *string `proj:"global.database.dbname"`

	// Server settings
	ServerHost *string `proj:"server.host"`
	ServerPort *uint   `proj:"server.port" restart:"true"`

	// Provision settings
	PortPoolFrom         *uint             `proj:"provision.portPool.from" restart:"true"`
	PortPoolTo           *uint             `proj:"provision.portPool.to" restart:"true"`
	UseSudo              *bool             `proj:"provision.useSudo"`
	KeepUserPasswords    *bool             `proj:"provision.keepUserPasswords"`
	CloneAccessAddresses *string           `proj:"provision.cloneAccessAddresses"`
	ContainerConfig      map[string]string `proj:"provision.containerConfig"`

	// Database container settings
	DockerImage     *string                `proj:"databaseContainer.dockerImage"`
	DatabaseConfigs map[string]interface{} `proj:"databaseConfigs.configs"`

	// Cloning settings
	MaxIdleMinutes *uint   `proj:"cloning.maxIdleMinutes"`
	AccessHost     *string `proj:"cloning.accessHost"`

	// Retrieval settings
	Timetable            *string                `proj:"retrieval.refresh.timetable"`
	SkipStartRefresh     *bool                  `proj:"retrieval.refresh.skipStartRefresh"`
	DBName               *string                `proj:"retrieval.spec.logicalDump.options.source.connection.dbname"`
	Host                 *string                `proj:"retrieval.spec.logicalDump.options.source.connection.host"`
	Password             *string                `proj:"retrieval.spec.logicalDump.options.source.connection.password" groups:"sensitive"`
	Port                 *int64                 `proj:"retrieval.spec.logicalDump.options.source.connection.port"`
	Username             *string                `proj:"retrieval.spec.logicalDump.options.source.connection.username"`
	DBList               map[string]interface{} `proj:"retrieval.spec.logicalDump.options.databases,createKey"`
	DumpParallelJobs     *int64                 `proj:"retrieval.spec.logicalDump.options.parallelJobs"`
	RestoreParallelJobs  *int64                 `proj:"retrieval.spec.logicalRestore.options.parallelJobs"`
	DumpCustomOptions    []interface{}          `proj:"retrieval.spec.logicalDump.options.customOptions"`
	RestoreCustomOptions []interface{}          `proj:"retrieval.spec.logicalRestore.options.customOptions"`
	IgnoreDumpErrors     *bool                  `proj:"retrieval.spec.logicalDump.options.ignoreErrors"`
	IgnoreRestoreErrors  *bool                  `proj:"retrieval.spec.logicalRestore.options.ignoreErrors"`

	// Pool Manager settings
	PoolMountDir     *string `proj:"poolManager.mountDir" restart:"true"`
	PoolSelectedPool *string `proj:"poolManager.selectedPool"`

	// Observer settings
	ObserverReplacementRules map[string]string `proj:"observer.replacementRules"`

	// Diagnostic settings
	LogsRetentionDays *int `proj:"diagnostic.logsRetentionDays"`

	// EmbeddedUI settings
	EmbeddedUIEnabled     *bool   `proj:"embeddedUI.enabled" restart:"true"`
	EmbeddedUIDockerImage *string `proj:"embeddedUI.dockerImage"`
	EmbeddedUIHost        *string `proj:"embeddedUI.host"`
	EmbeddedUIPort        *int    `proj:"embeddedUI.port" restart:"true"`

	// Platform settings
	PlatformURL                 *string `proj:"platform.url"`
	PlatformOrgKey              *string `proj:"platform.orgKey" groups:"sensitive"`
	PlatformProjectName         *string `proj:"platform.projectName"`
	PlatformEnablePersonalToken *bool   `proj:"platform.enablePersonalTokens"`
	PlatformEnableTelemetry     *bool   `proj:"platform.enableTelemetry"`

	// Webhooks settings
	WebhooksHooks []WebhookHookProjection `proj:"webhooks.hooks"`
}

// WebhookHookProjection is a projection for webhook hook configuration.
type WebhookHookProjection struct {
	URL     string   `proj:"url" json:"url"`
	Secret  string   `proj:"secret" json:"secret" groups:"sensitive"`
	Trigger []string `proj:"trigger" json:"trigger"`
}

// ConfigUpdateResponse represents the response from a config update with warnings.
type ConfigUpdateResponse struct {
	Config           interface{}     `json:"config"`
	Warnings         []ConfigWarning `json:"warnings,omitempty"`
	RequiresRestart  bool            `json:"requiresRestart"`
	ChangedSettings  []string        `json:"changedSettings,omitempty"`
	RestartSettings  []string        `json:"restartSettings,omitempty"`
}

// ConfigWarning represents a warning message for configuration changes.
type ConfigWarning struct {
	Setting string `json:"setting"`
	Message string `json:"message"`
	Type    string `json:"type"` // "restart", "security", "info"
}

// RestartRequiredSettings lists settings that require a restart when changed.
var RestartRequiredSettings = map[string]string{
	"server.port":           "Changing the server port requires a restart to take effect",
	"provision.portPool.from":   "Changing the port pool requires a restart to take effect",
	"provision.portPool.to":     "Changing the port pool requires a restart to take effect",
	"poolManager.mountDir":      "Changing the mount directory requires a restart to take effect",
	"embeddedUI.enabled":        "Enabling or disabling the embedded UI requires a restart to take effect",
	"embeddedUI.port":           "Changing the embedded UI port requires a restart to take effect",
}
