package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jacobsa/go-serial/serial"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/urfave/cli.v1"
)

var (
	tickTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "power_meter_tick_time",
		Help: "Time in milliseconds between two ticks ",
	})
	tickCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "power_meter_ticks_total",
		Help: "Number of ticks",
	})
	registry = prometheus.NewRegistry()
)

func init() {
	// Register the summary and the histogram with Prometheus's default registry.
	registry.MustRegister(tickTime)
	registry.MustRegister(tickCounter)
}

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
		go run(c)
		return httpServer(c)
	}

	app.Run(os.Args)
}

func httpServer(c *cli.Context) error {
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	log.Printf("Serving /metrics on %s", c.GlobalString("listen-address"))
	return http.ListenAndServe(c.GlobalString("listen-address"), nil)
}

func run(c *cli.Context) error {
	options := serial.OpenOptions{
		PortName:        c.GlobalString("serial-port"),
		BaudRate:        57600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 1,
	}
	// Open the port.
	log.Printf("Opening %s", c.GlobalString("serial-port"))
	port, err := serial.Open(options)
	if err != nil {
		return fmt.Errorf("serial.Open: %v", err)
	}
	scanner := bufio.NewScanner(port)
	for scanner.Scan() {
		vals := strings.Split(scanner.Text(), "\t")
		millisSinceLastTick, err := strconv.ParseFloat(vals[0], 64)
		if err != nil {
			log.Printf("Error parsing float: %s", err)
		} else {
			tickTime.Set(millisSinceLastTick)
		}
		tickCounter.Inc()
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	return nil

}
