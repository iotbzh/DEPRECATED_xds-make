// TODO add Doc
//
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/iotbzh/xds-server/lib/apiv1"
	"github.com/iotbzh/xds-server/lib/common"
	"github.com/iotbzh/xds-server/lib/crosssdk"
	"github.com/iotbzh/xds-server/lib/xdsconfig"
	"github.com/zhouhui8915/go-socket.io-client"
)

const (
	appName         = "xds-make"
	appDescription  = "make utility of X(cross) Development System\n"
	appCopyright    = "Apache-2.0"
	appUsage        = "wrapper on make for X(cross) Development System."
	defaultLogLevel = "error"
)

var appAuthors = []cli.Author{
	cli.Author{Name: "Sebastien Douheret", Email: "sebastien@iot.bzh"},
}

// AppVersion is the version of this application
var AppVersion = "?.?.?"

// AppSubVersion is the git tag id added to version string
// Should be set by compilation -ldflags "-X main.AppSubVersion=xxx"
var AppSubVersion = "unknown-dev"

// Create logger
var log = logrus.New()

// ExecCommand is the HTTP url command to execute
var ExecCommand = "/make"

// main
func main() {
	var uri, prjID, rPath, logLevel, sdkid string
	var withTimestamp, listProject bool

	// Create a new App instance
	app := cli.NewApp()
	app.Name = appName
	app.Usage = appUsage
	app.Version = AppVersion + " (" + AppSubVersion + ")"
	app.Authors = appAuthors
	app.Copyright = appCopyright
	app.Metadata = make(map[string]interface{})
	app.Metadata["version"] = AppVersion
	app.Metadata["git-tag"] = AppSubVersion
	app.Metadata["logger"] = log

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "id",
			EnvVar:      "XDS_PROJECT_ID",
			Hidden:      true,
			Usage:       "project ID you want to build",
			Destination: &prjID,
		},
		cli.BoolFlag{
			Name:        "list, ls",
			Usage:       "list existing projects",
			Destination: &listProject,
		},
		cli.StringFlag{
			Name:        "log",
			EnvVar:      "XDS_LOGLEVEL",
			Hidden:      true,
			Usage:       "logging level (supported levels: panic, fatal, error, warn, info, debug)",
			Value:       defaultLogLevel,
			Destination: &logLevel,
		},
		cli.StringFlag{
			Name:        "rpath",
			EnvVar:      "XDS_RPATH",
			Hidden:      true,
			Usage:       "relative path into project",
			Destination: &rPath,
		},
		cli.StringFlag{
			Name:        "sdkid",
			EnvVar:      "XDS_SDK_ID",
			Hidden:      true,
			Usage:       "SDK ID to use to compile project",
			Destination: &sdkid,
		},
		cli.BoolFlag{
			Name:        "timestamp, ts",
			EnvVar:      "XDS_TIMESTAMP",
			Hidden:      true,
			Usage:       "prefix output with timestamp",
			Destination: &withTimestamp,
		},
		cli.StringFlag{
			Name:        "url",
			EnvVar:      "XDS_SERVER_URL",
			Hidden:      true,
			Value:       "localhost:8000",
			Usage:       "remote XDS server url",
			Destination: &uri,
		},
	}

	// FIXME - don't duplicate, but reuse flag definition
	dynDesc := "\nENVIRONMENT VARIABLES:" +
		"\n XDS_PROJECT_ID      project ID you want to build (mandatory variable)" +
		"\n XDS_LOGLEVEL        logging level (supported levels: panic, fatal, error, warn, info, debug)" +
		"\n XDS_RPATH           relative path into project" +
		"\n XDS_SDK_ID          Cross Sdk ID to use to build project" +
		"\n XDS_TIMESTAMP       prefix output with timestamp" +
		"\n XDS_SERVER_URL      remote XDS server url (default http://localhost:8000)"

	app.Description = appDescription + dynDesc

	exeName := filepath.Base(os.Args[0])
	args := make([]string, len(os.Args))
	args[0] = os.Args[0]
	argsCommand := make([]string, len(os.Args))

	// Only decode arguments when executable is this wrapper
	// IOW, pass all arguments without processing when executable name is "make"
	if exeName != "make" {
		// only process args before skip arguments, IOW before '--'
		found := false
		for idx, a := range os.Args[1:] {
			switch a {
			// Allow to print help and version of this utility and
			// not help or version of sub-process
			case "-h", "--help", "-v", "--version", "--list", "-ls":
				args[1] = a
				found = true
				goto exit_loop

			// Detect skip option (IOW '--') to split arguments
			case "--":
				copy(args, os.Args[0:idx+1])
				copy(argsCommand, os.Args[idx+2:])
				found = true
				goto exit_loop
			}
		}
	exit_loop:
		if !found {
			copy(argsCommand, os.Args[1:])
		}
	} else {
		// Pass all arguments when invoked with executable name "make"
		copy(argsCommand, os.Args[1:])
	}

	// only one action
	app.Action = func(ctx *cli.Context) error {
		var err error

		// Set logger level and formatter
		if log.Level, err = logrus.ParseLevel(logLevel); err != nil {
			fmt.Printf("Invalid log level : \"%v\"\n", logLevel)
			os.Exit(1)
		}
		log.Formatter = &logrus.TextFormatter{}

		cmdArgs := strings.Trim(strings.Join(argsCommand, " "), " ")

		log.Infof("Execute: %s %v", ExecCommand, cmdArgs)

		// Define HTTP and WS url
		baseURL := uri
		if !strings.HasPrefix(uri, "http://") {
			baseURL = "http://" + uri
		}

		// Create HTTP client
		log.Debugln("Connect HTTP client on ", baseURL)
		conf := common.HTTPClientConfig{
			URLPrefix:           "/api/v1",
			HeaderClientKeyName: "XDS-SID",
			CsrfDisable:         true,
		}
		c, err := common.HTTPNewClient(baseURL, conf)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}

		// First call to check that daemon is alive
		var data []byte
		if err := c.HTTPGet("/folders", &data); err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		log.Infof("Result of /folders: %v", string(data[:]))

		folders := xdsconfig.FoldersConfig{}
		errMar := json.Unmarshal(data, &folders)

		// Check mandatory args
		if prjID == "" || listProject {
			msg := ""
			exc := 0
			if !listProject {
				msg = "XDS_PROJECT_ID environment variable must be set !\n"
				exc = 1
			}
			if errMar == nil {
				msg += "List of existing projects (ID - Label): "
				for _, f := range folders {
					msg += fmt.Sprintf("\n  %s\t - %s", f.ID, f.Label)
					if f.DefaultSdk != "" {
						msg += fmt.Sprintf("\t(default SDK: %s)", f.DefaultSdk)
					}
				}
				msg += "\n"
			}

			data = nil
			if err := c.HTTPGet("/sdks", &data); err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			log.Infof("Result of /sdks: %v", string(data[:]))

			sdks := []crosssdk.SDK{}
			errMar = json.Unmarshal(data, &sdks)
			if errMar == nil {
				msg += "\nList of installed cross SDKs (ID - Name):\n"
				for _, s := range sdks {
					msg += fmt.Sprintf("  %s\t - %s\n", s.ID, s.Name)
				}
			}

			return cli.NewExitError(msg, exc)
		}

		// Create io Websocket client
		log.Debugln("Connecting IO.socket client on ", baseURL)

		opts := &socketio_client.Options{
			Transport: "websocket",
			Header:    make(map[string][]string),
		}
		opts.Header["XDS-SID"] = []string{c.GetClientID()}

		iosk, err := socketio_client.NewClient(baseURL, opts)
		if err != nil {
			return cli.NewExitError("IO.socket connection error: "+err.Error(), 1)
		}

		// Process Socket IO events
		type exitResult struct {
			error error
			code  int
		}
		exitChan := make(chan exitResult, 1)

		iosk.On("error", func(err error) {
			fmt.Println("ERROR: ", err.Error())
		})

		iosk.On("disconnection", func(err error) {
			exitChan <- exitResult{err, 2}
		})

		iosk.On(apiv1.MakeOutEvent, func(ev apiv1.MakeOutMsg) {
			tm := ""
			if withTimestamp {
				tm = ev.Timestamp + "| "
			}
			if ev.Stdout != "" {
				fmt.Printf("%s%s\n", tm, ev.Stdout)
			}
			if ev.Stderr != "" {
				fmt.Fprintf(os.Stderr, "%s%s\n", tm, ev.Stderr)
			}
		})

		iosk.On(apiv1.MakeExitEvent, func(ev apiv1.MakeExitMsg) {
			exitChan <- exitResult{ev.Error, ev.Code}
		})

		// Retrieve the folder definition
		folder := &xdsconfig.FolderConfig{}
		for _, f := range folders {
			if f.ID == prjID {
				folder = &f
				break
			}
		}

		// Auto setup rPath if needed
		if rPath == "" && folder != nil {
			cwd, err := os.Getwd()
			if err == nil {
				fldRp := folder.RelativePath
				if !strings.HasPrefix(fldRp, "/") {
					fldRp = "/" + fldRp
				}
				log.Debugf("Try to auto-setup rPath: cwd=%s ; RelativePath=%s", cwd, fldRp)
				if sp := strings.SplitAfter(cwd, fldRp); len(sp) == 2 {
					rPath = strings.Trim(sp[1], "/")
					log.Debugf("Auto-setup rPath to: '%s'", rPath)
				}
			}
		}

		// Send build command
		args := apiv1.MakeArgs{
			ID:         prjID,
			RPath:      rPath,
			Args:       cmdArgs,
			SdkID:      sdkid,
			CmdTimeout: 60,
		}
		body, err := json.Marshal(args)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		log.Infof("POST %s%s %v", uri, ExecCommand, string(body))
		if err := c.HTTPPost(ExecCommand, string(body)); err != nil {
			return cli.NewExitError(err.Error(), 1)
		}

		// Wait exit
		select {
		case res := <-exitChan:
			errStr := ""
			if res.code == 0 {
				log.Debugln("Exit successfully")
			}
			if res.error != nil {
				log.Debugln("Exit with ERROR: ", res.error.Error())
				errStr = res.error.Error()
			}
			return cli.NewExitError(errStr, res.code)
		}
	}

	app.Run(args)
}
