package pressure_report_format

import (
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestK6AppDataPressure(t *testing.T) {
	const (
		concurrency = 50                                      // 并发数
		totalCalls  = 1000                                    // 总请求量
		targetURL   = "https://k6.io/page-data/app-data.json" // 测试目标URL
	)

	var (
		wg          sync.WaitGroup
		failures    int32                                  // 失败计数器
		latencies   = make([]time.Duration, 0, totalCalls) // 延迟记录
		latencyLock sync.Mutex                             // 保护latencies的锁
	)

	// 创建HTTP客户端（配置超时等参数）
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       concurrency,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}

	startTime := time.Now()

	// 启动并发请求
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < totalCalls/concurrency; j++ {
				start := time.Now()

				// 执行HTTP GET请求
				resp, err := client.Get(targetURL)
				if err != nil {
					atomic.AddInt32(&failures, 1)
					continue
				}

				// 必须读取并关闭响应体
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					atomic.AddInt32(&failures, 1)
					continue
				}

				// 记录延迟
				latency := time.Since(start)
				latencyLock.Lock()
				latencies = append(latencies, latency)
				latencyLock.Unlock()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)

	// 输出中文报告
	// PrintPressureLog(t, Chinese, concurrency, totalCalls, duration, failures, latencies)

	// 输出英文报告（可选）
	PrintPressureLog(t, English, concurrency, totalCalls, duration, failures, latencies)
}
