package main

import (
	"os"
	"fmt"
	"sync"
	"math"
	"time"
	"runtime"
	"strconv"
	"encoding/csv"
)

type ExperimentConfig struct {
	Workload string
	GOMAXPROCS int
	Goroutines int
	TasksPerGor int
	InnerLoopIters int
}

type Result struct {
	Workload string
	GOMAXPROCS int
	Goroutines int
	TotalTasks int
	Throughput float64
	TotalTime time.Duration
	AvgLatency time.Duration
	MinLatency time.Duration
	MaxLatency time.Duration
	StdLatency time.Duration
}

type GoroutineStatistics struct {
	TaskCount int
	SumLatency time.Duration
	MinLatency time.Duration
	MaxLatency time.Duration
	SumLatencySquared float64
}

func cpuTask(iter int) {
	sum := 0
	for i := 0; i < iter; i++ {
		sum += i*i
	}
	_ = sum
}

func mixedTask(iter int, mutex *sync.Mutex) {
	sum := 0
	for i := 0; i < iter; i++ {
		sum += i*i
	}
	mutex.Lock()
	sum++
	mutex.Unlock()
	_ = sum
	time.Sleep(10*time.Microsecond)
}

func worker(id int, config ExperimentConfig, mutex *sync.Mutex, statisticsChannel chan GoroutineStatistics, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	grStatistics := GoroutineStatistics{TaskCount: config.TasksPerGor, MinLatency: time.Duration(1<<63 - 1), MaxLatency: 0, SumLatency: 0, SumLatencySquared: 0}

	for task := 0; task < config.TasksPerGor; task++ {
		startTask := time.Now()

		if config.Workload == "cpu" {
			cpuTask(config.InnerLoopIters)
		} else if config.Workload == "mixed" {
			mixedTask(config.InnerLoopIters, mutex)
		} else {
			cpuTask(config.InnerLoopIters)
		}

		latency := time.Since(startTask)
		grStatistics.SumLatency += latency
		latencyNs := float64(latency.Nanoseconds())
		grStatistics.SumLatencySquared += latencyNs * latencyNs

		if latency < grStatistics.MinLatency {
			grStatistics.MinLatency = latency
		}
		if latency > grStatistics.MaxLatency {
			grStatistics.MaxLatency = latency
		}
	}

	statisticsChannel <- grStatistics
}

func closeWhenDone(waitGroup *sync.WaitGroup, statisticsChannel chan GoroutineStatistics) {
	waitGroup.Wait()
	close(statisticsChannel)
}


func runExperiment(config ExperimentConfig) Result {
	previousProcs := runtime.GOMAXPROCS(config.GOMAXPROCS)
	defer runtime.GOMAXPROCS(previousProcs)

	var mutex sync.Mutex
	var waitGroup sync.WaitGroup
	statisticsChannel := make(chan GoroutineStatistics, config.Goroutines)

	startAll := time.Now()
	for gr := 0; gr < config.Goroutines; gr++ {
		waitGroup.Add(1)
		go worker(gr, config, &mutex, statisticsChannel, &waitGroup)
	}

	go closeWhenDone(&waitGroup, statisticsChannel)

	totalTasks := 0
	totalSumLatencySquared := 0.0
	totalSumLatency := time.Duration(0)
	minLatency := time.Duration(1<<63 - 1)
	maxLatency := time.Duration(0)

	for grStatistics := range statisticsChannel {
		totalTasks += grStatistics.TaskCount
		totalSumLatency += grStatistics.SumLatency
		totalSumLatencySquared += grStatistics.SumLatencySquared
		if grStatistics.MinLatency < minLatency {
			minLatency = grStatistics.MinLatency
		}
		if grStatistics.MaxLatency > maxLatency {
			maxLatency = grStatistics.MaxLatency
		}
	}

	totalTime := time.Since(startAll)
	throughput := float64(totalTasks) / totalTime.Seconds()
	avgLatency := totalSumLatency / time.Duration(totalTasks)
	avgLatencyNs := float64(totalSumLatency.Nanoseconds()) / float64(totalTasks)
	varianceLatency := (totalSumLatencySquared / float64(totalTasks)) - (avgLatencyNs * avgLatencyNs)
	stdLatencyNs := math.Sqrt(varianceLatency)
	stdLatency := time.Duration(stdLatencyNs)

	// fmt.Println(totalSumLatency, time.Duration(totalTasks), totalTasks, avgLatency)
	// fmt.Println(float64(totalTasks), totalTime, totalTime.Seconds(), throughput)

	res := Result{Workload: config.Workload, GOMAXPROCS: config.GOMAXPROCS, Goroutines: config.Goroutines, TotalTasks: totalTasks, TotalTime:  totalTime, Throughput: throughput, AvgLatency: avgLatency, MinLatency: minLatency, MaxLatency: maxLatency, StdLatency: stdLatency}
	return res
}

func saveResultsToCSV(results []Result, fileName string) error {
	os.MkdirAll("results", os.ModePerm)
	file, fileError := os.Create(fileName)
	if fileError != nil {
		return fileError
	}
	writer := csv.NewWriter(file)
	defer file.Close()
	defer writer.Flush()

	header := []string{"Workload", "GOMAXPROCS", "Goroutines", "TotalTasks", "Throughput_tasks_per_sec", "TotalTime_µs", "AvgLatency_µs", "MinLatency_µs", "MaxLatency_µs", "StdLatency_µs"}

	writer.Write(header)

	for _, res := range results {
		row := []string{
			res.Workload,
			strconv.Itoa(res.GOMAXPROCS),
			strconv.Itoa(res.Goroutines),
			strconv.Itoa(res.TotalTasks),
			strconv.FormatFloat(float64(res.Throughput), 'f', 3, 64),
			strconv.FormatFloat(float64(res.TotalTime.Nanoseconds())/1e3, 'f', 3, 64),
			strconv.FormatFloat(float64(res.AvgLatency.Nanoseconds())/1e3, 'f', 3, 64),
			strconv.FormatFloat(float64(res.MinLatency.Nanoseconds())/1e3, 'f', 3, 64),
			strconv.FormatFloat(float64(res.MaxLatency.Nanoseconds())/1e3, 'f', 3, 64),
			strconv.FormatFloat(float64(res.StdLatency.Nanoseconds())/1e3, 'f', 3, 64),
		}

		writer.Write(row)
	}

	return nil
}


func main() {
	workloads := []string{"cpu", "mixed"}
	procsList := []int{1, 2, runtime.NumCPU()}
	goroutineCounts := []int{1, 2, 4, 8, 16, 32, 64}
	csvFileName := "results/experiment_results.csv"

	cpuTasksPerGor := 2000
	cpuInnerLoopIters := 10000
	mixedTasksPerGor := 2000
	mixedInnerLoopIters := 10000

	var results []Result
	fmt.Printf("Number of CPU detected: %d\n", runtime.NumCPU())

	for _, workload := range workloads {
		for _, procs := range procsList {
			for _, gr := range goroutineCounts {

				var tasksPerGor, innerLoopIters int
				if workload == "cpu" {
					tasksPerGor = cpuTasksPerGor
					innerLoopIters = cpuInnerLoopIters
				} else {
					tasksPerGor = mixedTasksPerGor
					innerLoopIters = mixedInnerLoopIters
				}

				config := ExperimentConfig{Workload: workload, GOMAXPROCS: procs, Goroutines: gr, TasksPerGor: tasksPerGor, InnerLoopIters: innerLoopIters}
				fmt.Printf("Running: workload=%s, GOMAXPROCS=%d, goroutines=%d\n", config.Workload, config.GOMAXPROCS, config.Goroutines)

				res := runExperiment(config)
				results = append(results, res)
			}
		}
	}

	fileError := saveResultsToCSV(results, csvFileName)
	if fileError != nil {
		fmt.Printf("Error in saving CSV: %v\n", fileError)
	} else {
		fmt.Printf("Results saved to %s\n", csvFileName)
	}
}
