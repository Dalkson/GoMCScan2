package main

import (
	"GoMCScan/mcping"
	"GoMCScan/mcping/types"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/aherve/gopool"
)

type options struct {
	threads int
	timeout int
	addressList []string
	portList []uint16
	outputPath string
}

type statistics struct {
	pinged int
	completed int
	found int
	total int
	startTime time.Time
}

var stats statistics
var conf options

var pool *gopool.GoPool

func main() {
	conf = getFlags()
	stats.total = totalToSend()
	pool = gopool.NewPool(conf.threads)
	fmt.Println("Total to scan:", stats.total)
	go logLoop(2 * time.Second)
	loopBlock()
	pool.Wait()
	fmt.Println("Scan Complete!")
}

func loopBlock() {
	stats.startTime = time.Now()
	for _, port := range conf.portList {
		for _, address := range conf.addressList {
			if !strings.Contains(address, "/") { //puts single addresses in CIDR notation
				address = fmt.Sprintf("%v/32", address)
			}
			ip, ipnet, err := net.ParseCIDR(address)
			if err != nil {
				log.Fatal(err)
			}
			for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
				pool.Add(1)
				go pingIt(string(net.IP.String(ip)), port)
				stats.pinged++
			}
		}
	}
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

type formattedOutput struct {
	Timestamp string
	Ip string
	Version string
	Motd string 
	PlayersCount types.PlayerCount
	Sample []types.PlayerSample
}

func pingIt(ip string, port uint16) {
	defer pool.Done()
	data, _, err := mcping.PingWithTimeout(ip, port, time.Duration(conf.timeout)*time.Second)
	stats.completed++ // this is somewhat broken because of concurency
	if err == nil {
		stats.found++ // also would be broken
		printStatus(fmt.Sprintf("%v:%v | %v  online | %v", ip, port, data.PlayerCount.Online, data.Motd))
		formatted := formattedOutput{time.Now().Format("2006-01-02 15:04:04"), ip+":"+fmt.Sprint(port), data.Version, data.Motd, data.PlayerCount, data.Sample}
		record(formatted)
	}
}
