package main

import (
	"context"
	"log"
	"log/slog"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ericvolp12/atproto.tools/pkg/plc"
	"github.com/ericvolp12/atproto.tools/pkg/plc/handlers"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.App{
		Name:    "plc-exporter",
		Usage:   "plc exporter",
		Version: "0.0.1",
	}

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "enable debug logging",
			EnvVars: []string{"PLC_EXPORTER_DEBUG"},
		},
		&cli.StringFlag{
			Name:    "listen-addr",
			Usage:   "listen address for http server",
			EnvVars: []string{"PLC_EXPORTER_LISTEN_ADDR"},
			Value:   ":3260",
		},
		&cli.StringFlag{
			Name:    "metrics-listen-addr",
			Usage:   "listen address for http server",
			EnvVars: []string{"PLC_EXPORTER_METRICS_LISTEN_ADDR"},
			Value:   ":3261",
		},
		&cli.StringFlag{
			Name:    "plc-host",
			Usage:   "Host of the PLC Directory",
			EnvVars: []string{"ATP_PLC_HOST"},
			Value:   "https://plc.directory",
		},
		&cli.StringFlag{
			Name:    "data-dir",
			Usage:   "path to data directory",
			EnvVars: []string{"PLC_EXPORTER_DATA_DIR"},
			Value:   "./data/plc-exporter",
		},
		&cli.DurationFlag{
			Name:    "check-interval",
			Usage:   "interval to check for new data",
			EnvVars: []string{"PLC_EXPORTER_CHECK_INTERVAL"},
			Value:   5 * time.Second,
		},
	}

	app.Action = PLCExporter

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func PLCExporter(cctx *cli.Context) error {
	ctx := cctx.Context
	logLevel := slog.LevelInfo
	if cctx.Bool("debug") {
		logLevel = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	})))

	logger := slog.Default()

	// Make sure data directory exists
	dataDir := cctx.String("data-dir")
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		logger.Error("failed to create data directory", "err", err)
		return err
	}

	p, err := plc.NewPLC(ctx, cctx.String("plc-host"), dataDir, logger, cctx.Duration("check-interval"))
	if err != nil {
		logger.Error("failed to create plc", "err", err)
		return err
	}

	go func() {
		err := p.Run(ctx)
		if err != nil {
			logger.Error("failed to run plc", "err", err)
		}
	}()

	h := handlers.NewAPI(p)

	// Create a new echo instance
	e := echo.New()

	// Add Prometheus middleware
	echoProm := echoprometheus.NewMiddlewareWithConfig(echoprometheus.MiddlewareConfig{
		Namespace: "plc_mirror",
		HistogramOptsFunc: func(opts prometheus.HistogramOpts) prometheus.HistogramOpts {
			opts.Buckets = prometheus.ExponentialBuckets(0.00001, 2, 20)
			return opts
		},
	})
	e.Use(echoProm)

	// Add Prometheus metrics handler
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.GET("/:did", h.HandleGetDIDDoc)
	e.GET("/reverse/:handleOrDID", h.HandleReverseSimple)

	// Start the HTTP server
	go func() {
		err := e.Start(cctx.String("listen-addr"))
		if err != nil {
			logger.Error("failed to start http server", "err", err)
		}
	}()

	// Wait for SIGINT or SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Shutdown the HTTP server
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = e.Shutdown(ctx)
	if err != nil {
		logger.Error("failed to shutdown http server", "err", err)
	}

	// Shutdown the PLC
	err = p.Shutdown(ctx)
	if err != nil {
		logger.Error("failed to shutdown plc", "err", err)
	}

	return nil
}
