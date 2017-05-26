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

var appAuthors = []cli.Author{
	cli.Author{Name: "Sebastien Douheret", Email: "sebastien@iot.bzh"},
}

// AppName name of this application
var AppName = ""

// AppNativeName native command name that this application can overload
var AppNativeName = ""

// AppVersion Version of this application
var AppVersion = "?.?.?"

// AppSubVersion is the git tag id added to version string
// Should be set by compilation -ldflags "-X main.AppSubVersion=xxx"
var AppSubVersion = "unknown-dev"

// Create logger
var log = logrus.New()

// Application details
const (
	appCopyright    = "Apache-2.0"
	defaultLogLevel = "error"
)

// main
func main() {
	var uri, prjID, rPath, logLevel, sdkid string
	var withTimestamp, listProject bool

	// Allow to set app name from exec (useful for debugging)
	if AppName == "" {
		AppName = os.Getenv("XDS_APPNAME")
	}
	if AppName == "" {
		panic("Invalid setup, AppName not define !")
	}
	if AppNativeName == "" {
		AppNativeName = AppName[4:]
	}
	appDescription := fmt.Sprintf("%s utility of X(cross) Development System\n", AppNativeName)
	appUsage := fmt.Sprintf("wrapper on %s for X(cross) Development System.", AppNativeName)

	// Create a new App instance
	app := cli.NewApp()
	app.Name = AppName
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
			Usage:       "project ID you want to build (mandatory variable)",
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
			Usage:       "logging level (supported levels: panic, fatal, error, warn, info, debug)",
			Value:       defaultLogLevel,
			Destination: &logLevel,
		},
		cli.StringFlag{
			Name:        "rpath",
			EnvVar:      "XDS_RPATH",
			Usage:       "relative path into project",
			Destination: &rPath,
		},
		cli.StringFlag{
			Name:        "sdkid",
			EnvVar:      "XDS_SDK_ID",
			Usage:       "Cross Sdk ID to use to build project",
			Destination: &sdkid,
		},
		cli.BoolFlag{
			Name:        "timestamp, ts",
			EnvVar:      "XDS_TIMESTAMP",
			Usage:       "prefix output with timestamp",
			Destination: &withTimestamp,
		},
		cli.StringFlag{
			Name:        "url",
			EnvVar:      "XDS_SERVER_URL",
			Value:       "localhost:8000",
			Usage:       "remote XDS server url",
			Destination: &uri,
		},
	}

	// Create env vars help
	dynDesc := "\nENVIRONMENT VARIABLES:"
	for _, f := range app.Flags {
		var env, usage string
		switch f.(type) {
		case cli.StringFlag:
			fs := f.(cli.StringFlag)
			env = fs.EnvVar
			usage = fs.Usage
		case cli.BoolFlag:
			fb := f.(cli.BoolFlag)
			env = fb.EnvVar
			usage = fb.Usage
		default:
			panic("Un-implemented option type")
		}
		if env != "" {
			dynDesc += fmt.Sprintf("\n %s \t\t %s", env, usage)
		}
	}
	app.Description = appDescription + dynDesc

	args := make([]string, len(os.Args))
	args[0] = os.Args[0]
	argsCommand := make([]string, len(os.Args))
	exeName := filepath.Base(os.Args[0])

	// Split xds-xxx options from native command (eg. make) options
	// only process args before skip arguments, IOW before '--'
	found := false
	if exeName != AppNativeName {
		for idx, a := range os.Args[1:] {
			if a == "--" {
				// Detect skip option (IOW '--') to split arguments
				copy(args, os.Args[0:idx+1])
				copy(argsCommand, os.Args[idx+2:])
				found = true
				goto exit_loop
			}
		}
	exit_loop:
		if !found {
			copy(args, os.Args)
		}
	} else {
		copy(argsCommand, os.Args)
	}

	// only one action
	app.Action = func(ctx *cli.Context) error {
		var err error

		var execCommand, ccHelp string
		switch AppName {
		case "xds-make":
			execCommand = "/make"
			ccHelp = "all"
		case "xds-exec":
			execCommand = "/exec"
			ccHelp = "'mkdir build; cd build; cmake ..'"
		default:
			panic("Un-implemented command")
		}

		// Set logger level and formatter
		if log.Level, err = logrus.ParseLevel(logLevel); err != nil {
			fmt.Printf("Invalid log level : \"%v\"\n", logLevel)
			os.Exit(1)
		}
		log.Formatter = &logrus.TextFormatter{}

		log.Infof("Execute: %s %v", execCommand, argsCommand)

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
				msg += "List of existing projects (use: export XDS_PROJECT_ID=<< ID >>): \n"
				msg += "  ID\t\t\t\t | Label"
				for _, f := range folders {
					msg += fmt.Sprintf("\n  %s\t | %s", f.ID, f.Label)
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
				msg += "\nList of installed cross SDKs (use: export XDS_SDK_ID=<< ID >>): \n"
				msg += "  ID\t\t\t\t\t | NAME\n"
				for _, s := range sdks {
					msg += fmt.Sprintf("  %s\t | %s\n", s.ID, s.Name)
				}
			}

			if len(folders) > 0 && len(sdks) > 0 {
				msg += fmt.Sprintf("\n")
				msg += fmt.Sprintf("For example: \n")
				msg += fmt.Sprintf("  %s --id %q --sdkid %q -- %s\n", AppName, folders[0].ID, sdks[0].ID, ccHelp)
				msg += " or\n"
				msg += fmt.Sprintf("  XDS_PROJECT_ID=%q XDS_SDK_ID=%q  %s %s\n", folders[0].ID, sdks[0].ID, AppNativeName, ccHelp)
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

		outFunc := func(timestamp, stdout, stderr string) {
			tm := ""
			if withTimestamp {
				tm = timestamp + "| "
			}
			if withTimestamp {
				tm = timestamp + "| "
			}
			if stdout != "" {
				fmt.Printf("%s%s\n", tm, stdout)
			}
			if stderr != "" {
				fmt.Fprintf(os.Stderr, "%s%s\n", tm, stderr)
			}
		}

		switch AppName {
		case "xds-make":
			iosk.On(apiv1.MakeOutEvent, func(ev apiv1.MakeOutMsg) {
				outFunc(ev.Timestamp, ev.Stdout, ev.Stderr)
			})

			iosk.On(apiv1.MakeExitEvent, func(ev apiv1.MakeExitMsg) {
				exitChan <- exitResult{ev.Error, ev.Code}
			})
		case "xds-exec":
			iosk.On(apiv1.ExecOutEvent, func(ev apiv1.ExecOutMsg) {
				outFunc(ev.Timestamp, ev.Stdout, ev.Stderr)
			})

			iosk.On(apiv1.ExecExitEvent, func(ev apiv1.ExecExitMsg) {
				exitChan <- exitResult{ev.Error, ev.Code}
			})
		}

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
		var body []byte
		switch AppName {
		case "xds-make":
			args := apiv1.MakeArgs{
				ID:         prjID,
				SdkID:      sdkid,
				Args:       argsCommand,
				Env:        []string{},
				RPath:      rPath,
				CmdTimeout: 60,
			}
			body, err = json.Marshal(args)

		case "xds-exec":
			args := apiv1.ExecArgs{
				ID:         prjID,
				SdkID:      sdkid,
				Cmd:        strings.Trim(strings.Join(argsCommand, " "), " "),
				Args:       []string{},
				Env:        []string{},
				RPath:      rPath,
				CmdTimeout: 60,
			}
			body, err = json.Marshal(args)
		}
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		log.Infof("POST %s%s %v", uri, execCommand, string(body))
		if err := c.HTTPPost(execCommand, string(body)); err != nil {
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
