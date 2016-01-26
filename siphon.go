package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"os"
	"sync"
	"time"
)

// Dateform
const shortForm string = "2006-Jan-02"

var (
	// Parse command line options
	region       = flag.String("region", "us-east-1", "AWS Region to siphon metrics.")
	period       = flag.Int64("period", 300, "Period is the length of time associated with a specific CloudWatch statistic.")
	baseDir      = flag.String("baseDir", "/tmp/cloudwatch", "Base directory to store datapoint file structure.")
	startDatePtr = flag.String("startDate", "", "Start date for datapoint collection. (ex. 2006-Jan-02)")
	endDatePtr   = flag.String("endDate", "", "End date for datapoint collection. (ex. 2006-Jan-02)")
	durationPtr  = flag.String("duration", "24h", "Subtract duration from Now for the metric search.")
)

// Convenience function for error checking
func check(e error) {
	if e != nil {
		fmt.Println(e.Error())
	}
}

func main() {
	// Parse flags
	flag.Parse()
	// Process start and end dates
	duration, _ := time.ParseDuration(*durationPtr)
	var startTime, endTime time.Time
	if (*startDatePtr != "") && (*endDatePtr != "") {
		startTime, _ = time.Parse(shortForm, *startDatePtr)
		endTime, _ = time.Parse(shortForm, *endDatePtr)
	} else {
		startTime = time.Now()
		endTime = startTime.Add(duration)
	}
	// Create a CloudWatch service object
	var svc = cloudwatch.New(session.New(), &aws.Config{Region: region})
	// Create WaitGroup
	var wg sync.WaitGroup
	// Send an empty parameter set to get all metrics in the region
	params := &cloudwatch.ListMetricsInput{}
	// Count how many metrics we get back
	totalMetrics := 0
	// Display arguments
	fmt.Println("Querying account for metrics with these settings:")
	fmt.Printf("Region: %s\n", *region)
	fmt.Printf("Start Date: %s\n", startTime)
	fmt.Printf("End Date: %s\n", endTime)
	fmt.Printf("Period: %d\n", *period)
	fmt.Printf("Saving to: %s\n", *baseDir)
	// Get all pages of metrics
	err := svc.ListMetricsPages(params, func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
		totalMetrics += len(page.Metrics)
		for _, metric := range page.Metrics {
			// Add to WaitGroup
			wg.Add(1)
			go getDataPoints(*metric, svc, &wg, startTime, endTime)
		}
		return true
	})
	check(err)

	// Print the page count
	wg.Wait()
	fmt.Printf("Searched %d metrics from %s to %s.\n", totalMetrics, startTime, endTime)
}

// Grabs datapoints from CloudWatch API and writes them to disk
func getDataPoints(metric cloudwatch.Metric, svc *cloudwatch.CloudWatch, wg *sync.WaitGroup, startTime, endTime time.Time) {
	// Signal WaitGroup
	defer wg.Done()
	// Set search parameters
	params := &cloudwatch.GetMetricStatisticsInput{
		EndTime:    aws.Time(endTime),
		MetricName: metric.MetricName,
		Namespace:  metric.Namespace,
		Period:     period,
		StartTime:  aws.Time(startTime),
		Statistics: []*string{
			aws.String("SampleCount"),
			aws.String("Average"),
			aws.String("Sum"),
			aws.String("Minimum"),
			aws.String("Maximum"),
		},
		Dimensions: metric.Dimensions,
		Unit:       aws.String("Seconds"),
	}
	// use metric to query GetMetricStatistics
	resp, err := svc.GetMetricStatistics(params)
	check(err)
	// Check for data points
	if resp.Datapoints != nil {
		// Build directory structure
		var filename string
		var dirname string
		if metric.Dimensions != nil {
			filename = fmt.Sprintf("%s/%s/%s/%s", *baseDir, *metric.Namespace, *metric.Dimensions[0].Name, *metric.Dimensions[0].Value)
			dirname = fmt.Sprintf("%s/%s/%s", *baseDir, *metric.Namespace, *metric.Dimensions[0].Name)
		} else {
			filename = fmt.Sprintf("%s/%s/%s", *baseDir, *metric.Namespace, *metric.MetricName)
			dirname = fmt.Sprintf("%s/%s", *baseDir, *metric.Namespace)
		}

		// Create any missing directories
		err := os.MkdirAll(dirname, 0755)
		check(err)
		// Open/create file for writing/appending
		fmt.Printf("Writing %v data points to %v\n", len(resp.Datapoints), filename)
		json, err := json.Marshal(resp)
		check(err)
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		check(err)
		defer f.Close()
		_, err = f.Write(json)
	}
}
