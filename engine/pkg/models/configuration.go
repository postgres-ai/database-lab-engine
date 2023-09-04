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
	Debug                *bool                  `proj:"global.debug"`
	DatabaseConfigs      map[string]interface{} `proj:"databaseConfigs.configs"`
	DockerImage          *string                `proj:"databaseContainer.dockerImage"`
	Timetable            *string                `proj:"retrieval.refresh.timetable"`
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
}
