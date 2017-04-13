package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
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

func main() {
	// Parse flags
	flag.Parse()
	// Process start and end dates
	duration, _ := time.ParseDuration(*durationPtr)
	startTime := time.Now()
	endTime := startTime.Add(duration)
	if (*startDatePtr != "") && (*endDatePtr != "") {
		startTime, _ = time.Parse(shortForm, *startDatePtr)
		endTime, _ = time.Parse(shortForm, *endDatePtr)
	}
	// Create a CloudWatch service object
	svc := cloudwatch.New(session.New(), &aws.Config{Region: region})
	// Create WaitGroup
	var wg sync.WaitGroup
	// Send an empty parameter set to get all metrics in the region
	params := &cloudwatch.ListMetricsInput{}
	// Count how many metrics we get back
	totalMetrics := 0
	// Display arguments
	log.Println("Querying account for metrics with these settings:")
	log.Printf("\tRegion: %s\n", *region)
	log.Printf("\tStart Date: %s\n", startTime)
	log.Printf("\tEnd Date: %s\n", endTime)
	log.Printf("\tPeriod: %d\n", *period)
	log.Printf("\tSaving to: %s\n", *baseDir)
	// Get all pages of metrics
	log.Println("Searching for non-empty datapoints:")
	err := svc.ListMetricsPages(params, func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
		totalMetrics += len(page.Metrics)
		for _, metric := range page.Metrics {
			// Add to WaitGroup
			wg.Add(1)
			go getDataPoints(*metric, svc, &wg, startTime, endTime)
		}
		return true
	})
	if err != nil {
		log.Println(err)
	}

	// Print the page count
	wg.Wait()
	log.Printf("\nSearched %d metrics from %s to %s.\n", totalMetrics, startTime, endTime)
}

// Grabs datapoints from CloudWatch API and writes them to disk
func getDataPoints(metric cloudwatch.Metric, svc *cloudwatch.CloudWatch, wg *sync.WaitGroup, startTime, endTime time.Time) error {
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
	if err != nil {
		return err
	}
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
		if err != nil {
			return err
		}
		// Open/create file for writing/appending
		log.Printf(strings.Repeat(".", len(resp.Datapoints)))
		json, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(json)
		if err != nil {
			return err
		}
	}
	return nil
}
