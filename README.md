# multisource-downloader

Download accelerator that supports fetching a file from multiple sources concurrently.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)
- [Make](https://www.gnu.org/software/make/)

_i.e., no need to have Go installed locally_

## Usage

### Make targets

#### tidy dependencies

```bash
$ make deps
```

#### format all `.go` files in project

```bash
$ make fmt
```

#### run tests

```bash
$ make test
```

#### build executable binary

```bash
$ make build target={target OS}
```
- generates `msdl` inside `bin/` directory
- possible values for `target` can be found [here](https://github.com/golang/go/blob/58c5db3169c801737cb0e0ed4886554763c861eb/src/go/build/syslist.go#L14-L33)
- example: `make build target=linux`

### CLI reference

Running `msdl` (after generating it using [`make build`](#build-executable-binary)):

#### synopsis
`msdl [flags...] [space-delimited URLs...]`


#### example
```bash
$ ./msdl -f destfile.txt http://source1.com/a.txt http://source2.com/a.txt http://source3.com/a.txt
```
- downloads `a.txt` from 3 different sources concurrently and saves it to a local file named `destfile.txt`
- note: the filenames can be different in the sources as long as they are effectively the same file

#### available flags
```
-c, --connections uint   max number of concurrent connections [optional; default 5]
    --etag               check ETag match (using MD5 hash of downloaded file) if available [optional; default false]
-f, --file string        destination file path [required for download]
-h, --help               help for msdl
-q, --quiet              disable logging to stdout [optional; default false]
-t, --timeout uint       timeout for each connection in seconds [optional; default 10]
```
