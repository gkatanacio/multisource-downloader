package download_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gkatanacio/multisource-downloader/download"
)

// Test servers are expected to be running and serving the files
// from the test fixtures directory (i.e, `testdata/`).
const (
	testServer1 = "http://test-server-1:8080"
	testServer2 = "http://test-server-2:8080"
)

func Test_Service_Download_Success(t *testing.T) {
	testCases := map[string]struct {
		opts       download.Options
		sourceUrls []string
	}{
		"single source, single connection": {
			opts: download.Options{
				Connections:  1,
				Timeout:      3,
				DestFilePath: "single_src_single_conn.txt",
			},
			sourceUrls: []string{
				fmt.Sprintf("%s/dummy.txt", testServer1),
			},
		},
		"single source, multiple connections": {
			opts: download.Options{
				Connections:  2,
				Timeout:      3,
				DestFilePath: "single_src_multi_conn.txt",
			},
			sourceUrls: []string{
				fmt.Sprintf("%s/dummy.txt", testServer1),
			},
		},
		"multiple sources": {
			opts: download.Options{
				Connections:  4,
				Timeout:      3,
				DestFilePath: "multi_src.txt",
			},
			sourceUrls: []string{
				fmt.Sprintf("%s/dummy.txt", testServer1),
				fmt.Sprintf("%s/dummy.txt", testServer2),
			},
		},
		"connections < sources": {
			opts: download.Options{
				Connections:  1,
				Timeout:      3,
				DestFilePath: "conn_less_src.txt",
			},
			sourceUrls: []string{
				fmt.Sprintf("%s/dummy.txt", testServer1),
				fmt.Sprintf("%s/dummy.txt", testServer2),
			},
		},
	}

	for scenario, tc := range testCases {
		t.Run(scenario, func(t *testing.T) {
			downloadService := download.NewService(tc.opts)

			err := downloadService.Download(tc.sourceUrls)
			assert.NoError(t, err)

			fileInfo, err := os.Stat(tc.opts.DestFilePath)
			assert.NoError(t, err)
			assert.NotEmpty(t, fileInfo.Size())

			if err := os.Remove(tc.opts.DestFilePath); err != nil {
				log.Fatal(err)
			}
		})
	}
}

func Test_Service_Download_Failed(t *testing.T) {
	testCases := map[string]struct {
		opts        download.Options
		sourceUrls  []string
		specificErr error
	}{
		"empty sourceUrls": {
			sourceUrls:  []string{},
			specificErr: download.ErrNoSourceUrls,
		},
		"mismatched file from sources": {
			sourceUrls: []string{
				fmt.Sprintf("%s/dummy.txt", testServer1),
				fmt.Sprintf("%s/dummy.png", testServer2),
			},
			specificErr: download.ErrSourcesFileMismatch,
		},
		"invalid source": {
			sourceUrls: []string{"http://qwkehbjalcnlsapjaxc/dummy.txt"},
		},
	}

	for scenario, tc := range testCases {
		t.Run(scenario, func(t *testing.T) {
			downloadService := download.NewService(tc.opts)

			err := downloadService.Download(tc.sourceUrls)
			assert.Error(t, err)

			if tc.specificErr != nil {
				assert.ErrorIs(t, err, tc.specificErr)
			}
		})
	}
}
