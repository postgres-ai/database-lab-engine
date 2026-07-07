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
//
// The `RetrievalMode` field has no `proj:` tag because it does not map to a YAML
// key — it is derived from which job types are present in `retrieval.jobs` and
// `retrieval.spec`. The projectedAdminConfig handler populates it on the response
// map after StoreJSON returns, and the request-side gate reads it directly from
// the incoming JSON. See `engine/internal/srv/config.go`.
type ConfigProjection struct {
	RetrievalMode            RetrievalMode          `json:"retrievalMode,omitempty"`
	Debug                    *bool                  `proj:"global.debug"`
	DatabaseConfigs          map[string]interface{} `proj:"databaseConfigs.configs"`
	DockerImage              *string                `proj:"databaseContainer.dockerImage"`
	Timetable                *string                `proj:"retrieval.refresh.timetable"`
	ConnectionString         *string                `proj:"retrieval.spec.logicalDump.options.source.connectionString,createKey" groups:"sensitive"` //nolint:lll
	DBName                   *string                `proj:"retrieval.spec.logicalDump.options.source.connection.dbname"`
	Host                     *string                `proj:"retrieval.spec.logicalDump.options.source.connection.host"`
	Password                 *string                `proj:"retrieval.spec.logicalDump.options.source.connection.password" groups:"sensitive"`
	Port                     *int64                 `proj:"retrieval.spec.logicalDump.options.source.connection.port"`
	Username                 *string                `proj:"retrieval.spec.logicalDump.options.source.connection.username"`
	DBList                   map[string]interface{} `proj:"retrieval.spec.logicalDump.options.databases,createKey"`
	DumpParallelJobs         *int64                 `proj:"retrieval.spec.logicalDump.options.parallelJobs"`
	RestoreParallelJobs      *int64                 `proj:"retrieval.spec.logicalRestore.options.parallelJobs"`
	RestoreConfigs           map[string]interface{} `proj:"retrieval.spec.logicalRestore.options.configs,createKey"`
	DumpCustomOptions        []interface{}          `proj:"retrieval.spec.logicalDump.options.customOptions"`
	RestoreCustomOptions     []interface{}          `proj:"retrieval.spec.logicalRestore.options.customOptions"`
	IgnoreDumpErrors         *bool                  `proj:"retrieval.spec.logicalDump.options.ignoreErrors"`
	IgnoreRestoreErrors      *bool                  `proj:"retrieval.spec.logicalRestore.options.ignoreErrors"`
	RDSIAMDBInstance         *string                `proj:"retrieval.spec.logicalDump.options.source.rdsIam.dbInstanceIdentifier"`
	PhysicalTool             *string                `proj:"retrieval.spec.physicalRestore.options.tool"`
	PhysicalDockerImage      *string                `proj:"retrieval.spec.physicalRestore.options.dockerImage"`
	PhysicalSyncEnabled      *bool                  `proj:"retrieval.spec.physicalRestore.options.sync.enabled"`
	PhysicalWalgBackupName   *string                `proj:"retrieval.spec.physicalRestore.options.walg.backupName"`
	PhysicalPgbackrestStanza *string                `proj:"retrieval.spec.physicalRestore.options.pgbackrest.stanza"`
	PhysicalPgbackrestDelta  *bool                  `proj:"retrieval.spec.physicalRestore.options.pgbackrest.delta"`
	PhysicalEnvs             map[string]interface{} `proj:"retrieval.spec.physicalRestore.options.envs,createKey"`
}
