package app

import (
	"os"
	"strings"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libcompose/cli/logger"
	rLookup "github.com/rancher/rancher-compose/lookup"
	"github.com/rancher/rancher-compose/project"
	"github.com/rancher/rancher-compose/project/options"
	"github.com/rancher/rancher-compose/rancher"
	"github.com/urfave/cli"
)

type ProjectFactory struct {
}

func (p *ProjectFactory) Create(c *cli.Context) (project.APIProject, error) {
	/*rancherComposeFile, err := rancher.ResolveRancherCompose(c.GlobalString("file"),
		c.GlobalString("rancher-file"))
	if err != nil {
		return nil, err
	}*/

	// TODO
	/*qLookup, err := rLookup.NewQuestionLookup(rancherComposeFile, &lookup.OsEnvLookup{})
	if err != nil {
		return nil, err
	}

	envLookup, err := rLookup.NewFileEnvLookup(c.GlobalString("env-file"), qLookup)
	if err != nil {
		return nil, err
	}*/

	context := &rancher.Context{
		Context: project.Context{
			ResourceLookup: &rLookup.FileResourceLookup{},
			//EnvironmentLookup: envLookup,
			LoggerFactory: logger.NewColorLoggerFactory(),
		},
		RancherComposeFile: c.GlobalString("rancher-file"),
		Url:                c.GlobalString("url"),
		AccessKey:          c.GlobalString("access-key"),
		SecretKey:          c.GlobalString("secret-key"),
		PullCached:         c.Bool("cached"),
		Uploader:           &rancher.S3Uploader{},
		Args:               c.Args(),
		BindingsFile:       c.GlobalString("bindings-file"),
	}
	//qLookup.Context = context

	Populate(&context.Context, c)

	context.Upgrade = c.Bool("upgrade") || c.Bool("force-upgrade")
	context.ForceUpgrade = c.Bool("force-upgrade")
	context.Rollback = c.Bool("rollback")
	context.BatchSize = int64(c.Int("batch-size"))
	context.Interval = int64(c.Int("interval"))
	context.ConfirmUpgrade = c.Bool("confirm-upgrade")
	context.Pull = c.Bool("pull")

	return rancher.NewProject(context)
}

func Populate(context *project.Context, c *cli.Context) {
	// urfave/cli does not distinguish whether the first string in the slice comes from the envvar
	// or is from a flag. Worse off, it appends the flag values to the envvar value instead of
	// overriding it. To ensure the multifile envvar case is always handled, the first string
	// must always be split. It gives a more consistent behavior, then, to split each string in
	// the slice.
	for _, v := range c.GlobalStringSlice("file") {
		context.ComposeFiles = append(context.ComposeFiles, strings.Split(v, string(os.PathListSeparator))...)
	}

	if len(context.ComposeFiles) == 0 {
		context.ComposeFiles = []string{"docker-compose.yml"}
		if _, err := os.Stat("docker-compose.override.yml"); err == nil {
			context.ComposeFiles = append(context.ComposeFiles, "docker-compose.override.yml")
		}
	}

	context.ProjectName = c.GlobalString("project-name")
}

type ProjectAction func(project project.APIProject, c *cli.Context) error

func WithProject(factory *ProjectFactory, action ProjectAction) func(context *cli.Context) error {
	return func(context *cli.Context) error {
		p, err := factory.Create(context)
		if err != nil {
			logrus.Fatalf("Failed to read project: %v", err)
		}
		return action(p, context)
	}
}

func UpCommand(factory *ProjectFactory) cli.Command {
	return cli.Command{
		Name:   "up",
		Usage:  "Bring all services up",
		Action: WithProject(factory, ProjectUp),
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "pull, p",
				Usage: "Before doing the upgrade do an image pull on all hosts that have the image already",
			},
			cli.BoolFlag{
				Name:  "d",
				Usage: "Do not block and log",
			},
			cli.BoolFlag{
				Name:  "upgrade, u, recreate",
				Usage: "Upgrade if service has changed",
			},
			cli.BoolFlag{
				Name:  "force-upgrade, force-recreate",
				Usage: "Upgrade regardless if service has changed",
			},
			cli.BoolFlag{
				Name:  "confirm-upgrade, c",
				Usage: "Confirm that the upgrade was success and delete old containers",
			},
			cli.BoolFlag{
				Name:  "rollback, r",
				Usage: "Rollback to the previous deployed version",
			},
			cli.IntFlag{
				Name:  "batch-size",
				Usage: "Number of containers to upgrade at once",
				Value: 2,
			},
			cli.IntFlag{
				Name:  "interval",
				Usage: "Update interval in milliseconds",
				Value: 1000,
			},
		},
	}
}

func CreateCommand(factory *ProjectFactory) cli.Command {
	return cli.Command{
		Name:   "create",
		Usage:  "Create all services but do not start",
		Action: WithProject(factory, ProjectCreate),
	}
}

func ProjectCreate(p project.APIProject, c *cli.Context) error {
	if err := p.Create(context.Background(), options.Create{}, c.Args()...); err != nil {
		return err
	}

	// This is to fix circular links... What!? It works.
	if err := p.Create(context.Background(), options.Create{}, c.Args()...); err != nil {
		return err
	}

	return nil
}

func ProjectUp(p project.APIProject, c *cli.Context) error {
	if err := p.Create(context.Background(), options.Create{}, c.Args()...); err != nil {
		return err
	}

	if err := p.Up(context.Background(), options.Up{}, c.Args()...); err != nil {
		return err
	}

	if !c.Bool("d") {
		p.Log(context.Background(), true)
		// wait forever
		<-make(chan interface{})
	}

	return nil
}
