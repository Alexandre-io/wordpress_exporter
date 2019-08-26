package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"flag"
	"fmt"
	"os"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

const (
	namespace = "wordpress"
)

//This is my collector metrics
type wpCollector struct {
	numPostsMetric     *prometheus.Desc
	numCommentsMetric  *prometheus.Desc
	numUsersMetric     *prometheus.Desc
	numCustomersMetric *prometheus.Desc
	numSessionsMetric  *prometheus.Desc
	numWebhooksMetric  *prometheus.GaugeVec

	dbHost        string
	dbName        string
	dbUser        string
	dbPass        string
	dbTablePrefix string
}

//This is a constructor for my wpCollector struct
func newWordPressCollector(host string, dbname string, username string, pass string, tablePrefix string) *wpCollector {
	return &wpCollector{
		numPostsMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "num_posts_metric"),
			"Shows the number of total posts in the WordPress site",
			nil, nil,
		),
		numCommentsMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "num_comments_metric"),
			"Shows the number of total comments in the WordPress site",
			nil, nil,
		),
		numUsersMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "num_users_metric"),
			"Shows the number of registered users in the WordPress site",
			nil, nil,
		),

		numCustomersMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "num_customers_metric"),
			"Shows the number of customers in the WordPress site",
			nil, nil,
		),

		numSessionsMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "num_sessions_metric"),
			"Shows the number of sessions in the WordPress site",
			nil, nil,
		),

		numWebhooksMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "webhooks_metric",
			Help:      "Shows the number of webhooks in the WordPress site",
		},
			[]string{"status"},
		),

		dbHost:        host,
		dbName:        dbname,
		dbUser:        username,
		dbPass:        pass,
		dbTablePrefix: tablePrefix,
	}
}

//Describe method is required for a prometheus.Collector type
func (collector *wpCollector) Describe(ch chan<- *prometheus.Desc) {

	//We set the metrics
	ch <- collector.numPostsMetric
	ch <- collector.numCommentsMetric
	ch <- collector.numUsersMetric
	ch <- collector.numCustomersMetric
	ch <- collector.numSessionsMetric
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

	//select count(*) as numUsers from wp_users;
	var numUsers float64
	q1 := fmt.Sprintf("select count(*) as numUsers from %susers;", collector.dbTablePrefix)
	err = db.QueryRow(q1).Scan(&numUsers)
	if err != nil {
		log.Fatal(err)
	}
	ch <- prometheus.MustNewConstMetric(collector.numUsersMetric, prometheus.CounterValue, numUsers)

	//select count(*) as numCustomers from wp_users inner join wp_usermeta on wp_users.ID = wp_usermeta.user_id where wp_usermeta.meta_key = 'wp_capabilities' and wp_usermeta.meta_value like '%customer%';
	var numCustomers float64
	q2 := fmt.Sprintf("select count(*) as numCustomers from %susers inner join %susermeta on %susers.ID = %susermeta.user_id where %susermeta.meta_key = 'wp_capabilities' and %susermeta.meta_value like '%%customer%%';", collector.dbTablePrefix, collector.dbTablePrefix, collector.dbTablePrefix, collector.dbTablePrefix, collector.dbTablePrefix, collector.dbTablePrefix)
	err = db.QueryRow(q2).Scan(&numCustomers)
	if err != nil {
		log.Fatal(err)
	}
	ch <- prometheus.MustNewConstMetric(collector.numCustomersMetric, prometheus.CounterValue, numCustomers)

	//select count(*) as numComments from wp_comments;
	var numComments float64
	q3 := fmt.Sprintf("select count(*) as numComments from %scomments;", collector.dbTablePrefix)
	err = db.QueryRow(q3).Scan(&numComments)
	if err != nil {
		log.Fatal(err)
	}
	ch <- prometheus.MustNewConstMetric(collector.numCommentsMetric, prometheus.CounterValue, numComments)

	//select count(*) as numPosts from wp_posts WHERE post_type='post' AND post_status!='auto-draft';
	var numPosts float64
	q4 := fmt.Sprintf("select count(*) as numPosts from %sposts WHERE post_type='post' AND post_status!='auto-draft';", collector.dbTablePrefix)
	err = db.QueryRow(q4).Scan(&numPosts)
	if err != nil {
		log.Fatal(err)
	}
	ch <- prometheus.MustNewConstMetric(collector.numPostsMetric, prometheus.CounterValue, numPosts)

	//select count(*) as numSessions from wp_woocommerce_sessions;
	var numSessions float64
	q5 := fmt.Sprintf("select count(*) as numSessions from %swoocommerce_sessions;", collector.dbTablePrefix)
	err = db.QueryRow(q5).Scan(&numSessions)
	if err != nil {
		log.Fatal(err)
	}
	ch <- prometheus.MustNewConstMetric(collector.numSessionsMetric, prometheus.CounterValue, numSessions)

	//select post_status as statusWhs, count(*) as numWhs from wp_posts WHERE post_type='scheduled-action' GROUP BY post_status;
	q6 := fmt.Sprintf("select post_status as statusWhs, count(*) as numWhs from %sposts WHERE post_type='scheduled-action' GROUP BY post_status;", collector.dbTablePrefix)
	rows, err := db.Query(q6)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	collector.numWebhooksMetric.Reset()
	for rows.Next() {
		var statusWhs string
		var numWhs float64
		err = rows.Scan(&statusWhs, &numWhs)
		if err != nil {
			log.Fatal(err)
		}
		collector.numWebhooksMetric.WithLabelValues(statusWhs).Set(numWhs)
	}
	collector.numWebhooksMetric.Collect(ch)

}

func main() {

	wpHostPtr := flag.String("host", "127.0.0.1", "Hostname or Address for DB server")
	wpPortPtr := flag.String("port", "3306", "DB server port")
	wpNamePtr := flag.String("db", "", "DB name")
	wpUserPtr := flag.String("user", "", "DB user for connection")
	wpPassPtr := flag.String("pass", "", "DB password for connection")
	wpTablePrefixPtr := flag.String("tableprefix", "wp_", "Table prefix for WordPress tables")

	flag.Parse()

	dbHost := fmt.Sprintf("%s:%s", *wpHostPtr, *wpPortPtr)
	dbName := *wpNamePtr
	dbUser := *wpUserPtr
	dbPassword := *wpPassPtr
	tablePrefix := *wpTablePrefixPtr

	if dbName == "" {
		fmt.Fprintf(os.Stderr, "flag -db=dbname required!\n")
		os.Exit(1)
	}

	if dbUser == "" {
		fmt.Fprintf(os.Stderr, "flag -user=username required!\n")
		os.Exit(1)
	}

	//We create the collector
	collector := newWordPressCollector(dbHost, dbName, dbUser, dbPassword, tablePrefix)
	prometheus.MustRegister(collector)

	//This section will start the HTTP server and expose
	//any metrics on the /metrics endpoint.
	http.Handle("/metrics", promhttp.Handler())
	log.Info("Beginning to serve on port :9117")
	log.Fatal(http.ListenAndServe(":9117", nil))
}
