package qgstats

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	qg "github.com/achiku/qg"
	pgx "github.com/jackc/pgx"
	pgx_stdlib "github.com/jackc/pgx/stdlib"
	mp "github.com/mackerelio/go-mackerel-plugin-helper"
	"github.com/mackerelio/golib/logging"
)

var logger = logging.GetLogger("metrics.plugin.qg")

// QgPlugin mackerel plugin for PostgreSQL
type QgPlugin struct {
	Host        string
	Port        string
	User        string
	Password    string
	Database    string
	SSLMode     string
	SSLKey      string
	SSLCert     string
	SSLRootCert string
	Timeout     int
	Queue       string
	Type        string
	Prefix      string
	Tempfile    string

	db *sql.DB
}

// MetricKeyPrefix returns the metrics key prefix
func (p QgPlugin) MetricKeyPrefix() string {
	if p.Prefix == "" {
		p.Prefix = "qg"
	}
	return p.Prefix
}

// gatherStats filters and aggregates the statistics information
func (q QgPlugin) aggregateStats(stats []qg.JobStats) (retval qg.JobStats) {
	for _, stat := range stats {
		if q.Queue != "" && stat.Queue != q.Queue {
			continue
		}
		if q.Type != "" && stat.Type != q.Type {
			continue
		}
		retval.Count += stat.Count
		retval.CountWorking += stat.CountWorking
		retval.CountErrored += stat.CountErrored
		if retval.HighestErrorCount < stat.HighestErrorCount {
			retval.HighestErrorCount = stat.HighestErrorCount
		}
		if !stat.OldestRunAt.IsZero() && stat.OldestRunAt.Before(retval.OldestRunAt) {
			retval.OldestRunAt = stat.OldestRunAt
		}
	}
	return
}

func (q *QgPlugin) DB() (*sql.DB, error) {
	if q.db != nil {
		return q.db, nil
	}
	var dsnStr []string
	if q.SSLMode != "" {
		dsnStr = append(dsnStr, fmt.Sprintf("sslmode=%s", q.SSLMode))
	}
	if q.SSLKey != "" {
		dsnStr = append(dsnStr, fmt.Sprintf("sslkey=%s", q.SSLKey))
	}
	if q.SSLCert != "" {
		dsnStr = append(dsnStr, fmt.Sprintf("sslcert=%s", q.SSLCert))
	}
	if q.SSLRootCert != "" {
		dsnStr = append(dsnStr, fmt.Sprintf("sslrootcert=%s", q.SSLRootCert))
	}
	if q.Timeout != 0 {
		dsnStr = append(dsnStr, fmt.Sprintf("connect_timeout=%d", q.Timeout))
	}
	config, err := pgx.ParseDSN(strings.Join(dsnStr, " "))
	if err != nil {
		logger.Errorf("FetchMetrics: %s", err)
		return nil, err
	}
	if q.Host != "" {
		config.Host = q.Host
	}
	if q.Port != "" {
		port, err := strconv.Atoi(q.Port)
		config.Port = uint16(port)
		if err != nil {
			return nil, fmt.Errorf("invalid port number: %s", q.Port)
		}
	}
	if q.Database != "" {
		config.Database = q.Database
	}
	if q.User != "" {
		config.User = q.User
	}
	if q.Password != "" {
		config.Password = q.Password
	}

	driverConfig := &pgx_stdlib.DriverConfig{
		ConnConfig:   config,
		AfterConnect: qg.PrepareStatements,
	}
	pgx_stdlib.RegisterDriverConfig(driverConfig)
	q.db, err = sql.Open("pgx", driverConfig.ConnectionString("postgres:///"))
	return q.db, err
}

// FetchMetrics interface for mackerelplugin
func (q QgPlugin) FetchMetrics() (map[string]interface{}, error) {
	db, err := q.DB()
	if err != nil {
		return nil, err
	}
	c, err := qg.NewClient2(db)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	stats, err := c.Stats()
	if err != nil {
		return nil, err
	}

	stat := q.aggregateStats(stats)

	return map[string]interface{}{
		"count_total":         stat.Count,
		"count_working":       stat.CountWorking,
		"count_errored":       stat.CountErrored,
		"highest_error_count": stat.HighestErrorCount,
	}, nil
}

// GraphDefinition interface for mackerelplugin
func (p QgPlugin) GraphDefinition() map[string]mp.Graphs {
	labelPrefix := strings.Title(p.MetricKeyPrefix())

	var graphdef = map[string]mp.Graphs{
		"jobs": {
			Label: (labelPrefix + " Jobs"),
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "count_total", Label: "Total jobs", Diff: false, Stacked: false},
				{Name: "count_working", Label: "Jobs being processed", Diff: false, Stacked: false},
			},
		},
		"errors": {
			Label: (labelPrefix + " Errors"),
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "count_errored", Label: "Job failure count", Diff: false, Stacked: false},
				{Name: "highest_error_count", Label: "Highest error count", Diff: false, Stacked: false},
			},
		},
	}

	return graphdef
}

// Do the plugin
func Do() {
	optHost := flag.String("pghost", os.Getenv("PGHOST"), "Hostname to login to")
	optPort := flag.String("pgport", os.Getenv("PGPORT"), "Database port")
	optUser := flag.String("pguser", os.Getenv("PGUSER"), "Postgres User")
	optDatabase := flag.String("pgdatabase", os.Getenv("PGDATABASE"), "Database name")
	optPassword := flag.String("pgpassword", os.Getenv("PGPASSWORD"), "Postgres Password")
	optSSLMode := flag.String("pgsslmode", os.Getenv("PGSSLMODE"), "Whether to use SSL [disable|allow|prefer|require|verify-ca]")
	optSSLKey := flag.String("pgsslkey", os.Getenv("PGSSLMODE"), "Private key for client certificate")
	optSSLCert := flag.String("pgsslcert", os.Getenv("PGSSLCERT"), "Client certificate")
	optSSLRootCert := flag.String("pgsslrootcert", os.Getenv("PGSSLROOTCERT"), "CA for server certificate")
	optQueue := flag.String("queue", "", "Statistics for specific queue")
	optType := flag.String("type", "", "Statistics for specific job")
	optPrefix := flag.String("metric-key-prefix", "qg", "Metric key prefix")
	optConnectTimeout := flag.Int("connect_timeout", 5, "Maximum wait for connection, in seconds.")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	var qg QgPlugin
	qg.Host = *optHost
	qg.Port = *optPort
	qg.User = *optUser
	qg.Password = *optPassword
	qg.Database = *optDatabase
	qg.SSLMode = *optSSLMode
	qg.SSLKey = *optSSLKey
	qg.SSLCert = *optSSLCert
	qg.SSLRootCert = *optSSLRootCert
	qg.Timeout = *optConnectTimeout

	qg.Queue = *optQueue
	qg.Type = *optType
	qg.Prefix = *optPrefix

	helper := mp.NewMackerelPlugin(qg)

	helper.Tempfile = *optTempfile
	helper.Run()
}
