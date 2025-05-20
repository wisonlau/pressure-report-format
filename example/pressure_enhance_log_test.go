package example_test

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAPIPressure(t *testing.T) {
	// 在测试开始时记录初始goroutine数量
	startGoroutines := runtime.NumGoroutine()
	// 在测试开始时初始化监控
	cpuMon := NewCPUMonitor()
	debug.SetGCPercent(20) // 激进GC策略便于观察

	const (
		concurrency = 100
		totalCalls  = 10000
		timeout     = 1500 * time.Millisecond // 优化点：收紧超时阈值
	)

	var (
		latencies         = make([]time.Duration, 0, totalCalls)
		failures          int32
		slowRequests      int32
		currentConcurrent int32
		peakConcurrent    int32
		statusCodes       = make(map[int]int)
		errorMessages     = make(map[string]int)
		dataTransferred   int64
		mu                sync.Mutex
		wg                sync.WaitGroup
	)

	// 内存预热
	func() {
		tmp := make([][]byte, 100)
		for i := range tmp {
			tmp[i] = make([]byte, 1<<20) // 1MB
		}
	}()

	wg.Add(concurrency)
	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < totalCalls/concurrency; j++ {
				reqStart := time.Now()

				// 实时监控并发峰值
				curr := atomic.AddInt32(&currentConcurrent, 1)
				if curr > atomic.LoadInt32(&peakConcurrent) {
					atomic.StoreInt32(&peakConcurrent, curr)
				}
				defer atomic.AddInt32(&currentConcurrent, -1)

				// 执行请求
				latency, code, err := mockAPICall(reqStart, timeout)

				mu.Lock()
				if err != nil {
					failures++
					// 错误智能分类（新增）
					errType := classifyError(err)
					errorMessages[errType]++
				} else {
					latencies = append(latencies, latency)
					if latency > 500*time.Millisecond { // 记录慢请求
						atomic.AddInt32(&slowRequests, 1)
					}
					statusCodes[code]++
					dataTransferred += rand.Int63n(1024)
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)

	// 确保GC和CPU统计在测试结束时立即执行
	runtime.GC() // 强制触发GC

	// 获取最终系统状态
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 检测goroutine泄漏
	endGoroutines := runtime.NumGoroutine()
	leakThreshold := 5 // 允许的goroutine波动阈值
	goroutineLeak := (endGoroutines - startGoroutines) > leakThreshold

	PrintEnhancedReport(t, TestResult{
		Concurrency:     concurrency,
		TotalCalls:      totalCalls,
		Duration:        duration,
		Failures:        failures,
		SlowRequests:    slowRequests,
		Latencies:       latencies,
		StatusCodes:     statusCodes,
		ErrorMessages:   errorMessages,
		DataTransferred: dataTransferred,
		PeakConcurrent:  peakConcurrent,
		GCNum:           getGCNum(),
		CPUUsage:        cpuMon.Usage(),
		Goroutines:      runtime.NumGoroutine(),        // 新增
		MemAllocMB:      float64(memStats.Alloc) / 1e6, // 字节转MB
		GoroutineLeak:   goroutineLeak,
		StartGoroutines: startGoroutines,
	})
}

// 智能错误分类（新增）
func classifyError(err error) string {
	switch {
	case strings.Contains(err.Error(), "timeout"):
		return "timeout"
	case strings.Contains(err.Error(), "connection refused"):
		return "connection_error"
	default:
		return "unknown_error"
	}
}

// 模拟API调用（实际替换为真实逻辑）
func mockAPICall(reqStart time.Time, timeout time.Duration) (latency time.Duration, code int, err error) {
	defer func() { latency = time.Since(reqStart) }()

	// 模拟更真实的延迟分布（优化点）
	baseDelay := time.Duration(30+rand.Intn(70)) * time.Millisecond
	additionalDelay := time.Duration(rand.Float64()*600) * time.Millisecond

	// 模拟10%错误率（含不同类型错误）
	if rand.Float32() < 0.1 {
		switch rand.Intn(3) {
		case 0:
			time.Sleep(timeout + 100*time.Millisecond)
			return 0, 500, fmt.Errorf("timeout at %v", reqStart.Format("15:04:05.000"))
		case 1:
			return 0, 502, fmt.Errorf("connection refused")
		default:
			return 0, 503, fmt.Errorf("service unavailable")
		}
	}

	time.Sleep(baseDelay + additionalDelay)
	return 0, 200, nil
}

// TestResult 增强版测试结果结构体
type TestResult struct {
	Concurrency     int
	TotalCalls      int
	Duration        time.Duration
	Failures        int32
	SlowRequests    int32 // 慢请求计数
	Latencies       []time.Duration
	StatusCodes     map[int]int
	ErrorMessages   map[string]int
	DataTransferred int64
	PeakConcurrent  int32   // 峰值并发
	GCNum           uint32  // GC次数
	CPUUsage        float64 // CPU使用率百分比
	Goroutines      int     // 新增：当前goroutine数量
	MemAllocMB      float64 // 新增：内存分配量(MB)
	GoroutineLeak   bool    // 新增：goroutine泄漏标记
	StartGoroutines int     // 新增：测试开始时的goroutine数量
}

// PrintEnhancedReport 增强版测试报告
func PrintEnhancedReport(t *testing.T, result TestResult) {
	// 基础指标计算
	qps := float64(result.TotalCalls) / result.Duration.Seconds()
	successRate := float64(result.TotalCalls-int(result.Failures)) / float64(result.TotalCalls) * 100

	// 延迟统计
	avg := avgLatency(result.Latencies)
	p50 := percentile(result.Latencies, 0.5)
	p90 := percentile(result.Latencies, 0.9)
	p99 := percentile(result.Latencies, 0.99)
	max := maxLatency(result.Latencies)
	min := minLatency(result.Latencies)

	// 带宽计算
	throughput := float64(result.DataTransferred) / result.Duration.Seconds()

	// 可视化延迟分布
	histogram := buildLatencyHistogram(result.Latencies)

	leakStatus := "✅ 正常"
	if result.GoroutineLeak {
		leakStatus = "❌ 检测到泄漏"
	}

	t.Logf(`
🚀 增强版压力测试报告 
================================ 
📊 基础指标 
-------------------------------- 
并发数:       %d (req/goroutine)
总请求量:     %d (失败: %d)
成功率:       %.2f%%
测试时长:     %v 
QPS:         %.2f 
吞吐量:      %.2f MB/s 
慢请求(>500ms): %d (%.1f%%)
峰值并发:      %d
 
⏱️ 延迟统计 (单位: ms)
-------------------------------- 
平均:        %v 
P50:        %v 
P90:        %v 
P99:        %v 
最大:        %v 
最小:        %v 
 
📈 延迟分布 
%s 
 
🛠️ 系统指标 
-------------------------------- 
总数据量:    %s 
状态码分布:  %v
GC次数:     %d
CPU使用率:  %.4f%%
Goroutines: %d (初始: %d) %s 
内存分配:    %.2f MB`,
		result.Concurrency,
		result.TotalCalls,
		result.Failures,
		successRate,
		result.Duration,
		qps,
		throughput/(1024*1024),
		result.SlowRequests,
		float64(result.SlowRequests)/float64(len(result.Latencies))*100,
		result.PeakConcurrent,

		formatDuration(avg),
		formatDuration(p50),
		formatDuration(p90),
		formatDuration(p99),
		formatDuration(max),
		formatDuration(min),

		histogram,

		formatBytes(result.DataTransferred),
		formatStatusCodes(result.StatusCodes),
		result.GCNum,
		result.CPUUsage,
		result.Goroutines,
		result.StartGoroutines,
		leakStatus,
		result.MemAllocMB,
	)

	// 输出错误摘要
	if len(result.ErrorMessages) > 0 {
		t.Logf("\n❌ 错误摘要:\n%s", formatErrors(result.ErrorMessages))
	}
}

// 获取GC次数（兼容Go 1.18+）
func getGCNum() uint32 {
	// 强制触发一次GC确保统计准确
	debug.FreeOSMemory()

	var stats debug.GCStats
	debug.ReadGCStats(&stats)
	return uint32(stats.NumGC)
}

// CPUMonitor 精确的CPU使用率监控
type CPUMonitor struct {
	startTime time.Time
	startCPU  uint64
}

func NewCPUMonitor() *CPUMonitor {
	return &CPUMonitor{
		startTime: time.Now(),
		startCPU:  getProcessCPUTime(),
	}
}

func (m *CPUMonitor) Usage() float64 {
	elapsed := time.Since(m.startTime).Seconds()
	cpuTime := float64(getProcessCPUTime() - m.startCPU)
	usage := (cpuTime / elapsed) * 100 / float64(runtime.NumCPU())
	return math.Min(100, math.Max(0, usage)) // 限制在0-100%范围内
}

// 获取进程累计CPU时间（纳秒）
func getProcessCPUTime() uint64 {
	switch runtime.GOOS {
	case "linux", "darwin":
		cmd := exec.Command("ps", "-o", "time=", "-p", strconv.Itoa(os.Getpid()))
		out, err := cmd.Output()
		if err != nil {
			return 0
		}
		parts := strings.Split(strings.TrimSpace(string(out)), ":")
		if len(parts) == 3 {
			h, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])
			s, _ := strconv.Atoi(parts[2])
			return uint64(h*3600+m*60+s) * 1e9
		}
	case "windows":
		cmd := exec.Command("wmic", "process", "where",
			"processid="+strconv.Itoa(os.Getpid()), "get", "KernelModeTime,UserModeTime")
		out, err := cmd.Output()
		if err != nil {
			return 0
		}
		lines := strings.Split(string(out), "\n")
		if len(lines) >= 2 {
			times := strings.Fields(lines[1])
			if len(times) == 2 {
				kernel, _ := strconv.ParseUint(times[0], 10, 64)
				user, _ := strconv.ParseUint(times[1], 10, 64)
				return (kernel + user) * 100 // 转换为纳秒
			}
		}
	}
	return 0
}

// 新增辅助函数
func maxLatency(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	max := durations[0]
	for _, d := range durations {
		if d > max {
			max = d
		}
	}
	return max
}

func minLatency(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	min := durations[0]
	for _, d := range durations {
		if d < min {
			min = d
		}
	}
	return min
}

// 构建延迟直方图
func buildLatencyHistogram(durations []time.Duration) string {
	if len(durations) == 0 {
		return "无延迟数据"
	}

	buckets := []time.Duration{
		0, 10 * time.Millisecond, 50 * time.Millisecond,
		100 * time.Millisecond, 200 * time.Millisecond,
		500 * time.Millisecond, 1 * time.Second,
		2 * time.Second, 5 * time.Second,
	}

	counts := make([]int, len(buckets))
	for _, d := range durations {
		for i := len(buckets) - 1; i >= 0; i-- {
			if d >= buckets[i] {
				counts[i]++
				break
			}
		}
	}

	var sb strings.Builder
	total := len(durations)
	for i := 0; i < len(buckets)-1; i++ {
		percentage := float64(counts[i]) / float64(total) * 100
		sb.WriteString(fmt.Sprintf("%6s-%-6s | %-60s %.1f%%\n",
			formatDuration(buckets[i]),
			formatDuration(buckets[i+1]),
			strings.Repeat("█", int(percentage/2)),
			percentage))
	}

	return sb.String()
}

// 格式化工具函数
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatStatusCodes(codes map[int]int) string {
	var sb strings.Builder
	for code, count := range codes {
		sb.WriteString(fmt.Sprintf("%d: %d, ", code, count))
	}
	return strings.TrimSuffix(sb.String(), ", ")
}

func formatErrors(errors map[string]int) string {
	var sb strings.Builder
	for msg, count := range errors {
		sb.WriteString(fmt.Sprintf("  - %s (出现 %d 次)\n", msg, count))
	}
	return sb.String()
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
