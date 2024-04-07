package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/gkatanacio/multisource-downloader/download"
)

var downloadOpts download.Options

var rootCmd = &cobra.Command{
	Use:          "msdl [space-delimited URLs]",
	Short:        "Download accelerator that allows fetching a file from multiple sources concurrently.",
	Example:      "./msdl -c 8 -t 10 --etag -f destfile.txt http://source1.com/a.txt http://source2.com/a.txt http://source3.com/a.txt",
	SilenceUsage: true,
	Args: func(cmd *cobra.Command, args []string) error {
		return cobra.MinimumNArgs(1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		downloadService := download.NewService(downloadOpts)
		return downloadService.Download(args)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().UintVarP(&downloadOpts.Connections, "connections", "c", 5, "max number of concurrent connections")
	rootCmd.Flags().UintVarP(&downloadOpts.Timeout, "timeout", "t", 10, "timeout for each connection in seconds")
	rootCmd.Flags().BoolVar(&downloadOpts.CheckETag, "etag", false, "check ETag match (using MD5 hash of downloaded file) if available")
	rootCmd.Flags().StringVarP(&downloadOpts.DestFilePath, "file", "f", "", "destination file path")

	rootCmd.MarkFlagRequired("file")
}
