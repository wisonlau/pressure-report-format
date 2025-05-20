package pressure_report_format

import (
	"sort"
	"testing"
	"time"
)

// Language 定义语言类型
type Language int

const (
	Chinese Language = iota
	English
)

// PrintPressureLog 输出格式化测试报告（支持中英文）
func PrintPressureLog(t *testing.T, lang Language, concurrency int, totalCalls int, duration time.Duration, failures int32, latencies []time.Duration) {
	var template string

	switch lang {
	case Chinese:
		template = `
🚀 %s 
-------------------------------- 
%s:       %d 
%s:     %d 
%s:     %v 
%s:     %d 
%s:         %.2f 
%s:     %v 
%s:      %v`
		t.Logf(template,
			getLabel(lang, "title"),
			getLabel(lang, "concurrency"), concurrency,
			getLabel(lang, "totalCalls"), totalCalls,
			getLabel(lang, "duration"), duration,
			getLabel(lang, "failures"), failures,
			getLabel(lang, "qps"), float64(totalCalls)/duration.Seconds(),
			getLabel(lang, "avgLatency"), avgLatency(latencies),
			getLabel(lang, "p99Latency"), percentile(latencies, 0.99),
		)
	case English:
		template = `
🚀 %s 
-------------------------------- 
%s:       %d 
%s:     %d 
%s:     %v 
%s:     %d 
%s:         %.2f 
%s:     %v 
%s:      %v`
		t.Logf(template,
			getLabel(lang, "title"),
			getLabel(lang, "concurrency"), concurrency,
			getLabel(lang, "totalCalls"), totalCalls,
			getLabel(lang, "duration"), duration,
			getLabel(lang, "failures"), failures,
			getLabel(lang, "qps"), float64(totalCalls)/duration.Seconds(),
			getLabel(lang, "avgLatency"), avgLatency(latencies),
			getLabel(lang, "p99Latency"), percentile(latencies, 0.99),
		)
	}
}

// getLabel 根据语言和键返回对应的标签文本
func getLabel(lang Language, key string) string {
	labels := map[Language]map[string]string{
		Chinese: {
			"title":       "并发测试报告",
			"concurrency": "并发数",
			"totalCalls":  "总请求量",
			"duration":    "测试时长",
			"failures":    "失败请求",
			"qps":         "QPS",
			"avgLatency":  "平均延迟",
			"p99Latency":  "P99延迟",
		},
		English: {
			"title":       "Pressure Test Report",
			"concurrency": "Concurrency",
			"totalCalls":  "Total Requests",
			"duration":    "Duration",
			"failures":    "Failures",
			"qps":         "QPS",
			"avgLatency":  "Avg Latency",
			"p99Latency":  "P99 Latency",
		},
	}

	return labels[lang][key]
}

// avgLatency 计算平均延迟
func avgLatency(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	return sum / time.Duration(len(durations))
}

// percentile 计算百分位延迟
func percentile(durations []time.Duration, p float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// 排序延迟数据
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	// 计算百分位位置（线性插值）
	k := float64(len(durations)-1) * p
	floor := int(k)
	ceil := floor + 1

	if ceil >= len(durations) {
		return durations[floor]
	}

	weight := k - float64(floor)
	return time.Duration(float64(durations[floor])*(1-weight) + float64(durations[ceil])*weight)
}
