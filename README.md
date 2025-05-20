# Pressure Test Report Generation

<p>
   English | <a href="README_ZH.md">中文<a/>
</p>

## Functional Overview
This package provides standardized pressure test report generation with the following features:

- ✅ Bilingual output (English/Chinese)
- ⏱️ Automatic calculation of key metrics (QPS, latency percentiles, etc.)
- 📈 Thread-safe concurrent data collection
- 🎨 Visually formatted console output

## Usage Examples
### Basic Usage
```go
func TestExample(t *testing.T) {
    latencies := []time.Duration{
        120 * time.Millisecond,
        150 * time.Millisecond,
        // ...more test data 
    }
    
    pressure_report_format.PrintPressureLog(
        t,
        pressure_report_format.English,
        100,    // Concurrency
        1000,   // Total requests 
        10*time.Second, // Test duration 
        5,      // Failure count 
        latencies,
    )
}
```

### Metric Calculations
| Metric Type       | Calculation Method                  | Notes     |
|----------------|--------------------------|--------|
| QPS            | Total requests / Duration (s)      | Includes failed requests  |
| Latency Percentile      | P50/P90/P99              | Linear interpolation algorithm |
| Throughput         | Successful requests / Duration (s)    |        |

### Sample Reports

#### Chinese Mode

```plaintext
🚀 并发测试报告 
-------------------------------- 
并发数:       50 
总请求量:     1000 
测试时长:     6.066652097s 
失败请求:     0 
QPS:         164.84 
平均延迟:     293.044987ms 
P99延迟:      825.862915ms
```

#### English Mode
```plaintext
🚀 Pressure Test Report 
-------------------------------- 
Concurrency:       50 
Total Requests:     1000 
Duration:     5.759418107s 
Failures:     0 
QPS:         173.63 
Avg Latency:     254.09147ms 
P99 Latency:      750.907967ms
```