# Provider Benchmarking System

A comprehensive benchmarking suite for comparing internet speed test provider performance.

## Features

### Benchmark Metrics

The benchmarking system tracks and analyzes:

- **Download Speed**: Average, Min, Max (in Mbps)
- **Upload Speed**: Average, Min, Max (in Mbps)
- **Latency**: Average response time (in ms)
- **Jitter**: Latency variation (in ms)
- **Reliability**: Percentage of successful tests
- **Consistency**: Stability of results (coefficient of variation)

### Scoring System

Each provider receives an **Overall Performance Score** (0-100) calculated as:

```
Overall Score = (Download×0.30) + (Upload×0.20) + (Latency×0.20)
              + (Jitter×0.10) + (Reliability×0.15) + (Consistency×0.05)
```

#### Scoring Methodology

- **Download**: 300 Mbps = 100 points (scales linearly)
- **Upload**: 150 Mbps = 100 points (scales linearly)
- **Latency**: 0 ms = 100 points, 100 ms = 0 points
- **Jitter**: 0 ms = 100 points, 50 ms = 0 points
- **Reliability**: Percentage of successful tests (0-100)
- **Consistency**: Based on standard deviation of results (0-100)

### Rankings

The system generates multiple rankings:

1. **Overall Performance**: Combined score across all metrics
2. **Download Speed**: Ranked by average download speed
3. **Upload Speed**: Ranked by average upload speed
4. **Latency**: Ranked by average latency (lower is better)
5. **Jitter**: Ranked by average jitter (lower is better)
6. **Reliability**: Ranked by success rate
7. **Consistency**: Ranked by consistency score (higher is better)

## Usage

### Basic Benchmark

Run a single test on each provider:

```bash
./speedtest benchmark
```

Output includes:
- Overall performance rankings with scores
- Download speed rankings (with relative percentages)
- Upload speed rankings (with relative percentages)
- Latency rankings
- Jitter rankings
- Reliability rankings
- Consistency rankings
- Detailed statistics per provider

### Advanced Options

```bash
# Run 3 tests per provider with custom duration
./speedtest benchmark --tests 3 --duration 12s

# Run with verbose output showing progress
./speedtest benchmark --verbose

# Save results to file
./speedtest benchmark --output benchmark_results.txt

# Combine options
./speedtest benchmark -t 2 -d 15s --verbose -o results.txt
```

### CLI Flags

- `--tests, -t`: Number of tests per provider (default: 1)
- `--duration, -d`: Duration of each test (default: 10s)
- `--verbose`: Enable detailed progress output
- `--output, -o`: Save benchmark report to file

## Architecture

### Core Components

1. **BenchmarkSuite** (`pkg/benchmarks/suite.go`)
   - Aggregates test results from all providers
   - Calculates statistics and rankings
   - Generates performance scores

2. **Formatter** (`pkg/benchmarks/formatter.go`)
   - Formats benchmark results for display
   - Generates detailed comparison tables
   - Explains scoring methodology

3. **Client Integration** (`pkg/speedtest/benchmark.go`)
   - Orchestrates multi-provider testing
   - Collects results into benchmark suite
   - Handles error management

### Data Flow

```
CLI Command (benchmark)
    ↓
cmdBenchmark (cmd/main.go)
    ↓
Client.RunBenchmark() (pkg/speedtest/benchmark.go)
    ↓
Run tests for each provider
    ↓
BenchmarkSuite.AddResult() (pkg/benchmarks/suite.go)
    ↓
BenchmarkSuite.Finalize() - Calculate aggregates
    ↓
FormatBenchmarkReport() (pkg/benchmarks/formatter.go)
    ↓
Display / Save to file
```

## Output Example

```
╔════════════════════════════════════════════════════════════════╗
║          SPEED TEST PROVIDER BENCHMARK RESULTS                  ║
╚════════════════════════════════════════════════════════════════╝

OVERALL PERFORMANCE RANKINGS
────────────────────────────────────────────────────────────────
1. fastcom               Score:  78.45/100  Grade: C
2. mlab                  Score:  72.31/100  Grade: C
3. cloudflare            Score:  65.22/100  Grade: D

DOWNLOAD SPEED RANKINGS
────────────────────────────────────────────────────────────────
1. fastcom                   98.50 Mbps  [100.0%]
2. mlab                      87.32 Mbps  [88.6%]
3. cloudflare                82.15 Mbps  [83.4%]

LATENCY RANKINGS (Lower is Better)
────────────────────────────────────────────────────────────────
1. fastcom                    8.45 ms   [100.0%]
2. mlab                      12.30 ms   [68.7%]
3. cloudflare                15.67 ms   [53.9%]

... [more rankings]

QUICK COMPARISON TABLE
════════════════════════════════════════════════════════════════
Provider             | Download   | Upload   | Latency  | Score | Grade
────────────────────────────────────────────────────────────────
fastcom              |   98.50 M  |   45.67 M |   8.45 ms |  78.45 | C
mlab                 |   87.32 M  |   42.13 M |  12.30 ms |  72.31 | C
cloudflare           |   82.15 M  |   38.90 M |  15.67 ms |  65.22 | D
════════════════════════════════════════════════════════════════

SCORING METHODOLOGY
════════════════════════════════════════════════════════════════
Overall Score = (Download×0.30) + (Upload×0.20) + (Latency×0.20)
              + (Jitter×0.10) + (Reliability×0.15) + (Consistency×0.05)

Where:
  - Download:    300 Mbps = 100 points
  - Upload:      150 Mbps = 100 points
  - Latency:     0 ms = 100 points, 100 ms = 0 points
  - Jitter:      0 ms = 100 points, 50 ms = 0 points
  - Reliability: % of successful tests
  - Consistency: Based on result variance (lower stddev = higher score)

Benchmark Duration: 2m15s
```

## Interpretation Guide

### Overall Score Grades

- **A**: 90-100 - Excellent provider, highly recommended
- **B**: 80-89 - Good provider, reliable performance
- **C**: 70-79 - Fair provider, adequate for most uses
- **D**: 60-69 - Below average, may have reliability issues
- **E**: 50-59 - Poor provider, not recommended
- **F**: < 50 - Very poor provider, avoid if possible

### Performance Percentages

Shown as `[XX.X%]` in rankings, indicating performance relative to the best provider in that metric:

- `[100.0%]` = Best performer in that metric
- `[90.0%]` = 90% of best performer's speed
- `[50.0%]` = Half the speed of best performer

### Reliability

- `100%` = All tests successful
- `80%` = 4 out of 5 tests successful (indicates inconsistent provider)
- `< 50%` = Provider frequently fails tests (not recommended)

### Consistency Score

- `90-100` = Highly consistent, reliable results
- `70-89` = Mostly consistent, minor variations
- `50-69` = Moderate variations between tests
- `< 50` = High variability, unpredictable results

## Use Cases

### 1. Provider Selection
Run a benchmark to determine which provider best fits your needs:
```bash
./speedtest benchmark -t 3 -d 15s
```

### 2. Provider Comparison
Compare specific providers:
```bash
./speedtest --provider fastcom,mlab,cloudflare
# Then compare results manually
```

### 3. Performance Tracking
Monitor provider performance over time:
```bash
for i in {1..5}; do
  ./speedtest benchmark -o results_run_$i.txt
  sleep 3600  # Run hourly
done
```

### 4. Network Diagnosis
Identify which providers work best on your network:
```bash
./speedtest benchmark --verbose
# See which providers fail and which succeed
```

## Implementation Details

### Consistency Calculation

Consistency is based on the **coefficient of variation** (CV) of download speeds:

```
CV = Standard Deviation / Mean
Consistency Score = MAX(0, 100 - (CV × 100))
```

Example:
- 5 tests with speeds: 100, 102, 101, 99, 98 Mbps
- Mean = 100, StdDev = 1.4, CV = 0.014
- Consistency = 100 - 1.4 = 98.6/100

### Reliability Calculation

```
Reliability % = (Successful Tests / Total Tests) × 100
```

### Performance Score Weighting

The weighted average prioritizes:
1. **Download speed** (30%) - Most important for users
2. **Upload speed** (20%) - Secondary importance
3. **Latency** (20%) - Critical for responsiveness
4. **Jitter** (10%) - Affects stability
5. **Reliability** (15%) - Must work consistently
6. **Consistency** (5%) - Minor factor

## Extending Benchmarks

To add custom metrics or modify scoring:

1. Update `ProviderBenchmark` struct in `benchmark_types.go`
2. Modify `Finalize()` in `suite.go` to calculate new metrics
3. Update `calculatePerformanceScore()` to include new weights
4. Update `FormatBenchmarkReport()` in `formatter.go` to display new data

## Notes

- Benchmarks run sequentially (one provider at a time)
- Total benchmark duration = (tests per provider) × (duration) × (number of providers)
- Internet quality varies; run multiple tests for accurate comparisons
- Best results when network is otherwise idle
- Consistency scores are more meaningful with 3+ tests per provider
