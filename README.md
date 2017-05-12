xds-make: wrapper on make for XDS
=================================

xds-make is a wrapper on make command for X(cross) Development System.

This tool can be used in lieu of "standard" `make` command to trigger build of
your application by a remote `xds-server`.
`xds-make` uses [Syncthing](https://syncthing.net/) tool to synchronize your
projects files from your machine to the XDS build server machine (or container).

> **NOTE**: For now, only Syncthing sharing method is supported to synchronize
projects files.

> **SEE ALSO**: [xds-server](https://github.com/iotbzh/xds-server), a web server
used to remotely cross build applications.


## How to build

### Prerequisites
 You must install and setup [Go](https://golang.org/doc/install) version 1.7 or
 higher to compile this tool.

### Building
Clone this repo into your `$GOPATH/src/github.com/iotbzh` and use delivered Makefile:
```bash
 mkdir -p $GOPATH/src/github.com/iotbzh
 cd $GOPATH/src/github.com/iotbzh
 git clone https://github.com/iotbzh/xds-make.git
 cd xds-make
 make all
```

## How to use xds-make

You must have a running `xds-server` (locally or on the Cloud), see
[README.txt of xds-server](https://github.com/iotbzh/xds-server/blob/master/README.md)
for more details.

Then connect your favorite Web browser to the XDS dashboard (default url
http://localhost:8000) and follow instructions to start local source file
synchronizer (eg. Syncthing) and then create your project.

`XDS_PROJECT_ID` environment variable should be used to specify which project
you want to build.
Used `--list` option to list all existing projects ID:
```bash
./bin/xds-make --list

List of existing projects:
  CKI7R47-UWNDQC3_myProject
  CKI7R47-UWNDQC3_test2
  CKI7R47-UWNDQC3_test3
```

You are now ready to cross build your project. For example:
```bash
export XDS_PROJECT_ID=CKI7R47-UWNDQC3_myProject
export XDS_SERVER_URL=http://localhost:8000
export XDS_RPATH=<<local_path_of_my_project>>
./bin/xds-make clean
./bin/xds-make -j all
```

You can also add the build directory of `xds-make` to your `PATH` in order to
use the symbolic link `./bin/make -> ./bin/xds-make` and overwrite the native
`make` command.

```bash
export PATH=<<directory_of_xds_make_repo>>/bin:$PATH

export XDS_PROJECT_ID=CKI7R47-UWNDQC3_myProject
export XDS_SERVER_URL=http://localhost:8000
cd <<local_path_of_my_project>>
make clean
make all
```

## Usage

```bash
./bin/xds-make --help

NAME:
   xds-make - wrapper on make for X(cross) Development System.

USAGE:
   xds-make [global options] command [command options] [arguments...]

VERSION:
   1.0.0 (4e22f6f)

DESCRIPTION:
   make utility of X(cross) Development System

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
   --list         list existing projects
   --help, -h     show help
   --version, -v  print the version

COPYRIGHT:
   Apache-2.0
```

## Debug

Visual Studio Code launcher settings can be found into `.vscode/launch.json`.

>**Tricks:** To debug both `xds-make` (client part) and `xds-server` (server part),
it may be useful use the same local sources.
So you should replace `xds-server` in `vendor` directory by a symlink.
So clone first `xds-server` sources next to `xds-make` directory.
You should have the following tree:
```
> tree -L 3 src
src
|-- github.com
    |-- iotbzh
       |-- xds-make
       |-- xds-server
```
Then invoke `vendor/debug` Makefile rule to create a symlink inside vendor
directory :
```bash
cd src/github.com/iotbzh/xds-make
make vendor/debug
```
