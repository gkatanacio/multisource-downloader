package download

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

var (
	ErrNoSourceUrls                  = errors.New("source URLs required")
	ErrSourcesFileMismatch           = errors.New("file metadata from source URLs are not matching")
	ErrETagMismatch                  = errors.New("ETag mismatch")
	ErrUnknownContentLength          = errors.New("unknown content length")
	ErrPartialRequestUnsupported     = errors.New("partial request not supported")
	ErrFailedChunkDownloadAllSources = errors.New("failed to download chunk after attempting from all sources")
)

const suffixOngoingDownload = ".download"

// Service is the service layer that contains operations for downloading.
type Service struct {
	opts          Options
	calculateETag ETagCalculator
	httpClient    *http.Client
}

func NewService(opts Options, calculateETag ETagCalculator) *Service {
	return &Service{
		opts:          opts,
		calculateETag: calculateETag,
		httpClient: &http.Client{
			Timeout: time.Second * time.Duration(opts.Timeout),
		},
	}
}

// Download attempts to download a file from the given sources in a concurrent manner (i.e., in chunks).
// This creates a temporary file while the download is ongoing and moves it to the actual configured
// destination file once the download is successfully completed.
func (s *Service) Download(sourceUrls []string) error {
	if len(sourceUrls) == 0 {
		return ErrNoSourceUrls
	}

	srcFileMetas, err := s.fetchFileMetadataFromSources(sourceUrls)
	if err != nil {
		return err
	}

	if !allSourcesMatchFileMetadata(srcFileMetas, s.opts.CheckETag) {
		return ErrSourcesFileMismatch
	}

	fileMetadata := srcFileMetas[0].fileMetadata // any will do since they are assumed to be matching

	ongoingDownloadFile, err := os.Create(s.opts.DestFilePath + suffixOngoingDownload)
	if err != nil {
		return err
	}
	defer ongoingDownloadFile.Close()

	if err := s.downloadFileContents(
		sourceUrlsSortedByEstLatency(srcFileMetas), // sort to prioritize sources with lowest estimated latency
		fileMetadata,
		ongoingDownloadFile,
	); err != nil {
		return err
	}

	if s.opts.CheckETag && len(fileMetadata.eTag) > 0 {
		calculatedETag, err := s.calculateETag(ongoingDownloadFile)
		if err != nil {
			return err
		}

		if calculatedETag != fileMetadata.eTag {
			return ErrETagMismatch
		}
	}

	if err := os.Rename(ongoingDownloadFile.Name(), s.opts.DestFilePath); err != nil {
		return err
	}

	s.logln("Download complete:", s.opts.DestFilePath)

	return nil
}

// fetchFileMetadataFromSources returns file metadata corresponding to each of the given sources.
func (s *Service) fetchFileMetadataFromSources(sourceUrls []string) ([]sourceFileMetadata, error) {
	srcFileMetasChan := make(chan sourceFileMetadata)

	eg, ctx := errgroup.WithContext(context.Background())

	for _, url := range sourceUrls {
		eg.Go(func() error {
			req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
			if err != nil {
				return err
			}

			start := time.Now()

			resp, err := s.httpClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			estLatency := time.Since(start)

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("received %d response from %s", resp.StatusCode, url)
			}

			if resp.ContentLength == -1 {
				return ErrUnknownContentLength
			}

			acceptRanges := resp.Header.Get("Accept-Ranges")
			if len(acceptRanges) == 0 || acceptRanges == "none" {
				return ErrPartialRequestUnsupported
			}

			srcFileMetasChan <- sourceFileMetadata{
				url:        url,
				estLatency: estLatency,
				fileMetadata: fileMetadata{
					size:        resp.ContentLength,
					contentType: resp.Header.Get("Content-Type"),
					eTag:        strings.Trim(resp.Header.Get("ETag"), `"`),
				},
			}

			return nil
		})
	}

	go func() {
		eg.Wait()
		close(srcFileMetasChan)
	}()

	var srcFileMetas []sourceFileMetadata
	for sfm := range srcFileMetasChan {
		srcFileMetas = append(srcFileMetas, sfm)
	}

	return srcFileMetas, eg.Wait()
}

// downloadFileContents downloads the file contents from the given source URLs in chunks and
// writes them in proper order in the provided destination file. The source URLs are prioritized
// based on their ordering in the given slice.
func (s *Service) downloadFileContents(sourceUrls []string, fileMetadata fileMetadata, destFile *os.File) error {
	chunkSize := fileMetadata.size / int64(s.opts.Connections)

	eg, ctx := errgroup.WithContext(context.Background())
	eg.SetLimit(int(s.opts.Connections))

	for offset, i := int64(0), 0; offset < fileMetadata.size; offset, i = offset+chunkSize, i+1 {
		srcIdxInitAttempt := i % len(sourceUrls)
		limit := min(offset+chunkSize, fileMetadata.size)

		eg.Go(func() error {
			var chunk []byte
			url := sourceUrls[srcIdxInitAttempt]

			chunk, err := s.fetchChunk(ctx, url, offset, limit)
			if err != nil {
				printErr(fmt.Errorf("failed initial download of chunk %d from %s: %w", i, url, err))

				// try to download chunk from other sources (priority based on sourceUrls ordering)
				for j := 0; j < len(sourceUrls) && err != nil; j++ {
					// stop retrying if context already done (e.g., error returned in another goroutine)
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}

					if j == srcIdxInitAttempt {
						continue
					}

					url = sourceUrls[j]
					chunk, err = s.fetchChunk(ctx, url, offset, limit)
					if err != nil {
						printErr(fmt.Errorf("failed download retry of chunk %d from %s: %w", i, url, err))
					}
				}

				if err != nil {
					return ErrFailedChunkDownloadAllSources
				}
			}

			s.logln(fmt.Sprintf("chunk %d downloaded from %s", i, url))

			_, err = io.Copy(io.NewOffsetWriter(destFile, offset), bytes.NewReader(chunk))
			return err
		})
	}

	return eg.Wait()
}

// fetchChunk attempts to GET a chunk of the file from the given URL.
func (s *Service) fetchChunk(ctx context.Context, url string, start, end int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("received %d response from %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}

// logln prints the arguments (separated by space) and a newline if the service is not in quiet mode.
func (s *Service) logln(args ...any) {
	if !s.opts.Quiet {
		fmt.Println(args...)
	}
}
