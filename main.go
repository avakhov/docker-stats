package main

import (
	"flag"
	"fmt"
	"github.com/avakhov/docker-stats/stats"
	"github.com/avakhov/docker-stats/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
)

var upMetric = prometheus.NewDesc("docker_up", "is container up", append(stats.LABELS, "id"), nil)
var memUsedMetric = prometheus.NewDesc("docker_mem_used", "memory used", append(stats.LABELS, "id"), nil)
var memTotalMetric = prometheus.NewDesc("docker_mem_total", "memory total", append(stats.LABELS, "id"), nil)
var cpuUsedMetric = prometheus.NewDesc("docker_cpu_used", "cpu used", append(stats.LABELS, "id"), nil)

type Exporter struct {
	stats *stats.Stats
}

func NewExporter(stats *stats.Stats) *Exporter {
	return &Exporter{stats: stats}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- upMetric
	ch <- memUsedMetric
	ch <- memTotalMetric
	ch <- cpuUsedMetric
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	for _, c := range e.stats.GetContainers() {
		id := c.ID[:8]
		ch <- prometheus.MustNewConstMetric(upMetric, prometheus.GaugeValue, float64(c.Up), append(c.Labels, id)...)
		ch <- prometheus.MustNewConstMetric(memUsedMetric, prometheus.GaugeValue, float64(c.MemUsed), append(c.Labels, id)...)
		ch <- prometheus.MustNewConstMetric(memTotalMetric, prometheus.GaugeValue, float64(c.MemTotal), append(c.Labels, id)...)
		ch <- prometheus.MustNewConstMetric(cpuUsedMetric, prometheus.GaugeValue, c.CpuUsed, append(c.Labels, id)...)
	}
}

func doMain(args []string) error {
	// parse args
	flagSet := flag.NewFlagSet("run", flag.ContinueOnError)
	var host string
	var showHelp bool
	flagSet.StringVar(&host, "host", "127.0.0.1", "bind to host")
	flagSet.BoolVar(&showHelp, "help", false, "print help")
	err := flagSet.Parse(args)
	if err != nil {
		return util.WrapError(err)
	}
	if showHelp {
		fmt.Printf("Usage: docker-stats [options]\n")
		fmt.Printf("Options:\n")
		flagSet.PrintDefaults()
		return nil
	}

	// run stats grabber
	stats := stats.NewStats()
	go stats.Run()

	// run expoter
	exporter := NewExporter(stats)
	prometheus.MustRegister(exporter)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		htmlBody := "<a href='/metrics'>metrics</a>"
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(htmlBody))
		if err != nil {
			fmt.Println("Error writing response:", err)
		}
	})
	http.Handle("/metrics", promhttp.Handler())
	fmt.Printf("Listening on %s:3130\n", host)
	err = http.ListenAndServe(host+":3130", nil)
	if err != nil {
		return util.WrapError(err)
	}
	return nil
}

func main() {
	fmt.Printf("docker-stats version=%s\n", util.GetVersion())
	err := doMain(os.Args[1:])
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Printf("done\n")
}
