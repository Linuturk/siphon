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

var (
	region  = flag.String("region", "us-east-1", "AWS Region to siphon metrics.")
	period  = flag.Int64("period", 300, "Period is the length of time associated with a specific CloudWatch statistic.")
	baseDir = flag.String("baseDir", "/tmp/cloudwatch", "Base directory to store datapoint file structure.")
	start   = flag.String("start", "", "Start date for datapoint collection. (ex. 2006-Jan-02)")
	end     = flag.String("end", "", "End date for datapoint collection. (ex. 2006-Jan-02)")
)

func main() {

	// Parse flags
	flag.Parse()

	// Defaults
	now := time.Now()
	later := now.Add(24 * time.Hour)

	// Handle the start/end strings
	endTime, err := time.Parse("2006-Jan-02", *end)
	if err != nil {
		log.Println(err)
		log.Printf("Invalid or no end set. Using %s as end time.", now)
		endTime = now
	}
	startTime, err := time.Parse("2006-Jan-02", *start)
	if err != nil {
		log.Println(err)
		log.Printf("Invalid or no start set. Using %s as start time.", later)
		startTime = later
	}

	// Display arguments
	log.Println("Querying account for metrics with these settings:")
	log.Printf("\tRegion: %s\n", *region)
	log.Printf("\tStart Date: %s\n", startTime)
	log.Printf("\tEnd Date: %s\n", endTime)
	log.Printf("\tPeriod: %d\n", *period)
	log.Printf("\tSaving to: %s\n", *baseDir)

	// Create a CloudWatch service object
	svc := cloudwatch.New(session.New(), &aws.Config{Region: region})

	// Create WaitGroup
	var wg sync.WaitGroup

	// Send an empty parameter set to get all metrics in the region
	params := &cloudwatch.ListMetricsInput{}

	// Count how many metrics we get back
	totalMetrics := 0

	// Get all pages of metrics
	log.Println("Searching for non-empty datapoints:")
	err = svc.ListMetricsPages(params,
		func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
			totalMetrics += len(page.Metrics)
			for _, metric := range page.Metrics {
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
	log.Printf("Searched %d metrics from %s to %s.\n", totalMetrics, startTime, endTime)
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
