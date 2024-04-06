package download

import (
	"crypto/md5"
	"fmt"
	"os"
	"sort"
)

// allSourcesMatchFileMetadata returns false if there is a mismatch in the file metadata
// across the sources. The checkETag parameter can be used to optionally consider the ETag
// consistency across the sources as well.
func allSourcesMatchFileMetadata(srcFileMetas []sourceFileMetadata, checkETag bool) bool {
	for i := 1; i < len(srcFileMetas); i++ {
		if srcFileMetas[i].size != srcFileMetas[i-1].size {
			return false
		}

		if checkETag && srcFileMetas[i].eTag != srcFileMetas[i-1].eTag {
			return false
		}
	}

	return true
}

// sourceUrlsSortedByEstLatency returns the source URLs sorted by the estimated latency
// of the sources in ascending order.
func sourceUrlsSortedByEstLatency(srcFileMetas []sourceFileMetadata) []string {
	// just to avoid parameter mutation
	srcFileMetasCopy := make([]sourceFileMetadata, len(srcFileMetas))
	copy(srcFileMetasCopy, srcFileMetas)

	sort.Slice(srcFileMetasCopy, func(i, j int) bool {
		return srcFileMetasCopy[i].estLatency < srcFileMetasCopy[j].estLatency
	})

	var sourceUrls []string
	for _, sfm := range srcFileMetasCopy {
		sourceUrls = append(sourceUrls, sfm.url)
	}

	return sourceUrls
}

// getMD5Hash calculates the MD5 hash of the file contents and returns
// the hex encoding.
func getMD5Hash(file *os.File) (string, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return "", err
	}

	b := make([]byte, fileInfo.Size())
	_, err = file.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", md5.Sum(b)), nil
}

// min returns the minimum of two numbers.
func min(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}
