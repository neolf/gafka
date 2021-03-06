package command

import (
	"bufio"
	"flag"
	"fmt"
	"strings"

	"github.com/funkygao/gafka/ctx"
	"github.com/funkygao/gafka/zk"
	"github.com/funkygao/gocli"
	"github.com/funkygao/golib/color"
	"github.com/funkygao/golib/pipestream"
	log "github.com/funkygao/log4go"
)

type Partition struct {
	Ui  cli.Ui
	Cmd string
}

func (this *Partition) Run(args []string) (exitCode int) {
	var (
		zone       string
		topic      string
		cluster    string
		partitions int
	)
	cmdFlags := flag.NewFlagSet("partition", flag.ContinueOnError)
	cmdFlags.Usage = func() { this.Ui.Output(this.Help()) }
	cmdFlags.StringVar(&zone, "z", "", "")
	cmdFlags.StringVar(&cluster, "c", "", "")
	cmdFlags.StringVar(&topic, "t", "", "")
	cmdFlags.IntVar(&partitions, "n", 1, "")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if validateArgs(this, this.Ui).
		require("-z", "-c", "-t", "-n").
		requireAdminRights("-z").
		invalid(args) {
		return 2
	}

	zkzone := zk.NewZkZone(zk.DefaultConfig(zone, ctx.ZoneZkAddrs(zone)))
	zkcluster := zkzone.NewCluster(cluster)
	this.addPartition(zkcluster.ZkConnectAddr(), topic, partitions)
	return
}

func (this *Partition) addPartition(zkAddrs string, topic string, partitions int) error {
	log.Info("adding partitions to topic: %s", topic)

	cmd := pipestream.New(fmt.Sprintf("%s/bin/kafka-topics.sh", ctx.KafkaHome()),
		fmt.Sprintf("--zookeeper %s", zkAddrs),
		fmt.Sprintf("--alter"),
		fmt.Sprintf("--topic %s", topic),
		fmt.Sprintf("--partitions %d", partitions),
	)
	err := cmd.Open()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(cmd.Reader())
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		this.Ui.Output(color.Yellow(scanner.Text()))
	}
	err = scanner.Err()
	if err != nil {
		return err
	}
	cmd.Close()

	log.Info("added partitions to topic: %s", topic)
	return nil
}

func (*Partition) Synopsis() string {
	return "Add partition num to a topic for better parallel"
}

func (this *Partition) Help() string {
	help := fmt.Sprintf(`
Usage: %s partition -z zone -c cluster -t topic -n num

    %s
`, this.Cmd, this.Synopsis())
	return strings.TrimSpace(help)
}
