package yaml

// DefaultConfigMask return default config copy configuration
func DefaultConfigMask() *Mask {
	sensitive := []string{
		"server.verificationToken",
		"platform.accessToken",
		"retrieval.spec.logicalDump.options.source.connection.password",
	}

	return NewMask(sensitive)
}
