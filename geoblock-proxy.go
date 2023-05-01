package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/mdouchement/geoblock-proxy/proxy"
	"github.com/mdouchement/geoblock/lookup"
	"github.com/mdouchement/logger"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type controller struct {
	cfg       string
	config    Configuration
	ctx       context.Context
	evaluator *Evaluator
	proxy     proxy.Proxy

	allowed  *prometheus.CounterVec
	rejected *prometheus.CounterVec
}

func main() {
	c := controller{
		allowed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "geoblock",
			Subsystem: "",
			Name:      "allowed_total",
			Help:      "Total of allowed requests.",
		}, []string{"country"}),
		rejected: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "geoblock",
			Subsystem: "",
			Name:      "rejected_total",
			Help:      "Total of rejected requests.",
		}, []string{"country"}),
	}

	cmd := &cobra.Command{
		Use:   "geoblock-proxy",
		Short: "Starts the geoblock proxy server",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			if c.cfg == "" {
				c.cfg = "geoblock-proxy.yml"
			}

			logr := logrus.New()
			logr.SetFormatter(&logger.LogrusTextFormatter{
				DisableColors:   false,
				ForceColors:     true,
				ForceFormatting: true,
				PrefixRE:        regexp.MustCompile(`^(\[.*?\])\s`),
				FullTimestamp:   true,
				TimestampFormat: "2006-01-02 15:04:05",
			})
			log := logger.WrapLogrus(logr)
			c.ctx = logger.WithLogger(context.Background(), log)

			//

			{

				log.Infof("Reading configuration from %s", c.cfg)
				payload, err := os.ReadFile(c.cfg)
				if err != nil {
					if err != nil {
						return errors.Wrapf(err, "could not read configuration file %s", c.cfg)
					}
				}

				err = yaml.Unmarshal(payload, &c.config)
				if err != nil {
					if err != nil {
						return errors.Wrapf(err, "could not parse configuration file %s", c.cfg)
					}
				}

				c.evaluator, err = NewEvaluator("evaluator", c.config)
				if err != nil {
					return errors.Wrap(err, "could not create geoblock evaluator")
				}

				if c.config.Logger != "" {
					l, err := logrus.ParseLevel(c.config.Logger)
					if err != nil {
						return errors.Wrapf(err, "could not parse logger level %s", c.cfg)
					}
					logr.SetLevel(l)
				}

				if c.config.Metrics != "" {
					prometheus.Register(c.allowed)  //nolint:errcheck
					prometheus.Register(c.rejected) //nolint:errcheck

					go func() {
						log.Infof("Starting metrics endpoint on %s", c.config.Metrics)

						http.Handle("/metrics", promhttp.Handler())
						err := http.ListenAndServe(c.config.Metrics, nil)
						if err != nil {
							log.WithError(err).Error("Could not run metrics endpoint")
						}
					}()
				}
			}

			if err := c.setup(); err != nil {
				return err
			}

			defer c.close()
			c.proxy.Run()

			return nil
		},
	}
	cmd.Flags().StringVarP(&c.cfg, "config", "c", os.Getenv("GEOBLOCK_PROXY_CONFIG"), "Server's configuration")

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (c *controller) setup() error {
	for _, databasename := range c.config.Databases {
		lookup, err := lookup.OpenIP2location(databasename)
		if err != nil {
			return errors.Wrapf(err, "ip2location: %s", databasename)
		}

		c.evaluator.AddLookup(lookup)
	}

	//

	var err error
	var backend, frontend net.Addr

	switch c.config.Protocol {
	case ProtocolUDP:
		backend, err = net.ResolveUDPAddr("udp", c.config.Backend)
		if err != nil {
			return errors.Wrapf(err, "could not resolve UDP backend: %s", c.config.Backend)
		}

		frontend, err = net.ResolveUDPAddr("udp", c.config.Frontend)
		if err != nil {
			return errors.Wrapf(err, "could not resolve UDP frontend: %s", c.config.Frontend)
		}
	case ProtocolTCP:
		backend, err = net.ResolveTCPAddr("tcp", c.config.Backend)
		if err != nil {
			return errors.Wrapf(err, "could not resolve TCP backend: %s", c.config.Backend)
		}

		frontend, err = net.ResolveTCPAddr("tcp", c.config.Frontend)
		if err != nil {
			return errors.Wrapf(err, "could not resolve TCP frontend: %s", c.config.Frontend)
		}
	}

	c.proxy, err = proxy.NewProxy(c.ctx, frontend, backend, func(ctx context.Context, ip net.IP) bool {
		if ip == nil {
			return false
		}

		log := logger.LogWith(ctx)

		allowed, country, err := c.evaluator.Evaluate(ip.String())
		if err != nil {
			log.Infof("%s - %v", ip, err)
			return false
		}

		if !allowed {
			log.Infof("%s from %s is blocked", ip, strings.ToUpper(country))
			c.rejected.WithLabelValues(country).Inc()
			return false
		}

		c.allowed.WithLabelValues(country).Inc()
		return true
	})

	return errors.Wrap(err, "could not create proxy")
}

func (c *controller) close() {
	if c.proxy != nil {
		c.proxy.Close()
	}
}
