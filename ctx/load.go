package ctx

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	jsconf "github.com/funkygao/jsconf"
)

func LoadConfig(fn string) {
	cf, err := jsconf.Load(fn)
	if err != nil {
		panic(err)
	}

	conf = new(config)
	conf.hostname, _ = os.Hostname()
	conf.kafkaHome = cf.String("kafka_home", "")
	conf.logLevel = cf.String("loglevel", "info")
	conf.zkDefaultZone = cf.String("zk_default_zone", "")
	conf.esDefaultZone = cf.String("es_default_zone", conf.zkDefaultZone)
	conf.upgradeCenter = cf.String("upgrade_center", "")

	conf.aliases = make(map[string]string)
	for i := 0; i < len(cf.List("aliases", nil)); i++ {
		section, err := cf.Section(fmt.Sprintf("aliases[%d]", i))
		if err != nil {
			panic(err)
		}

		conf.aliases[section.String("cmd", "")] = section.String("alias", "")
	}

	conf.zones = make(map[string]*zone)
	for i := 0; i < len(cf.List("zones", nil)); i++ {
		section, err := cf.Section(fmt.Sprintf("zones[%d]", i))
		if err != nil {
			panic(err)
		}

		z := new(zone)
		z.loadConfig(section)
		conf.zones[z.Name] = z
	}

	conf.reverseDns = make(map[string][]string)
	for _, entry := range cf.StringList("reverse_dns", nil) {
		if entry != "" {
			// entry e,g. k11000b.sit.mycorp.kfk.com:10.10.1.1
			parts := strings.SplitN(entry, ":", 2)
			if len(parts) != 2 {
				panic("invalid reverse_dns record")
			}

			ip, host := strings.TrimSpace(parts[1]), strings.TrimSpace(parts[0])
			if _, present := conf.reverseDns[ip]; !present {
				conf.reverseDns[ip] = make([]string, 0)
			}

			conf.reverseDns[ip] = append(conf.reverseDns[ip], host)
		}
	}

}

func LoadFromHome() {
	var configFile string
	if usr, err := user.Current(); err == nil {
		configFile = filepath.Join(usr.HomeDir, ".gafka.cf")
	} else {
		panic(err)
	}

	_, err := os.Stat(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// create the config file on the fly
			if e := ioutil.WriteFile(configFile,
				[]byte(strings.TrimSpace(DefaultConfig)), 0644); e != nil {
				panic(e)
			}
		} else {
			panic(err)
		}
	}

	LoadConfig(configFile)
}
