package app

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/libcompose/cli/app"
	"github.com/docker/libcompose/cli/command"
	"github.com/docker/libcompose/cli/logger"
	"github.com/docker/libcompose/lookup"
	"github.com/docker/libcompose/project"
	rLookup "github.com/rancher/rancher-compose/lookup"
	"github.com/rancher/rancher-compose/rancher"
	"github.com/rancher/rancher-compose/upgrade"
)

type ProjectFactory struct {
}

func (p *ProjectFactory) Create(c *cli.Context) (*project.Project, error) {
	envLookup, err := rLookup.NewFileEnvLookup(c.GlobalString("env-file"), &lookup.OsEnvLookup{})
	if err != nil {
		return nil, err
	}

	context := &rancher.Context{
		Context: project.Context{
			ConfigLookup:      &lookup.FileConfigLookup{},
			EnvironmentLookup: envLookup,
			LoggerFactory:     logger.NewColorLoggerFactory(),
		},
		RancherComposeFile: c.GlobalString("rancher-file"),
		Url:                c.GlobalString("url"),
		AccessKey:          c.GlobalString("access-key"),
		SecretKey:          c.GlobalString("secret-key"),
		PullCached:         c.Bool("cached"),
		Uploader:           &rancher.S3Uploader{},
	}
	command.Populate(&context.Context, c)

	return rancher.NewProject(context)
}

func UpgradeCommand(factory app.ProjectFactory) cli.Command {
	return cli.Command{
		Name:   "upgrade",
		Usage:  "Perform rolling upgrade between services",
		Action: app.WithProject(factory, Upgrade),
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "batch-size",
				Usage: "Number of containers to upgrade at once",
				Value: 2,
			},
			cli.IntFlag{
				Name:  "scale",
				Usage: "Final number of running containers",
				Value: -1,
			},
			cli.IntFlag{
				Name:  "interval",
				Usage: "Update interval in milliseconds",
				Value: 2000,
			},
			cli.BoolTFlag{
				Name:  "update-links",
				Usage: "Update inbound links on target service",
			},
			cli.BoolFlag{
				Name:  "wait,w",
				Usage: "Wait for upgrade to complete",
			},
			cli.BoolFlag{
				Name:  "pull, p",
				Usage: "Before doing the upgrade do an image pull on all hosts that have the image already",
			},
			cli.BoolFlag{
				Name:  "cleanup, c",
				Usage: "Remove the original service definition once upgraded, implies --wait",
			},
		},
	}
}

func PullCommand(factory app.ProjectFactory) cli.Command {
	return cli.Command{
		Name:   "pull",
		Usage:  "Pulls images for services",
		Action: app.WithProject(factory, app.ProjectPull),
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "cached, c",
				Usage: "Only update hosts that have the image cached, don't pull new",
			},
		},
	}
}

func Upgrade(p *project.Project, c *cli.Context) {
	args := c.Args()
	if len(args) != 2 {
		logrus.Fatalf("Please pass arguments in the form: [from service] [to service]")
	}

	err := upgrade.Upgrade(p, args[0], args[1], upgrade.UpgradeOpts{
		BatchSize:      c.Int("batch-size"),
		IntervalMillis: c.Int("interval"),
		FinalScale:     c.Int("scale"),
		UpdateLinks:    c.Bool("update-links"),
		Wait:           c.Bool("wait"),
		CleanUp:        c.Bool("cleanup"),
		Pull:           c.Bool("pull"),
	})

	if err != nil {
		logrus.Fatal(err)
	}
}