package main

import (
	"flag"
	"fmt"
	"github.com/avakhov/docker-stats/ps"
	"github.com/avakhov/docker-stats/stats"
	"github.com/avakhov/docker-stats/util"
	"github.com/avakhov/ext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"strings"
	"time"
)

type Exporter struct {
	startedAt   time.Time
	metrics     map[string]*prometheus.Desc
	dockerStats *stats.Stats
	psStats     *ps.Stats
}

func NewExporter(dockerMetrics bool, dockerLabels []string, psMetrics bool) *Exporter {
	out := Exporter{
		startedAt: time.Now(),
		metrics:   map[string]*prometheus.Desc{},
	}
	if dockerMetrics {
		out.dockerStats = stats.NewStats(dockerLabels)
		go out.dockerStats.Run()
		out.metrics["upMetric"] = prometheus.NewDesc("docker_up", "is container up", append(dockerLabels, "id"), nil)
		out.metrics["memUsedMetric"] = prometheus.NewDesc("docker_mem_used", "memory used", append(dockerLabels, "id"), nil)
		out.metrics["memTotalMetric"] = prometheus.NewDesc("docker_mem_total", "memory total", append(dockerLabels, "id"), nil)
		out.metrics["cpuUsedMetric"] = prometheus.NewDesc("docker_cpu_used", "cpu used", append(dockerLabels, "id"), nil)
	}
	if psMetrics {
		out.psStats = ps.NewStats()
		go out.psStats.Run()
		out.metrics["psAxMetric"] = prometheus.NewDesc("stats_ps_ax", "ps ax count", nil, nil)
		out.metrics["psElMetric"] = prometheus.NewDesc("stats_ps_el", "ps -eL count", nil, nil)
	}
	out.metrics["version"] = prometheus.NewDesc("docker_stats_version", "docker-stats version", []string{"version"}, nil)
	out.metrics["uptime"] = prometheus.NewDesc("docker_stats_uptime", "docker-stats uptime", nil, nil)
	return &out
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range e.metrics {
		ch <- metric
	}
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	if e.dockerStats != nil {
		for _, c := range e.dockerStats.GetContainers() {
			id := c.ID[:8]
			ch <- prometheus.MustNewConstMetric(e.metrics["upMetric"], prometheus.GaugeValue, float64(c.Up), append(c.Labels, id)...)
			ch <- prometheus.MustNewConstMetric(e.metrics["memUsedMetric"], prometheus.GaugeValue, float64(c.MemUsed), append(c.Labels, id)...)
			ch <- prometheus.MustNewConstMetric(e.metrics["memTotalMetric"], prometheus.GaugeValue, float64(c.MemTotal), append(c.Labels, id)...)
			ch <- prometheus.MustNewConstMetric(e.metrics["cpuUsedMetric"], prometheus.GaugeValue, c.CpuUsed, append(c.Labels, id)...)
		}
	}
	if e.psStats != nil {
		ch <- prometheus.MustNewConstMetric(e.metrics["psAxMetric"], prometheus.GaugeValue, float64(e.psStats.GetPsAx()))
		ch <- prometheus.MustNewConstMetric(e.metrics["psElMetric"], prometheus.GaugeValue, float64(e.psStats.GetPsEl()))
	}
	ch <- prometheus.MustNewConstMetric(e.metrics["version"], prometheus.GaugeValue, 1.0, util.GetVersion())
	ch <- prometheus.MustNewConstMetric(e.metrics["uptime"], prometheus.GaugeValue, time.Since(e.startedAt).Seconds())
}

func doMain(args []string) error {
	// parse args
	flagSet := flag.NewFlagSet("run", flag.ContinueOnError)
	var host string
	var showHelp bool
	var labels string
	var dockerMetrics bool
	var psMetrics bool
	flagSet.StringVar(&host, "host", "127.0.0.1", "bind to host")
	flagSet.BoolVar(&showHelp, "help", false, "print help")
	flagSet.StringVar(&labels, "labels", "", "comma separated labels values")
	flagSet.BoolVar(&dockerMetrics, "docker-metrics", false, "enable docker metrics")
	flagSet.BoolVar(&psMetrics, "ps-metrics", false, "enable ps metrics")
	err := flagSet.Parse(args)
	if err != nil {
		return ext.WrapError(err)
	}
	if showHelp {
		fmt.Printf("Usage: docker-stats [options]\n")
		fmt.Printf("Options:\n")
		flagSet.PrintDefaults()
		return nil
	}
	dockerLabels := []string{}
	if labels != "" {
		dockerLabels = strings.Split(labels, ",")
	}

	// run exporter
	exporter := NewExporter(dockerMetrics, dockerLabels, psMetrics)
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
		return ext.WrapError(err)
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
}
