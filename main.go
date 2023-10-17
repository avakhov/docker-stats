package main

import (
	"flag"
	"fmt"
	"github.com/avakhov/docker-stats/stats"
	"github.com/avakhov/docker-stats/util"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"os"
	"strings"
	"time"
)

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

	// stats
	s := stats.NewStats()
	go s.Run()

	// web server
	e := echo.New()
	f := "method: ${method}, uri: ${uri}, status: ${status}, latency: ${latency_human}\n"
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Format: f}))
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{Timeout: 60 * time.Second}))
	e.GET("/", func(c echo.Context) error {
		out := "<a href='/metrics'>metrics</a>"
		return c.HTML(http.StatusOK, out)
	})
	e.GET("/metrics", func(c echo.Context) error {
		out := []string{}
		containers := s.GetContainers()
		out = append(out, fmt.Sprintf("# HELP docker_up container is up"))
		out = append(out, fmt.Sprintf("# TYPE docker_up counter"))
		for _, c := range containers {
			out = append(out, fmt.Sprintf("docker_up{container=\"%s\"} %d", c.ID, c.Up))
		}
		return c.String(http.StatusOK, strings.Join(out, "\n"))
	})
	err = e.Start(host + ":3130")
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
