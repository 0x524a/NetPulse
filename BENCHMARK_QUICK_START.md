# Provider Benchmarking - Quick Start Guide

## What is Provider Benchmarking?

The benchmarking system lets you compare the performance of all speed test providers to see which one works best for your network.

## Key Features

### 🏆 Overall Performance Score
Each provider gets a score (0-100) combining:
- Download speed (30% weight)
- Upload speed (20% weight)
- Latency (20% weight)
- Jitter/Stability (10% weight)
- Reliability (15% weight)
- Consistency (5% weight)

### 📊 Seven Different Rankings
1. **Overall Performance** - Combined score
2. **Download Speed** - Pure download performance
3. **Upload Speed** - Pure upload performance
4. **Latency** - Response time (lower is better)
5. **Jitter** - Stability (lower is better)
6. **Reliability** - Success rate
7. **Consistency** - How stable results are

### 📈 Detailed Statistics
For each provider, see:
- Min/Max/Average for download/upload
- Successful vs failed tests
- Consistency score
- Min/Max/Average latency & jitter

## Quick Examples

### Basic benchmark (1 test per provider)
```bash
./speedtest benchmark
```

### Detailed benchmark (3 tests per provider, 15s each)
```bash
./speedtest benchmark -t 3 -d 15s
```

### With detailed progress
```bash
./speedtest benchmark --verbose
```

### Save to file
```bash
./speedtest benchmark -o results.txt
```

### Combination
```bash
./speedtest benchmark -t 2 -d 12s --verbose -o benchmark.txt
```

## Understanding the Output

### Overall Rankings Section
```
1. fastcom               Score:  78.45/100  Grade: C
2. mlab                  Score:  72.31/100  Grade: C
```
- **Score**: 0-100 combined performance
- **Grade**: A (90+), B (80+), C (70+), D (60+), E (50+), F (<50)

### Speed Rankings
```
1. fastcom                   98.50 Mbps  [100.0%]
2. mlab                      87.32 Mbps  [88.6%]
```
- Left number: Actual speed
- Percentage: Relative to best performer (100% = fastest)

### Latency/Jitter Rankings (Lower is Better)
```
1. fastcom                    8.45 ms   [100.0%]
2. cloudflare                15.67 ms   [53.9%]
```
- Lower latency = better responsiveness
- 100% = best (lowest latency)
- 53.9% = 53.9% as good as best

### Detailed Provider Stats
Shows for each provider:
- Total tests run / successful / failed
- Reliability percentage
- Consistency score (0-100)
- Min/Max/Average for download, upload, latency, jitter

## Grade Interpretation

| Grade | Score  | Meaning |
|-------|--------|---------|
| A     | 90-100 | Excellent - Use this |
| B     | 80-89  | Good - Reliable choice |
| C     | 70-79  | Fair - Acceptable |
| D     | 60-69  | Poor - May have issues |
| E     | 50-59  | Bad - Not recommended |
| F     | <50    | Terrible - Avoid |

## Reliability Interpretation

| % | Meaning |
|---|---------|
| 100% | All tests succeeded |
| 80%+ | Very reliable |
| 50-79% | Inconsistent |
| <50% | Frequently fails |

## Consistency Interpretation

| Score | Meaning |
|-------|---------|
| 90-100 | Highly consistent, reliable |
| 70-89 | Mostly consistent |
| 50-69 | Moderate variations |
| <50 | High variability, unreliable |

## Use Cases

### Choose a Provider
Compare all providers to pick the best one for your needs:
```bash
./speedtest benchmark
# Look at Overall Performance Rankings
```

### Check Provider Stability
Run multiple tests to see if provider is consistent:
```bash
./speedtest benchmark -t 5 -d 20s
# Look at Consistency Ranking and individual test results
```

### Compare Specific Providers
If you already have favorites, compare them:
```bash
./speedtest --provider fastcom,mlab
# Then manually compare results
```

### Track Performance Over Time
Save results periodically to track changes:
```bash
./speedtest benchmark -o benchmark_$(date +%Y%m%d_%H%M%S).txt
```

## Performance Percentages

The `[XX.X%]` shown in rankings means:
- `[100.0%]` = This is the fastest/best provider
- `[90.0%]` = This provider is 90% as fast as the best
- `[50.0%]` = This provider is 50% as fast as the best

## Tips for Accurate Benchmarking

1. **Close other apps** - Stop downloads, streaming, video calls
2. **Run multiple tests** - Use `-t 3` or higher for more reliable results
3. **Use longer duration** - Use `-d 20s` or higher for stability
4. **Run at different times** - Provider performance can vary throughout day
5. **Check consistency** - More stable results = better provider

## Output Saved to File

Use `-o filename.txt` to save results:
```bash
./speedtest benchmark -o results.txt
# Results are in human-readable format
cat results.txt
```

## Next Steps

- Compare providers to find the best one
- Run benchmarks weekly to track changes
- Use benchmark results to troubleshoot network issues
- Share results to help others choose providers
