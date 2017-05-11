xds-make: wrapper on make for XDS
=================================

xds-make is a wrapper on make for Cross Development System.

## How to build

### Prerequisites
 You must install and setup [go](https://golang.org/doc/install) version 1.7 or
 higher to compile this tool.

### Build procedure
Just use delivered Makefile
```bash
make build
```

## Usage
You must create first your XDS project using XDS dashboard.
Then you must specify the project ID using `XDS_PROJECT_ID` environment variable,
and finally you must specify the XDS server url using `XDS_SERVER_URL`.

For example to call the `clean` rule of your project:
```bash
 export XDS_PROJECT_ID=CKI7R47-UWNDQC3_myProject
 export XDS_SERVER_URL=http://localhost:8000
 cd <local_path_of_my_project>
 xds-make clean
```

You can also add the directory where you build this tool into your `PATH` and
use the symbolic link named `make` to overwrite your native `make`.

```bash
export XDS_PROJECT_ID=CKI7R47-UWNDQC3_myProject
export XDS_SERVER_URL=http://localhost:8000
export PATH=<directory_bin_xds-make>:$PATH
cd <local_path_of_my_project>
make clean
```

## Help

```bash
> xds-make help

NAME:
   xds-make - Usage: wrapper on make for Cross Development System.

USAGE:
   xds-make [global options] command [command options] [arguments...]

VERSION:
   0.0.1 (812a4c3)

DESCRIPTION:
   make utility of Cross Development System

ENVIRONMENT VARIABLES:
 XDS_PROJECT_ID      project ID you want to build (mandatory variable)
 XDS_LOGLEVEL        logging level (supported levels: panic, fatal, error, warn, info, debug)
 XDS_RPATH           relative path into project
 XDS_TIMESTAMP       prefix output with timestamp
 XDS_SERVER_URL      remote XDS server url (default http://localhost:8000)

AUTHOR:
   Sebastien Douheret <sebastien@iot.bzh>

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version

COPYRIGHT:
   Apache 2
```

## Debug

VSCode launcher settings can be found into `.vscode/launch.json`

_Tricks:_ To develop both client command tool (xds-make) and server part
(xds-server), it may be useful use the same local sources.
So you should replace xds-server directory in vendor by a symlink.
So clone first `xds_server` next to `xds-make` directory.
You should have the following tree:
```
> tree -L 3 src
src
|-- github.com
    |-- iotbzh
       |-- xds-make
       |-- xds-server
```
Then you use `vendor/debug` rule to create a symlink inside vendor directory
```bash
make vendor/debug
```
