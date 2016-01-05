package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/mesos-utility/docker-metrics/falcon"
	"github.com/mesos-utility/docker-metrics/g"
	"github.com/mesos-utility/docker-metrics/metric"
)

func handleVersion(displayVersion bool) {
	if displayVersion {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}
}

func main() {
	var dockerAddr string
	var transferAddr string
	//var containerMetricMap = make(map[string]metric.Metric)
	//var certDir string

	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("v", false, "show version")
	flag.Parse()

	handleVersion(*version)

	// global config
	g.ParseConfig(*cfg)

	metric.InitContainerMetricMap()

	dockerAddr = g.Config().Daemon.Addr
	transferAddr = g.Config().Transfer.Addr

	//cert := fmt.Sprintf("%s/cert.pem", certDir)
	//key := fmt.Sprintf("%s/key.pem", certDir)
	//ca := fmt.Sprintf("%s/ca.pem", certDir)
	//dockerclient, err := docker.NewTLSClient(dockerAddr, cert, key, ca)
	dockerclient, err := docker.NewClient(dockerAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	metric.SetGlobalSetting(dockerclient, 2, 3, "vnbe", "eth0")
	client := falcon.CreateFalconClient(transferAddr, 5*time.Millisecond)

	if containers, err := dockerclient.ListContainers(docker.ListContainersOptions{All: false}); err != nil {
		fmt.Println("get container error: ", err)
		os.Exit(1)
	} else {
		for _, container := range containers {
			//fmt.Println("ID: ", container.ID)
			hostname, _ := g.Hostname()
			shortID := container.ID[:g.IDLEN]
			tag := fmt.Sprintf("app=yks-web,id=%s", shortID)
			m := metric.CreateMetric(time.Duration(g.Interval)*time.Second, client, tag, hostname)
			metric.AddContainerMetric(container.ID, m)
		}
	}
	//fmt.Printf("%v\n", containerMetricMap)

	for key, value := range metric.ContainerMetricMap() {
		if c, err := dockerclient.InspectContainer(key); err != nil {
			fmt.Println(key, err)
			continue
		} else {
			go watcher(value, c.ID, c.State.Pid)
		}
	}

	for {
	REST:
		time.Sleep(time.Duration(g.Interval/2) * time.Second)
		if containers, err := dockerclient.ListContainers(docker.ListContainersOptions{All: false}); err != nil {
			fmt.Println("get container error: ", err)
			goto REST
		} else {
			for _, container := range containers {
				//fmt.Println(container.ID)
				if _, ok := metric.ContainerMetricMap()[container.ID]; ok {
					continue
				} else {
					fmt.Println("Add ID: ", container.ID)
					hostname, _ := g.Hostname()
					shortID := container.ID[:g.IDLEN]
					tag := fmt.Sprintf("app=yks-web,id=%s", shortID)
					//m := metric.CreateMetric(time.Duration(g.Interval)*time.Second, client, tag, fmt.Sprintf("%s_%s", hostname, shortID))
					m := metric.CreateMetric(time.Duration(g.Interval)*time.Second, client, tag, hostname)
					metric.AddContainerMetric(container.ID, m)
				}
			}
		}
	}
}

func watcher(serv metric.Metric, cid string, pid int) {
	defer serv.Client.Close()
	if err := serv.InitMetric(cid, pid); err != nil {
		fmt.Println("failed", err)
		return
	}

	t := time.NewTicker(serv.Step)
	defer t.Stop()
	fmt.Println("begin watch", cid)
	for {
		select {
		case now := <-t.C:
			go func() {
				if info, err := serv.UpdateStats(cid); err == nil {
					//fmt.Println(info)
					rate := serv.CalcRate(info, now)
					serv.SaveLast(info)
					// for safe
					//fmt.Println(rate)
					go serv.Send(rate)
				}
			}()
		case <-serv.Stop:
			return
		}
	}
}
