package download

import (
	"time"
)

// Options represents the configuration for the download service.
type Options struct {
	Connections  uint
	Timeout      uint
	CheckETag    bool
	DestFilePath string
}

// fileMetadata is comprised of relevant metadata for a download file.
type fileMetadata struct {
	size        int64
	contentType string
	eTag        string
}

// sourceFileMetadata represents the metadata of a file along with
// some details for its corresponding download source.
type sourceFileMetadata struct {
	fileMetadata
	url        string
	estLatency time.Duration
}
