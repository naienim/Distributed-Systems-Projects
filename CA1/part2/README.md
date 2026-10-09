# Part 2 – Concurrency, Scheduling and Context Switching in Go

## Overview

This directory contains the implementation for Part 2 of the assignment that runs a large number of concurrent Go execution units (goroutines), vary key parameters and observe how scheduling, CPU contention and synchronization affect performance.

## File Structure

```text
part2/
├── main.go
├── README.md
└── results/
```

## High Level Design

The program is an experiment runner that systematically explores the performance impact of:

1. Number of goroutines
2. Type of workload
3. GOMAXPROCS value

For each combination it:

- Spawns the given number of goroutines
- Assigns a fixed number of tasks to each goroutine
- Executes one of two workloads:
   - CPU‑bound
   - Mixed (CPU + synchronization + sleep)
- Measures:
   - Total execution time
   - Throughput
   - Average latency per task
   - Min, max, and standard deviation of latency
- Writes the results into a CSV file via the standard library.



## Parameter Space

1. Number of Goroutines: 1, 2, 4, 8, 16, 32, 64
2. GOMAXPROCS Values: he program tests three different GOMAXPROCS settings(1, 2, runtime.NumCPU())
3. Workload Types: The program defines and tests two workloads(cpu, mixed)


## Build Instructions

From inside the part2 directory:

```bash
go build -o part2_experiment main.go
```

This creates an executable named part2_experiment.

## Run Instructions

```bash
./part2_experiment
```

During execution, we will see logs like:

```text
Number of CPU detected: 8
Running: workload=cpu, GOMAXPROCS=1, goroutines=1
Running: workload=cpu, GOMAXPROCS=1, goroutines=2
...
Running: workload=mixed, GOMAXPROCS=8, goroutines=64
Results saved to results/experiment_results.csv
```

After completion:

- The results/ directory will exist.
- The CSV file results/experiment_results.csv will contain all experimental data.