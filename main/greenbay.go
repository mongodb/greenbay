package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mongodb/greenbay/operations"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
	"github.com/urfave/cli"
)

func main() {
	// this is where the main action of the program starts. The
	// command line interface is managed by the cli package and
	// its objects/structures. This, plus the basic configuration
	// in buildApp(), is all that's necessary for bootstrapping the
	// environment.
	app := buildApp()
	err := app.Run(os.Args)
	grip.CatchErrorFatal(err)
}

////////////////////////////////////////////////////////////////////////
//
// Set up cli.App environment, configure logging and register sub-commands
//
////////////////////////////////////////////////////////////////////////

// we build the app outside of main so that we can test the operation
func buildApp() *cli.App {
	app := cli.NewApp()
	app.Name = "curator"
	app.Usage = "a package repository generation tool."
	app.Version = "0.0.1-pre"

	// Register sub-commands here.
	app.Commands = []cli.Command{
		checks(),
	}

	// These are global options. Use this to configure logging or
	// other options independent from specific sub commands.
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "level",
			Value: "info",
			Usage: "Specify lowest visible loglevel as string: 'emergency|alert|critical|error|warning|notice|info|debug'",
		},
	}

	app.Before = func(c *cli.Context) error {
		return errors.Wrap(loggingSetup(app.Name, c.String("level")),
			"problem setting log level")
	}

	return app
}

// logging setup is separate to make it unit testable
func loggingSetup(name, level string) error {
	// grip is a systemd/standard logging wrapper.
	grip.SetName(name)
	grip.SetThreshold(level)

	// This set's the logging system to write logging messages to
	// standard output.
	//
	// Could also call "grip.UseSystemdLogger()" to write log
	// messages directly to systemd's journald logger,
	// grip.UseFileLogger(<filename>), to write log messages to a
	// file, among other possible logging backends.
	return errors.Wrap(grip.UseNativeLogger(), "issue setting logging backend.")
}

////////////////////////////////////////////////////////////////////////
//
// Define SubCommands
//
////////////////////////////////////////////////////////////////////////

func checks() cli.Command {
	defaultNumJobs := runtime.NumCPU()
	cwd, _ := os.Getwd()
	configPath := filepath.Join(cwd, "greenbay.yaml")

	return cli.Command{
		Name:  "run",
		Usage: "run greenbay suites",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name: "jobs",
				Usage: fmt.Sprintf("specify the number of parallel tests to run. (Default %s)",
					defaultNumJobs),
				Value: defaultNumJobs,
			},
			cli.StringFlag{
				Name: "conf",
				Usage: fmt.Sprintln("path to config file. '.json', '.yaml', and '.yml' extensions ",
					"supported.", "Default path:", configPath),
				Value: configPath,
			},
			cli.StringFlag{
				Name:  "output",
				Usage: "path of file to write output too. Defaults to *not* writing output to a file",
				Value: "",
			},
			cli.BoolFlag{
				Name:  "quiet",
				Usage: "specify to disable printed (standard output) results",
			},
			cli.StringFlag{
				Name: "format",
				Usage: fmt.Sprintln("Selects the output format, defautls to a format that mirrors gotest,",
					"but also supports evergreen's results format.",
					"Use either 'gotest' (default) or 'results'."),
				Value: "gotest",
			},
			cli.StringSliceFlag{
				Name:  "test",
				Usage: "specify a check, by name",
				Value: &cli.StringSlice{"base"},
			},
			cli.StringSliceFlag{
				Name:  "suite",
				Usage: "specify a suite or suites, by name",
				Value: &cli.StringSlice{"all"},
			},
		},
		Action: func(c *cli.Context) error {
			// Note: in the future in may make sense to
			// use this context to timeout the work of the
			// underlying processes.
			ctx := context.Background()

			app, err := operations.NewApp(
				c.String("conf"),
				c.String("output"),
				c.String("format"),
				c.Bool("quiet"),
				c.Int("jobs"),
				c.StringSlice("suite"),
				c.StringSlice("tests"))

			if err != nil {
				return errors.Wrap(err, "problem prepping to run tests")
			}

			return errors.Wrap(app.Run(ctx), "problem running tests")
		},
	}
}
