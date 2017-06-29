package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/funkygao/gafka/ctx"
	"github.com/funkygao/gafka/zk"
	"github.com/funkygao/gocli"
)

type Plugins struct {
	Ui  cli.Ui
	Cmd string
}

func (this *Plugins) Run(args []string) (exitCode int) {
	var (
		zone    string
		cluster string
	)
	cmdFlags := flag.NewFlagSet("plugins", flag.ContinueOnError)
	cmdFlags.Usage = func() { this.Ui.Output(this.Help()) }
	cmdFlags.StringVar(&zone, "z", ctx.EsDefaultZone(), "")
	cmdFlags.StringVar(&cluster, "c", "", "")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	zkzone := zk.NewZkZone(zk.DefaultConfig(zone, ctx.ZoneZkAddrs(zone)))
	handleCatCommand(this.Ui, zkzone, cluster, "plugins")

	return
}

func (*Plugins) Synopsis() string {
	return "Provides a view per node of running plugins"
}

func (this *Plugins) Help() string {
	help := fmt.Sprintf(`
Usage: %s plugins [options]

    %s

Options:

    -z zone

    -c cluster

`, this.Cmd, this.Synopsis())
	return strings.TrimSpace(help)
}
