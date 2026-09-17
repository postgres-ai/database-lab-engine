package yaml

// DefaultConfigMask returns the mask applied to the config before it is shown. A path that names
// a mapping, such as the physical restore envs, keeps its keys and masks every value: WAL-G and
// pgBackRest read their credentials from there.
func DefaultConfigMask() *Mask {
	sensitive := []string{
		"server.verificationToken",
		"platform.accessToken",
		"retrieval.spec.logicalDump.options.source.connection.password",
		"retrieval.spec.physicalRestore.options.envs",
	}

	return NewMask(sensitive)
}
