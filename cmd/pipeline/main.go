package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/flf2ko/fasthttp-prometheus"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/pipeline/pkg/models"
	"github.com/pipeline/pkg/service"
)

var (
	serviceVersion = "dev"
	methodError    = []string{"method", "error"}
	scope5Min      = "5min"
	scope30Min     = "30min"
	scope240Min    = "240min"
	startTime      = time.Date(2019, 01, 30, 7, 0, 0, 0, time.UTC)
	endTime        = time.Date(2019, 01, 30, 24, 0, 0, 0, time.UTC)
	startTime31    = time.Date(2019, 01, 31, 7, 0, 0, 0, time.UTC)
	endTime31      = time.Date(2019, 02, 1, 24, 0, 0, 0, time.UTC)
)

type configuration struct {
	Port               string `envconfig:"PORT" required:"true" default:"8080"`
	MaxRequestBodySize int    `envconfig:"MAX_REQUEST_BODY_SIZE" default:"10485760"` // 10 MB

	MetricsNamespace    string `envconfig:"METRICS_NAMESPACE" default:"test"`
	MetricsSubsystem    string `envconfig:"METRICS_SUBSYSTEM" default:"pipeline"`
	MetricsNameCount    string `envconfig:"METRICS_NAME_COUNT" default:"request_count"`
	MetricsNameDuration string `envconfig:"METRICS_NAME_DURATION" default:"request_duration"`
	MetricsHelpCount    string `envconfig:"METRICS_HELP_COUNT" default:"Request count"`
	MetricsHelpDuration string `envconfig:"METRICS_HELP_DURATION" default:"Request duration"`

	WriteTimeout int `envconfig:"WRITE_TIMEOUT" default:"30"`

	URIPathExecute string `envconfig:"URI_PATH_EXECUTE" default:"/execute"`
}

func main() {
	printVersion := flag.Bool("version", false, "print version and exit")
	path := flag.String("file", "", "open file for processing")
	flag.Parse()

	if *printVersion {
		fmt.Println(serviceVersion)
		os.Exit(0)
	}

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	_ = level.Info(logger).Log("msg", "initializing", "version", serviceVersion)

	var cfg configuration
	if err := envconfig.Process("", &cfg); err != nil {
		_ = level.Error(logger).Log("msg", "failed to load configuration", "err", err)
		os.Exit(1)
	}

	file, err := os.Open(*path)
	if err != nil {
		_ = level.Error(logger).Log("msg", "failed to open file", "err", err)
		os.Exit(1)
	}
	defer file.Close()

	r := csv.NewReader(file)

	freeFlowJobs := []models.Job{
		models.Job(func(in, out chan interface{}) {
			for {
				record, err := r.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					_ = level.Error(logger).Log("msg", "failed to read file", "err", err)
					os.Exit(1)
				}
				str := strings.Replace(record[3], " ", "T", 1)
				i := strings.Index(str, ".")
				str = strings.Replace(str, ".", "Z", 1)
				str = str[:i+1]
				t, _ := time.Parse(time.RFC3339, str)
				price, err := strconv.ParseFloat(record[1], 64)
				if err != nil {
					_ = level.Error(logger).Log("msg", "failed to parse price", "err", err)
					os.Exit(0)
				}
				rec := models.Record{
					Ticker:    record[0],
					Price:     price,
					Timestamp: t,
				}

				if (t.After(startTime) && t.Before(endTime)) || (t.After(startTime31) && t.Before(endTime31)) {
					out <- rec
				}
			}
		}),
		models.Job(func(in, out chan interface{}) {
			var (
				candles5Min       = startTime
				candles30Min      = startTime
				candles240Min     = startTime
				candles5MinInfo   = make(map[string]models.CandleInfo)
				candles30MinInfo  = make(map[string]models.CandleInfo)
				candles240MinInfo = make(map[string]models.CandleInfo)
				wg                sync.WaitGroup
				wg0               sync.WaitGroup
				mu                sync.RWMutex
			)

			for val := range in {
				if rec, ok := val.(models.Record); ok {
					t := rec.Timestamp
					if t.Before(candles5Min.Add(5 * time.Minute)) {
						wg.Add(1)
						go func() {
							defer wg.Done()
							mu.Lock()
							if info, ok := candles5MinInfo[rec.Ticker]; !ok {
								mu.Unlock()
								info.StartingPoint = rec.Price
								info.EndingPoint = rec.Price
								info.MinPrice = rec.Price
								info.MaxPrice = rec.Price
								mu.Lock()
								candles5MinInfo[rec.Ticker] = info
								mu.Unlock()
							} else {
								mu.Unlock()
								if rec.Price > info.MaxPrice {
									info.MaxPrice = rec.Price
								}
								if rec.Price < info.MinPrice {
									info.MinPrice = rec.Price
								}
								info.EndingPoint = rec.Price
								mu.Lock()
								candles5MinInfo[rec.Ticker] = info
								mu.Unlock()
							}
						}()
					} else {
						wg.Wait()
						mu.Lock()
						for k, v := range candles5MinInfo {
							v.Scope = scope5Min
							v.Ticker = k
							v.Date = candles5Min
							out <- v
							delete(candles5MinInfo, k)
						}
						mu.Unlock()
						var info models.CandleInfo
						info.StartingPoint = rec.Price
						info.EndingPoint = rec.Price
						info.MinPrice = rec.Price
						info.MaxPrice = rec.Price
						mu.Lock()
						candles5MinInfo[rec.Ticker] = info
						mu.Unlock()
						candles5Min = candles5Min.Add(5 * time.Minute)
						if candles5Min == endTime {
							candles5Min = startTime31
						}
						if t.Day() > candles5Min.Day() {
							candles5Min = startTime31
						}
					}

					if t.Before(candles30Min.Add(30 * time.Minute)) {
						wg.Add(1)
						go func() {
							defer wg.Done()
							mu.Lock()
							if info, ok := candles30MinInfo[rec.Ticker]; !ok {
								mu.Unlock()
								info.StartingPoint = rec.Price
								info.EndingPoint = rec.Price
								info.MinPrice = rec.Price
								info.MaxPrice = rec.Price
								mu.Lock()
								candles30MinInfo[rec.Ticker] = info
								mu.Unlock()
							} else {
								mu.Unlock()
								if rec.Price > info.MaxPrice {
									info.MaxPrice = rec.Price
								}
								if rec.Price < info.MinPrice {
									info.MinPrice = rec.Price
								}
								info.EndingPoint = rec.Price
								mu.Lock()
								candles30MinInfo[rec.Ticker] = info
								mu.Unlock()
							}
						}()
					} else {
						wg.Wait()
						mu.Lock()
						for k, v := range candles30MinInfo {
							v.Scope = scope30Min
							v.Ticker = k
							v.Date = candles30Min
							out <- v
							delete(candles30MinInfo, k)
						}
						mu.Unlock()
						var info models.CandleInfo
						info.StartingPoint = rec.Price
						info.EndingPoint = rec.Price
						info.MinPrice = rec.Price
						info.MaxPrice = rec.Price
						mu.Lock()
						candles30MinInfo[rec.Ticker] = info
						mu.Unlock()
						candles30Min = candles30Min.Add(30 * time.Minute)
						if candles30Min == endTime {
							candles30Min = startTime31
						}
						if t.Day() > candles30Min.Day() {
							candles30Min = startTime31
						}
					}

					if t.Before(candles240Min.Add(240 * time.Minute)) {
						wg.Add(1)
						go func() {
							defer wg.Done()
							mu.Lock()
							if info, ok := candles240MinInfo[rec.Ticker]; !ok {
								mu.Unlock()
								info.StartingPoint = rec.Price
								info.EndingPoint = rec.Price
								info.MinPrice = rec.Price
								info.MaxPrice = rec.Price
								mu.Lock()
								candles240MinInfo[rec.Ticker] = info
								mu.Unlock()
							} else {
								mu.Unlock()
								if rec.Price > info.MaxPrice {
									info.MaxPrice = rec.Price
								}
								if rec.Price < info.MinPrice {
									info.MinPrice = rec.Price
								}
								info.EndingPoint = rec.Price
								mu.Lock()
								candles240MinInfo[rec.Ticker] = info
								mu.Unlock()
							}
						}()
					} else {
						wg.Wait()
						mu.Lock()
						for k, v := range candles240MinInfo {
							v.Scope = scope240Min
							v.Ticker = k
							v.Date = candles240Min
							out <- v
							delete(candles240MinInfo, k)
						}
						mu.Unlock()
						candles240Min = candles240Min.Add(240 * time.Minute)
						if candles240Min == endTime {
							candles240Min = startTime31
						}
						if t.Day() > candles240Min.Day() {
							candles240Min = startTime31
						}
					}
				}
			}
			wg0.Add(1)
			go func() {
				defer wg0.Done()
				mu.Lock()
				for k, v := range candles5MinInfo {
					v.Scope = scope5Min
					v.Ticker = k
					v.Date = candles5Min
					out <- v
					delete(candles5MinInfo, k)
				}
				mu.Unlock()
			}()
			wg0.Add(1)
			go func() {
				defer wg0.Done()
				mu.Lock()
				for k, v := range candles30MinInfo {
					v.Scope = scope30Min
					v.Ticker = k
					v.Date = candles30Min
					out <- v
					delete(candles30MinInfo, k)
				}
				mu.Unlock()
			}()
			wg0.Add(1)
			go func() {
				defer wg0.Done()
				mu.Lock()
				for k, v := range candles240MinInfo {
					v.Scope = scope240Min
					v.Ticker = k
					v.Date = candles240Min
					out <- v
					delete(candles240MinInfo, k)
				}
				mu.Unlock()
			}()
			wg0.Wait()
		}),
		models.Job(func(in, out chan interface{}) {
			var (
				wg sync.WaitGroup
				mu sync.RWMutex
			)

			file5Min, err := os.Create("candles_5m.csv")
			if err != nil {
				_ = level.Error(logger).Log("msg", "failed to create file", "err", err)
				os.Exit(0)
			}
			defer file5Min.Close()

			file30Min, err := os.Create("candles_30m.csv")
			if err != nil {
				_ = level.Error(logger).Log("msg", "failed to create file", "err", err)
				os.Exit(0)
			}
			defer file30Min.Close()

			file240Min, err := os.Create("candles_240m.csv")
			if err != nil {
				_ = level.Error(logger).Log("msg", "failed to create file", "err", err)
				os.Exit(0)
			}
			defer file240Min.Close()
			w5Min := csv.NewWriter(file5Min)
			w30Min := csv.NewWriter(file30Min)
			w240Min := csv.NewWriter(file240Min)
			for val := range in {
				if rec, ok := val.(models.CandleInfo); ok {
					switch rec.Scope {
					case scope5Min:
						wg.Add(1)
						go func() {
							defer wg.Done()
							var record []string
							t := rec.Date.Format(time.RFC3339)
							start := fmt.Sprintf("%.2f", rec.StartingPoint)
							max := fmt.Sprintf("%.2f", rec.MaxPrice)
							min := fmt.Sprintf("%.2f", rec.MinPrice)
							end := fmt.Sprintf("%.2f", rec.EndingPoint)
							record = append(record, rec.Ticker)
							record = append(record, t)
							record = append(record, start)
							record = append(record, max)
							record = append(record, min)
							record = append(record, end)
							mu.Lock()
							err := w5Min.Write(record)
							if err != nil {
								_ = level.Error(logger).Log("msg", "failed to write to w5Min file", "err", err)
							}
							w5Min.Flush()
							mu.Unlock()
						}()
					case scope30Min:
						wg.Add(1)
						go func() {
							defer wg.Done()
							var record []string
							t := rec.Date.Format(time.RFC3339)
							start := fmt.Sprintf("%.2f", rec.StartingPoint)
							max := fmt.Sprintf("%.2f", rec.MaxPrice)
							min := fmt.Sprintf("%.2f", rec.MinPrice)
							end := fmt.Sprintf("%.2f", rec.EndingPoint)
							record = append(record, rec.Ticker)
							record = append(record, t)
							record = append(record, start)
							record = append(record, max)
							record = append(record, min)
							record = append(record, end)
							mu.Lock()
							err := w30Min.Write(record)
							if err != nil {
								_ = level.Error(logger).Log("msg", "failed to write to w30Min file", "err", err)
							}
							w30Min.Flush()
							mu.Unlock()
						}()
					case scope240Min:
						wg.Add(1)
						go func() {
							defer wg.Done()
							var record []string
							t := rec.Date.Format(time.RFC3339)
							start := fmt.Sprintf("%.2f", rec.StartingPoint)
							max := fmt.Sprintf("%.2f", rec.MaxPrice)
							min := fmt.Sprintf("%.2f", rec.MinPrice)
							end := fmt.Sprintf("%.2f", rec.EndingPoint)
							record = append(record, rec.Ticker)
							record = append(record, t)
							record = append(record, start)
							record = append(record, max)
							record = append(record, min)
							record = append(record, end)
							mu.Lock()
							err := w240Min.Write(record)
							if err != nil {
								_ = level.Error(logger).Log("msg", "failed to write to w240Min file", "err", err)
							}
							w240Min.Flush()
							mu.Unlock()
						}()
					}
				}
			}
			wg.Wait()
		}),
	}

	svc := service.NewService(freeFlowJobs)

	svc = service.NewLoggingMiddleware(logger, svc)
	svc = service.NewInstrumentingMiddleware(
		kitprometheus.NewCounterFrom(prometheus.CounterOpts{
			Namespace: cfg.MetricsNamespace,
			Subsystem: cfg.MetricsSubsystem,
			Name:      cfg.MetricsNameCount,
			Help:      cfg.MetricsHelpCount,
		}, methodError),
		kitprometheus.NewSummaryFrom(prometheus.SummaryOpts{
			Namespace: cfg.MetricsNamespace,
			Subsystem: cfg.MetricsSubsystem,
			Name:      cfg.MetricsNameDuration,
			Help:      cfg.MetricsHelpDuration,
		}, methodError),
		svc,
	)

	errorProcessor := service.NewErrorProcessor(http.StatusInternalServerError, "internal error")
	executeTransport := service.NewExecuteTransport(service.NewError)

	router := service.MakeFastHTTPRouter(
		[]*service.HandlerSettings{
			{
				Path:    cfg.URIPathExecute,
				Method:  http.MethodPost,
				Handler: service.NewExecuteServer(executeTransport, svc, errorProcessor),
			},
		})

	router.Handle("GET", "/debug/pprof/", fasthttpadaptor.NewFastHTTPHandlerFunc(pprof.Index))
	router.Handle("GET", "/debug/pprof/profile", fasthttpadaptor.NewFastHTTPHandlerFunc(pprof.Profile))

	p := fasthttpprometheus.NewPrometheus(cfg.MetricsSubsystem)
	fasthttpServer := &fasthttp.Server{
		Handler:            p.WrapHandler(router),
		MaxRequestBodySize: cfg.MaxRequestBodySize,
	}

	go func() {
		fasthttpServer.ReadTimeout = time.Second * 1
		_ = level.Info(logger).Log("msg", "starting http server", "port", cfg.Port)
		if err := fasthttpServer.ListenAndServe(":" + cfg.Port); err != nil {
			_ = level.Error(logger).Log("msg", "server run failure", "err", err)
			os.Exit(1)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	defer func(sig os.Signal) {
		_ = level.Info(logger).Log("msg", "received signal, exiting", "signal", sig)
		if err := fasthttpServer.Shutdown(); err != nil {
			_ = level.Error(logger).Log("msg", "server shutdown failure", "err", err)
		}

		_ = level.Info(logger).Log("msg", "goodbye")
	}(<-c)
}
