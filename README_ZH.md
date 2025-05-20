# 压力测试报告生成

<p>
   <a href="README.md"> English <a/>| 中文
</p>

## 功能概述
本包提供标准化的压力测试报告生成功能，支持：

- ✅ 中英文双语输出
- ⏱️ 自动计算QPS、延迟百分位等关键指标
- 📈 线程安全的并发数据收集
- 🎨 美观的格式化控制台输出


## 使用示例
### 基础用法
```go
func TestExample(t *testing.T) {
    latencies := []time.Duration{
        120 * time.Millisecond,
        150 * time.Millisecond,
        // ...更多测试数据 
    }
    
    pressure_report_format.PrintPressureLog(
        t,
        pressure_report_format.Chinese,
        100,    // 并发数 
        1000,   // 总请求 
        10*time.Second, // 测试时长 
        5,      // 失败数 
        latencies,
    )
}
```

### 指标计算
| 指标类型       | 计算方式                  | 说明     |
|----------------|--------------------------|--------|
| QPS            | 总请求数/测试时长(s)      | 含失败请求  |
| 延迟百分位      | P50/P90/P99              | 线性插值算法 |
| 吞吐量         | 成功请求数/测试时长(s)    |        |

### 基础报告样例

#### 中文模式

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

#### 英文模式
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