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

//This is my collector metrics
type wpCollector struct {
	numPostsMetric    *prometheus.Desc
	numCommentsMetric *prometheus.Desc
	numUsersMetric    *prometheus.Desc

	dbHost        string
	dbName        string
	dbUser        string
	dbPass        string
	dbTablePrefix string
}

//This is a constructor for my wpCollector struct
func newWordPressCollector(host string, dbname string, username string, pass string, tablePrefix string) *wpCollector {
	return &wpCollector{
		numPostsMetric: prometheus.NewDesc("wp_num_posts_metric",
			"Shows the number of total posts in the WordPress site",
			nil, nil,
		),
		numCommentsMetric: prometheus.NewDesc("wp_num_comments_metric",
			"Shows the number of total comments in the WordPress site",
			nil, nil,
		),
		numUsersMetric: prometheus.NewDesc("wp_num_users_metric",
			"Shows the number of registered users in the WordPress site",
			nil, nil,
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

	//select count(*) as num_users from wp_users;
	var numUsers float64
	q1 := fmt.Sprintf("select count(*) as num_users from %susers;", collector.dbTablePrefix)
	err = db.QueryRow(q1).Scan(&numUsers)
	if err != nil {
		log.Fatal(err)
	}

	//select count(*) as num_comments from wp_comments;
	var numComments float64
	q2 := fmt.Sprintf("select count(*) as num_comments from %scomments;", collector.dbTablePrefix)
	err = db.QueryRow(q2).Scan(&numComments)
	if err != nil {
		log.Fatal(err)
	}

	//select count(*) as num_posts from wp_posts WHERE post_type='post' AND post_status!='auto-draft';
	var numPosts float64
	q3 := fmt.Sprintf("select count(*) as num_posts from %sposts WHERE post_type='post' AND post_status!='auto-draft';", collector.dbTablePrefix)
	err = db.QueryRow(q3).Scan(&numPosts)
	if err != nil {
		log.Fatal(err)
	}

	//Write latest value for each metric in the prometheus metric channel.
	//Note that you can pass CounterValue, GaugeValue, or UntypedValue types here.
	ch <- prometheus.MustNewConstMetric(collector.numPostsMetric, prometheus.CounterValue, numPosts)
	ch <- prometheus.MustNewConstMetric(collector.numCommentsMetric, prometheus.CounterValue, numComments)
	ch <- prometheus.MustNewConstMetric(collector.numUsersMetric, prometheus.CounterValue, numUsers)

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
