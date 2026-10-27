# Catalog tail latency (Prometheus)

## HTTP p95 / p99

```promql
histogram_quantile(0.95, sum(rate(comics_catalog_http_duration_seconds_bucket[5m])) by (le, route))
histogram_quantile(0.99, sum(rate(comics_catalog_http_duration_seconds_bucket[5m])) by (le, route))
```

## Search path mix

```promql
sum(rate(comics_catalog_search_path_total[5m])) by (path)
```

## Repository phases

```promql
histogram_quantile(0.99, sum(rate(comics_catalog_repo_duration_seconds_bucket[5m])) by (le, phase, search_path))
```

## Scrape isolation

```promql
histogram_quantile(0.99, sum(rate(comics_scrape_phase_duration_seconds_bucket[5m])) by (le, phase))
sum(rate(comics_scrape_noop_writes_skipped_total[5m]))
```
