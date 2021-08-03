/*
2021 © Postgres.ai
*/

package logical

import (
	"path/filepath"
)

type compressionType string

const (
	noCompression    compressionType = "no"
	gzipCompression  compressionType = "gzip"
	bzip2Compression compressionType = "bzip2"
)

// getReadingArchiveCommand chooses command to read dump file.
func getReadingArchiveCommand(compressionType compressionType) string {
	switch compressionType {
	case gzipCompression:
		return "gunzip -c"

	case bzip2Compression:
		return "bunzip2 -c"

	default:
		return "cat"
	}
}

// getCompressionType returns archive type based on filename extension.
func getCompressionType(filename string) compressionType {
	switch filepath.Ext(filename) {
	case ".gz":
		return gzipCompression

	case ".bz2":
		return bzip2Compression

	default:
		return noCompression
	}
}
