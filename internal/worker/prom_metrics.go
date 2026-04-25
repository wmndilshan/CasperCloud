package worker

import (
	"context"
	"log"
	"net/http"
	"time"

	"caspercloud/internal/instancemetrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type libvirtStatsCollector struct {
	store *instancemetrics.MemoryStore

	memMaxKiB *prometheus.Desc
	memKiB    *prometheus.Desc
	cpuPct    *prometheus.Desc
	cpuTime   *prometheus.Desc
	vcpus     *prometheus.Desc

	diskRd *prometheus.Desc
	diskWr *prometheus.Desc
	netRx  *prometheus.Desc
	netTx  *prometheus.Desc
}

func newLibvirtStatsCollector(store *instancemetrics.MemoryStore) *libvirtStatsCollector {
	labels := []string{"project_id", "instance_id", "instance_name"}
	return &libvirtStatsCollector{
		store: store,
		memMaxKiB: prometheus.NewDesc(
			"caspercloud_libvirt_domain_max_memory_kib",
			"Maximum memory from virDomainGetInfo (KiB).",
			labels, nil,
		),
		memKiB: prometheus.NewDesc(
			"caspercloud_libvirt_domain_memory_kib",
			"Current memory from virDomainGetInfo (KiB).",
			labels, nil,
		),
		cpuPct: prometheus.NewDesc(
			"caspercloud_libvirt_domain_cpu_host_percent",
			"Approximate guest CPU usage vs wall time since last poll (0-100).",
			labels, nil,
		),
		cpuTime: prometheus.NewDesc(
			"caspercloud_libvirt_domain_cpu_time_ns",
			"CPU time from virDomainGetInfo (nanoseconds, cumulative while the domain is running).",
			labels, nil,
		),
		vcpus: prometheus.NewDesc(
			"caspercloud_libvirt_domain_vcpu_count",
			"Configured vCPU count.",
			labels, nil,
		),
		diskRd: prometheus.NewDesc(
			"caspercloud_libvirt_domain_disk_read_bytes",
			"Disk read bytes from virDomainBlockStats for the root disk (vda).",
			labels, nil,
		),
		diskWr: prometheus.NewDesc(
			"caspercloud_libvirt_domain_disk_write_bytes",
			"Disk write bytes from virDomainBlockStats for the root disk (vda).",
			labels, nil,
		),
		netRx: prometheus.NewDesc(
			"caspercloud_libvirt_domain_net_rx_bytes",
			"Network RX bytes from virDomainInterfaceStats (host vnet).",
			labels, nil,
		),
		netTx: prometheus.NewDesc(
			"caspercloud_libvirt_domain_net_tx_bytes",
			"Network TX bytes from virDomainInterfaceStats (host vnet).",
			labels, nil,
		),
	}
}

func (c *libvirtStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.memMaxKiB
	ch <- c.memKiB
	ch <- c.cpuPct
	ch <- c.cpuTime
	ch <- c.vcpus
	ch <- c.diskRd
	ch <- c.diskWr
	ch <- c.netRx
	ch <- c.netTx
}

func (c *libvirtStatsCollector) Collect(ch chan<- prometheus.Metric) {
	if c.store == nil {
		return
	}
	for _, p := range c.store.Snapshot() {
		lv := []string{p.ProjectID.String(), p.InstanceID.String(), p.InstanceName}
		ch <- prometheus.MustNewConstMetric(c.memMaxKiB, prometheus.GaugeValue, float64(p.MaxMemoryKiB), lv...)
		ch <- prometheus.MustNewConstMetric(c.memKiB, prometheus.GaugeValue, float64(p.MemoryKiB), lv...)
		ch <- prometheus.MustNewConstMetric(c.cpuPct, prometheus.GaugeValue, p.CPUHostPercent, lv...)
		ch <- prometheus.MustNewConstMetric(c.cpuTime, prometheus.GaugeValue, float64(p.CPUTimeNs), lv...)
		ch <- prometheus.MustNewConstMetric(c.vcpus, prometheus.GaugeValue, float64(p.VCPUCount), lv...)
		ch <- prometheus.MustNewConstMetric(c.diskRd, prometheus.GaugeValue, float64(p.DiskReadBytes), lv...)
		ch <- prometheus.MustNewConstMetric(c.diskWr, prometheus.GaugeValue, float64(p.DiskWriteBytes), lv...)
		ch <- prometheus.MustNewConstMetric(c.netRx, prometheus.GaugeValue, float64(p.NetRxBytes), lv...)
		ch <- prometheus.MustNewConstMetric(c.netTx, prometheus.GaugeValue, float64(p.NetTxBytes), lv...)
	}
}

// RunPrometheusMetricsServer serves GET /metrics until ctx is cancelled.
func RunPrometheusMetricsServer(ctx context.Context, listen string, store *instancemetrics.MemoryStore) {
	if listen == "" || store == nil {
		return
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(newLibvirtStatsCollector(store))

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	srv := &http.Server{
		Addr:         listen,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("caspercloud worker: prometheus listening on %s (/metrics)", listen)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("caspercloud worker: prometheus server: %v", err)
	}
}
