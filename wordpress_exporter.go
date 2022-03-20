package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"flag"
	"fmt"
	"os"
	"strconv"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

const (
	namespace = "wordpress"
)

//This is my collector metrics
type wpCollector struct {
	numPostsMetric        *prometheus.GaugeVec
	numCommentsMetric     *prometheus.GaugeVec
	numUsersMetric        *prometheus.Desc
	numCustomersMetric    *prometheus.Desc
	numSessionsMetric     *prometheus.Desc
	numWebhooksMetric     *prometheus.GaugeVec
	numAutoloadMetric     *prometheus.Desc
	numAutoloadSizeMetric *prometheus.Desc
	numDatabaseSizeMetric *prometheus.Desc
	numPostsTypeMetric    *prometheus.GaugeVec
	numOrderTypeMetric    *prometheus.GaugeVec

	dbHost            string
	dbName            string
	dbUser            string
	dbPass            string
	dbTablePrefix     string
	dbSkipWooCommerce bool
}

//This is a constructor for my wpCollector struct
func newWordPressCollector(host string, dbname string, username string, pass string, tablePrefix string, skipWooCommerce bool) *wpCollector {
	return &wpCollector{

		numUsersMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "users_total"),
			"Shows the number of registered users in the WordPress site",
			nil, nil,
		),

		numCustomersMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "customers_total"),
			"Shows the number of customers in the WordPress site",
			nil, nil,
		),

		numCommentsMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "comments_total",
			Help:      "Shows the number of total comments in the WordPress site",
		},
			[]string{"type"},
		),

		numSessionsMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "user_sessions_total"),
			"Shows the number of sessions in the WordPress site",
			nil, nil,
		),

		numPostsMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "posts_total",
			Help:      "Shows the number of total posts in the WordPress site",
		},
			[]string{"type"},
		),

		numWebhooksMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "webhooks_total",
			Help:      "Shows the number of webhooks in the WordPress site",
		},
			[]string{"status"},
		),

		numAutoloadMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "option_autoload_total"),
			"Shows the number of options with autoload",
			nil, nil,
		),

		numAutoloadSizeMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "option_autoload_bytes"),
			"Shows the size in bytes of options with autoload",
			nil, nil,
		),

		numDatabaseSizeMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "database_size_bytes"),
			"Shows the size in bytes of the wordpress's database",
			nil, nil,
		),

		numPostsTypeMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "posts_type_total",
			Help:      "Shows the number of total posts type in the WordPress site",
		},
			[]string{"type"},
		),

		numOrderTypeMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "order_type_total",
			Help:      "Shows the number of total orders type in WooCommerce",
		},
			[]string{"type"},
		),

		dbHost:            host,
		dbName:            dbname,
		dbUser:            username,
		dbPass:            pass,
		dbTablePrefix:     tablePrefix,
		dbSkipWooCommerce: skipWooCommerce,
	}
}

//Describe method is required for a prometheus.Collector type
func (collector *wpCollector) Describe(ch chan<- *prometheus.Desc) {

	//We set the metrics
	ch <- collector.numUsersMetric
	ch <- collector.numCustomersMetric
	ch <- collector.numSessionsMetric
	ch <- collector.numAutoloadMetric
	ch <- collector.numAutoloadSizeMetric
	ch <- collector.numDatabaseSizeMetric
	collector.numOrderTypeMetric.Describe(ch)
	collector.numPostsTypeMetric.Describe(ch)
	collector.numCommentsMetric.Describe(ch)
	collector.numPostsMetric.Describe(ch)
	collector.numWebhooksMetric.Describe(ch)
}

//Collect method is required for a prometheus.Collector type
func (collector *wpCollector) Collect(ch chan<- prometheus.Metric) {

	//We run DB queries here to retrieve the metrics we care about
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", collector.dbUser, collector.dbPass, collector.dbHost, collector.dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %s ...\n", err)
		os.Exit(1)
	}
	defer db.Close()

	//select count(*) as value from wp_users;
	queryNumUsersMetric := fmt.Sprintf("select count(*) as value from %susers;", collector.dbTablePrefix)
	wpQueryGauge(db, ch, collector.numUsersMetric, queryNumUsersMetric)

	//select count(*) as value from wp_users inner join wp_usermeta on wp_users.ID = wp_usermeta.user_id where wp_usermeta.meta_key = 'wp_capabilities' and wp_usermeta.meta_value like '%customer%';
	queryNumCustomersMetric := fmt.Sprintf("select count(*) as value from %susers inner join %susermeta on %susers.ID = %susermeta.user_id where %susermeta.meta_key = 'wp_capabilities' and %susermeta.meta_value like '%%customer%%';", collector.dbTablePrefix, collector.dbTablePrefix, collector.dbTablePrefix, collector.dbTablePrefix, collector.dbTablePrefix, collector.dbTablePrefix)
	wpQueryGauge(db, ch, collector.numCustomersMetric, queryNumCustomersMetric)

	//select COALESCE(NULLIF(comment_type, ''), 'comment') as label, count(*) as value from wp_comments group by comment_type;
	queryNumCommentsMetric := fmt.Sprintf("select COALESCE(NULLIF(comment_type, ''), 'comment') as label, count(*) as value from %scomments group by comment_type;", collector.dbTablePrefix)
	wpQueryGaugeVec(db, ch, collector.numCommentsMetric, queryNumCommentsMetric)

	//select post_status as label, count(*) as value from wp_posts WHERE post_type='post' GROUP BY post_status;
	queryNumPostsMetric := fmt.Sprintf("select post_status as label, count(*) as value from %sposts WHERE post_type='post' GROUP BY post_status;", collector.dbTablePrefix)
	wpQueryGaugeVec(db, ch, collector.numPostsMetric, queryNumPostsMetric)

	if !collector.dbSkipWooCommerce {
		//select count(*) as numSessions from wp_woocommerce_sessions;
		queryNumSessionsMetric := fmt.Sprintf("select count(*) as numSessions from %swoocommerce_sessions;", collector.dbTablePrefix)
		wpQueryGauge(db, ch, collector.numSessionsMetric, queryNumSessionsMetric)
	}

	//select post_status as label, count(*) as value from wp_posts WHERE post_type='scheduled-action' GROUP BY post_status;
	queryNumWebhooksMetric := fmt.Sprintf("select post_status as label, count(*) as value from %sposts WHERE post_type='scheduled-action' GROUP BY post_status;", collector.dbTablePrefix)
	wpQueryGaugeVec(db, ch, collector.numWebhooksMetric, queryNumWebhooksMetric)

	//select count(*) from wp_options where autoload = 'yes';
	queryNumAutoloadMetric := fmt.Sprintf("select count(*) from %soptions where autoload = 'yes';", collector.dbTablePrefix)
	wpQueryGauge(db, ch, collector.numAutoloadMetric, queryNumAutoloadMetric)

	//select ROUND(SUM(LENGTH(option_value))) as value from wp_options where autoload = 'yes';
	queryNumAutoloadSizeMetric := fmt.Sprintf("select ROUND(SUM(LENGTH(option_value))) as value from %soptions where autoload = 'yes';", collector.dbTablePrefix)
	wpQueryGauge(db, ch, collector.numAutoloadSizeMetric, queryNumAutoloadSizeMetric)

	//select count(*) from wp_options where autoload = 'yes';
	queryNumDatabaseSizeMetric := fmt.Sprintf("select ROUND(SUM(data_length+index_length),2) as value from information_schema.tables where table_schema='%s';", collector.dbName)
	wpQueryGauge(db, ch, collector.numDatabaseSizeMetric, queryNumDatabaseSizeMetric)

	//select post_type, count(*) from wp_posts group by post_type;
	queryNumPostsTypeMetric := fmt.Sprintf("select post_type, count(*) from %sposts group by post_type;", collector.dbTablePrefix)
	wpQueryGaugeVec(db, ch, collector.numPostsTypeMetric, queryNumPostsTypeMetric)

	//select SUBSTRING(post_status, 4), count(*) from wp_posts where post_type = 'shop_order' group by post_status;
	queryNumOrderTypeMetric := fmt.Sprintf("select SUBSTRING(post_status, 4), count(*) from %sposts where post_type = 'shop_order' group by post_status;", collector.dbTablePrefix)
	wpQueryGaugeVec(db, ch, collector.numOrderTypeMetric, queryNumOrderTypeMetric)

}

func wpQueryCounter(db *sql.DB, ch chan<- prometheus.Metric, desc *prometheus.Desc, mysqlRequest string) {
	var value float64
	var err = db.QueryRow(mysqlRequest).Scan(&value)
	if err != nil {
		log.Fatal(err)
	}
	ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, value)
}

func wpQueryGauge(db *sql.DB, ch chan<- prometheus.Metric, desc *prometheus.Desc, mysqlRequest string) {
	var value float64
	var err = db.QueryRow(mysqlRequest).Scan(&value)
	if err != nil {
		log.Fatal(err)
	}
	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value)
}

func wpQueryGaugeVec(db *sql.DB, ch chan<- prometheus.Metric, desc *prometheus.GaugeVec, mysqlRequest string) {
	rows, err := db.Query(mysqlRequest)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	desc.Reset()
	for rows.Next() {
		var label string
		var value float64
		err = rows.Scan(&label, &value)
		if err != nil {
			log.Fatal(err)
		}
		desc.WithLabelValues(label).Set(value)
	}
	desc.Collect(ch)
}

func main() {

	wpHostPtr := flag.String("host", "127.0.0.1", "Hostname or Address for DB server")
	wpPortPtr := flag.String("port", "3306", "DB server port")
	wpNamePtr := flag.String("db", "", "DB name")
	wpUserPtr := flag.String("user", "", "DB user for connection")
	wpPassPtr := flag.String("pass", "", "DB password for connection")
	wpTablePrefixPtr := flag.String("tableprefix", "wp_", "Table prefix for WordPress tables")
	wpSkipWooCommerce := flag.String("skipwoocommerce", "false", "Skip WooCommerce metrics")

	flag.Parse()

	dbHost := fmt.Sprintf("%s:%s", *wpHostPtr, *wpPortPtr)
	dbName := *wpNamePtr
	dbUser := *wpUserPtr
	dbPassword := *wpPassPtr
	tablePrefix := *wpTablePrefixPtr
	skipWooCommerce, _ := strconv.ParseBool(*wpSkipWooCommerce)

	if dbName == "" {
		fmt.Fprintf(os.Stderr, "flag -db=dbname required!\n")
		os.Exit(1)
	}

	if dbUser == "" {
		fmt.Fprintf(os.Stderr, "flag -user=username required!\n")
		os.Exit(1)
	}

	//We create the collector
	collector := newWordPressCollector(dbHost, dbName, dbUser, dbPassword, tablePrefix, skipWooCommerce)
	prometheus.MustRegister(collector)

	//This section will start the HTTP server and expose
	//any metrics on the /metrics endpoint.
	http.Handle("/metrics", promhttp.Handler())
	log.Info("Beginning to serve on port :9850")
	log.Fatal(http.ListenAndServe(":9850", nil))
}
