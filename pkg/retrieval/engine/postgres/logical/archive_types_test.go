/*
2021 © Postgres.ai
*/

package logical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadingArchiveCommand(t *testing.T) {
	testCases := []struct {
		compressionType compressionType
		expectedCommand string
	}{
		{
			compressionType: gzipCompression,
			expectedCommand: "gunzip -c",
		},
		{
			compressionType: bzip2Compression,
			expectedCommand: "bunzip2 -c",
		},
		{
			compressionType: noCompression,
			expectedCommand: "cat",
		},
		{
			compressionType: compressionType(""),
			expectedCommand: "cat",
		},
	}

	for _, tc := range testCases {
		command := getReadingArchiveCommand(tc.compressionType)
		assert.Equal(t, tc.expectedCommand, command)
	}
}

func TestArchiveType(t *testing.T) {
	testCases := []struct {
		filename                string
		expectedCompressionType compressionType
	}{
		{
			filename:                "dump.sql.gz",
			expectedCompressionType: gzipCompression,
		},
		{
			filename:                "dump.bz2",
			expectedCompressionType: bzip2Compression,
		},
		{
			filename:                "test.dmp",
			expectedCompressionType: noCompression,
		},
		{
			filename:                "dump",
			expectedCompressionType: noCompression,
		},
	}

	for _, tc := range testCases {
		compressionType := getCompressionType(tc.filename)
		assert.Equal(t, tc.expectedCompressionType, compressionType)
	}
}
