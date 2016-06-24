package kateway

import (
	"sync"
	"time"

	"github.com/funkygao/gafka/cmd/kguard/monitor"
	"github.com/funkygao/gafka/zk"
	log "github.com/funkygao/log4go"
	"github.com/rcrowley/go-metrics"
)

func init() {
	monitor.RegisterWatcher("kateway.kateway", func() monitor.Watcher {
		return &WatchKateway{
			Tick: time.Minute,
		}
	})
}

// WatchKateway monitors aliveness of kateway cluster.
type WatchKateway struct {
	Zkzone *zk.ZkZone
	Stop   <-chan struct{}
	Tick   time.Duration
	Wg     *sync.WaitGroup
}

func (this *WatchKateway) Init(ctx monitor.Context) {
	this.Zkzone = ctx.ZkZone()
	this.Stop = ctx.StopChan()
	this.Wg = ctx.WaitGroup()
}

func (this *WatchKateway) Run() {
	defer this.Wg.Done()

	ticker := time.NewTicker(this.Tick)
	defer ticker.Stop()

	liveKateways := metrics.NewRegisteredGauge("kateway.live", nil)

	// warmup
	kws, _ := this.Zkzone.KatewayInfos()
	liveKateways.Update(int64(len(kws)))

	for {
		select {
		case <-this.Stop:
			log.Info("kateway.kateway stopped")
			return

		case <-ticker.C:
			kws, _ := this.Zkzone.KatewayInfos()
			liveKateways.Update(int64(len(kws)))
		}
	}
}
