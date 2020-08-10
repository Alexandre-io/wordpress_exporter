# wordpress_exporter
Prometheus exporter for WordPress / WooCommerce

Inspired by [kotsis/wordpress_exporter](https://github.com/kotsis/wordpress_exporter) and [devent/wordpress_exporter](https://github.com/devent/wordpress_exporter)

# Usage of wordpress_exporter
```sh
docker run --name wordpress_exporter -p 9850:9850 -e WORDPRESS_DB_HOST="127.0.0.1" -e WORDPRESS_DB_PORT="3306" -e WORDPRESS_DB_USER="wordpress" -e WORDPRESS_DB_NAME="wordpress" -e WORDPRESS_DB_PASSWORD="wordpress" -e WORDPRESS_TABLE_PREFIX="wp_" -d alexandreio/wordpress_exporter:latest
```
# Prometheus configuration for wordpress_exporter
For Prometheus to start scraping the metrics you have to edit /etc/prometheus/prometheus.yml and add:

```sh
  - job_name: 'wordpress'
    # metrics_path defaults to '/metrics'
    # scheme defaults to 'http'.
    static_configs:
    - targets: ['localhost:9850']
```
