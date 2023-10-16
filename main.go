package main

import (
	"fmt"
	"github.com/avakhov/docker-stats/util"
	"os"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func doMain() error {
  // parse args
	flagSet := flag.NewFlagSet("run", flag.ContinueOnError)
	var host string
	flagSet.StringVar(&host, "host", "127.0.0.1", "bind to host")
	flagSet.BoolVar(&showHelp, "help", false, "print help")
	err = flagSet.Parse(args)
	if err != nil {
		return util.WrapError(err)
	}
	if showHelp {
		fmt.Printf("Usage: tool2 run [options]\n")
		fmt.Printf("Options:\n")
		flagSet.PrintDefaults()
		return nil
	}

  // web server
	e := echo.New()
	f := "method: ${method}, uri: ${uri}, status: ${status}, latency: ${latency_human}\n"
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Format: f}))
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{Timeout: 60 * time.Second}))
	err = e.Start(cmdHost + ":9111")
	if err != nil {
		return util.WrapError(err)
	}
  return nil
}

func main() {
	fmt.Printf("docker-stats version=%s\n", util.GetVersion())
	err := doMain()
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Printf("done\n")
}
