package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/metalmatze/signal/healthcheck"
	"github.com/metalmatze/signal/internalserver"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"

	v1 "github.com/leonnicolas/iban-gen/api/v1"
	"github.com/leonnicolas/iban-gen/bic"
	"github.com/leonnicolas/iban-gen/server"
	"github.com/leonnicolas/iban-gen/version"
)

const (
	logLevelAll   = "all"
	logLevelDebug = "debug"
	logLevelInfo  = "info"
	logLevelWarn  = "warn"
	logLevelError = "error"
	logLevelNone  = "none"

	logFmtJson = "json"
	logFmtFmt  = "fmt"

	bundesbankFile = "data/bundesbank.txt"
)

//go:embed data/*
var bankData embed.FS

var (
	availableLogLevels = strings.Join([]string{
		logLevelAll,
		logLevelDebug,
		logLevelInfo,
		logLevelWarn,
		logLevelError,
		logLevelNone,
	}, ", ")

	availableLogFmts = strings.Join([]string{
		logFmtJson,
		logFmtFmt,
	}, ",")
)

// Main is the principal function for the binary, wrapped only by `main` for convenience.
func Main() error {
	listen := flag.String("listen", ":8080", "The address at which to listen.")
	listenInternal := flag.String("listen-internal", ":9090", "The address at which to listen for health and metrics.")
	healthCheckURL := flag.String("healthchecks-url", "http://localhost:8080", "The URL against which to run healthchecks.")
	logLevel := flag.String("log-level", logLevelInfo, fmt.Sprintf("Log level to use. Possible values: %s", availableLogLevels))
	logFmt := flag.String("log-fmt", logFmtFmt, fmt.Sprintf("Log format to use. Possible values: %s", availableLogFmts))
	help := flag.Bool("h", false, "Show usage")
	printVersion := flag.Bool("version", false, "Show version")

	flag.Parse()

	if *help {
		flag.Usage()
		return nil
	}

	if *printVersion {
		fmt.Println(version.Version)
		return nil
	}

	var logger log.Logger
	switch *logFmt {
	case logFmtJson:
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	case logFmtFmt:
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	default:
		return fmt.Errorf("log format %v unknown; possible values are: %s", *logFmt, availableLogFmts)
	}

	switch *logLevel {
	case logLevelAll:
		logger = level.NewFilter(logger, level.AllowAll())
	case logLevelDebug:
		logger = level.NewFilter(logger, level.AllowDebug())
	case logLevelInfo:
		logger = level.NewFilter(logger, level.AllowInfo())
	case logLevelWarn:
		logger = level.NewFilter(logger, level.AllowWarn())
	case logLevelError:
		logger = level.NewFilter(logger, level.AllowError())
	case logLevelNone:
		logger = level.NewFilter(logger, level.AllowNone())
	default:
		return fmt.Errorf("log level %v unknown; possible values are: %s", *logLevel, availableLogLevels)
	}
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)
	stdlog.SetOutput(log.NewStdlibAdapter(logger))

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var g run.Group
	g.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))
	{
		l, err := net.Listen("tcp", *listen)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %v", *listen, err)
		}

		g.Add(func() error {
			level.Info(logger).Log("msg", "starting the openiban HTTP server", "addr", *listen, "version", version.Version)
			r := chi.NewRouter()
			r.Use(func(h http.Handler) http.Handler {
				fn := func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Access-Control-Allow-Origin", "*")
					h.ServeHTTP(w, r)
				}
				return http.HandlerFunc(fn)
			})
			bicsRepo := bic.NewBICRepo()
			f, err := bankData.Open(bundesbankFile)
			if err != nil {
				return err
			}
			defer f.Close()
			i, err := bicsRepo.Populate(f)
			if err != nil {
				return err
			}
			level.Info(logger).Log("msg", "loaded BIC data", "entries", i)
			// level.Debug(logger).Log("msg", "loaded BIC data", "count", i, "entries", bicsRepo.BICs())
			s := server.NewInstrumentedServerWithLogger(
				bicsRepo,
				prometheus.WrapRegistererWith(prometheus.Labels{"api": "v1"}, reg),
				log.With(logger, "component", "http-server"),
			)
			r.Mount("/", v1.Handler(s))
			if err := http.Serve(l, r); err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("error: server exited unexpectedly: %v", err)
			}
			return nil
		}, func(error) {
			l.Close()
		})
	}

	{
		// Run the internal HTTP server.
		healthchecks := healthcheck.NewMetricsHandler(healthcheck.NewHandler(), reg)
		// Checks if the server is up.
		healthchecks.AddLivenessCheck("http",
			healthcheck.HTTPCheckClient(
				http.DefaultClient,
				*healthCheckURL,
				http.MethodGet,
				http.StatusNotFound,
				time.Second,
			),
		)
		h := internalserver.NewHandler(
			internalserver.WithName("Internal - openiban"),
			internalserver.WithHealthchecks(healthchecks),
			internalserver.WithPrometheusRegistry(reg),
			internalserver.WithPProf(),
		)
		l, err := net.Listen("tcp", *listenInternal)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %v", *listenInternal, err)
		}

		g.Add(func() error {
			level.Info(logger).Log("msg", "starting the openiban internal HTTP server", "addr", *listenInternal)

			if err := http.Serve(l, h); err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("error: internal server exited unexpectedly: %v", err)
			}
			return nil
		}, func(error) {
			l.Close()
		})
	}

	return g.Run()
}

func main() {
	if err := Main(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
