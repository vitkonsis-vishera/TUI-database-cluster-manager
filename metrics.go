package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type MetricItem struct {
	ID          string
	Name        string
	Description string
	Category    string
	Enabled     bool
}

// Список всех доступных метрик PostgreSQL и HA-компонентов
func defaultMetrics() []MetricItem {
	return []MetricItem{
		// Host Resources
		{ID: "cpu_load", Name: "CPU Load (%)", Category: "Host Resources", Description: "Загрузка процессора хоста", Enabled: true},
		{ID: "ram_usage", Name: "RAM Usage (%)", Category: "Host Resources", Description: "Использование оперативной памяти", Enabled: true},
		{ID: "disk_usage", Name: "Disk Usage (%)", Category: "Host Resources", Description: "Заполненность основного диска", Enabled: true},

		// PostgreSQL Core
		{ID: "active_conn", Name: "Active Connections", Category: "PostgreSQL Core", Description: "Количество активных подключений", Enabled: true},
		{ID: "max_conn", Name: "Max Connections Limit", Category: "PostgreSQL Core", Description: "Максимальный лимит подключений", Enabled: true},
		{ID: "cache_hit", Name: "Cache Hit Ratio", Category: "PostgreSQL Core", Description: "Процент попаданий в кэш buffer cache", Enabled: true},
		{ID: "db_size", Name: "Database Size", Category: "PostgreSQL Core", Description: "Общий размер баз данных", Enabled: true},
		{ID: "tps", Name: "Transactions Per Sec (TPS)", Category: "PostgreSQL Core", Description: "Число транзакций в секунду", Enabled: true},
		{ID: "temp_files", Name: "Temp Files Created", Category: "PostgreSQL Core", Description: "Созданные временные файлы", Enabled: false},
		{ID: "deadlocks", Name: "Deadlocks Count", Category: "PostgreSQL Core", Description: "Количество дедлоков", Enabled: true},

		// High Availability & Replication
		{ID: "repl_lag_bytes", Name: "Replication Lag (Bytes)", Category: "Replication & HA", Description: "Задержка репликации в байтах", Enabled: true},
		{ID: "repl_lag_sec", Name: "Replication Lag (Seconds)", Category: "Replication & HA", Description: "Задержка репликации в секундах", Enabled: true},
		{ID: "patroni_health", Name: "Patroni Leader / Cluster State", Category: "Replication & HA", Description: "Статус кластера Patroni", Enabled: true},
		{ID: "dcs_state", Name: "DCS Health (etcd/Consul)", Category: "Replication & HA", Description: "Состояние распределенного хранилища", Enabled: false},

		// Performance & Locks
		{ID: "locks_waiting", Name: "Waiting Locks Count", Category: "Performance", Description: "Блокировки в ожидании", Enabled: true},
		{ID: "autovacuum", Name: "Autovacuum Active Workers", Category: "Performance", Description: "Активные воркеры autovacuum", Enabled: true},
		{ID: "bloat_ratio", Name: "Table/Index Bloat Ratio", Category: "Performance", Description: "Процент раздувания таблиц/индексов", Enabled: false},

		// Connection Pooling
		{ID: "pgbouncer_pools", Name: "PgBouncer Active Pools", Category: "PgBouncer / Proxy", Description: "Активные пулы PgBouncer", Enabled: false},
		{ID: "pgbouncer_avg_req", Name: "PgBouncer Avg Request Time", Category: "PgBouncer / Proxy", Description: "Среднее время запроса PgBouncer", Enabled: false},
	}
}

// Заглушка для получения значений метрик
func (m model) getMetricValue(id string) string {
	switch id {
	case "active_conn":
		return "7/100"
	case "max_conn":
		return "100"
	case "cache_hit":
		return "99.8%"
	case "db_size":
		return "42 GB"
	case "tps":
		return "1250"
	case "temp_files":
		return "0"
	case "deadlocks":
		return "0"
	case "repl_lag_bytes":
		return "0 B"
	case "repl_lag_sec":
		return "0s"
	case "patroni_health":
		return "Healthy (Leader)"
	case "dcs_state":
		return "etcd (ok)"
	case "locks_waiting":
		return "0"
	case "autovacuum":
		return "1 worker"
	case "bloat_ratio":
		return "2.1%"
	case "pgbouncer_pools":
		return "5 active"
	case "pgbouncer_avg_req":
		return "1.2ms"
	default:
		return "N/A"
	}
}

// Отрисовка списка метрик на вкладке 2: Metrics
func (m model) renderMetricsTab() string {
	var activeMetrics []string
	for _, item := range m.availableMetrics {
		if item.Enabled {
			val := m.getMetricValue(item.ID)
			activeMetrics = append(activeMetrics, fmt.Sprintf("%s: %s", item.Name, val))
		}
	}

	if len(activeMetrics) == 0 {
		return boxStyle.Width(m.width - 6).Render("No metrics selected.\nPress Ctrl+D to open metric display settings.")
	}

	body := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("PostgreSQL & Cluster Metrics") + "\n\n"
	body += strings.Join(activeMetrics, "  |  ")

	return boxStyle.Width(m.width - 6).Render(body)
}
