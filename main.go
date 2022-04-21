package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jacobsa/go-serial/serial"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/urfave/cli.v1"
)

var (
	registry = prometheus.NewRegistry()
)

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "serial-port,s",
			Value: "/dev/ttyACM0",
			Usage: "serial port to connect to",
		},
		cli.StringFlag{
			Name:  "listen-address,l",
			Value: ":3000",
			Usage: "",
		},
	}
	app.Name = "electricity-metrics"
	app.Action = func(c *cli.Context) error {
		cl := NewCollector()
		registry.Register(cl)
		go cl.Run(c.GlobalString("serial-port"))
		return httpServer(c)
	}

	app.Run(os.Args)
}

func httpServer(c *cli.Context) error {
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	log.Printf("Serving /metrics on %s", c.GlobalString("listen-address"))
	return http.ListenAndServe(c.GlobalString("listen-address"), nil)
}

func NewCollector() *collector {
	return &collector{
		ticksTimeDesc:    prometheus.NewDesc("power_meter_tick_time", "Time in milliseconds between two ticks", nil, nil),
		ticksCounterDesc: prometheus.NewDesc("power_meter_ticks_total", "Number of ticks", nil, nil),
	}
}

type collector struct {
	ticksCounterDesc *prometheus.Desc
	ticksTimeDesc    *prometheus.Desc

	ticksCounter        int64
	millisSinceLastTick float64
	lastTickTime        time.Time

	mu sync.Mutex
}

func (c *collector) Run(portName string) error {
	options := serial.OpenOptions{
		PortName:        portName,
		BaudRate:        57600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 1,
	}
	// Open the port.
	log.Printf("Opening %s", options.PortName)
	port, err := serial.Open(options)
	if err != nil {
		return fmt.Errorf("serial.Open: %v", err)
	}
	scanner := bufio.NewScanner(port)
	for scanner.Scan() {
		c.processTick(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Println(os.Stderr, "reading standard input: ", err)
	}

	return nil

}
func (c *collector) processTick(data string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastTickTime = time.Now()
	vals := strings.Split(data, "\t")
	millisSinceLastTick, err := strconv.ParseFloat(vals[0], 64)
	if err != nil {
		log.Printf("Error parsing float: %s", err)
	} else {
		c.millisSinceLastTick = millisSinceLastTick
	}
	c.ticksCounter++
	log.Printf("tick time: %v, counter: %s", time.Duration(int64(millisSinceLastTick))*time.Millisecond, vals[1])

}

func (c *collector) Describe(d chan<- *prometheus.Desc) {
	d <- c.ticksCounterDesc
	d <- c.ticksTimeDesc
}

func (c *collector) Collect(m chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.millisSinceLastTick > 0 {
		val := c.millisSinceLastTick
		val = math.Max(val, float64(time.Since(c.lastTickTime).Milliseconds()))
		m <- prometheus.MustNewConstMetric(
			c.ticksTimeDesc,
			prometheus.GaugeValue,
			val,
		)
	}

	m <- prometheus.MustNewConstMetric(
		c.ticksCounterDesc,
		prometheus.CounterValue,
		float64(c.ticksCounter),
	)
}
