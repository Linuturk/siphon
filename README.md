# siphon
Siphon off and archive CloudWatch Metrics.

```
Usage of siphon:
  -baseDir string
    	Base directory to store datapoint file structure. (default "/tmp/cloudwatch")
  -endDate string
    	End date for datapoint collection. (default "2016-Jan-20")
  -period int
    	Period is the length of time associated with a specific Amazon CloudWatch statistic. (default 300)
  -region string
    	AWS Region to siphon metrics. (default "us-east-1")
  -startDate string
    	Start date for datapoint collection. (default "2016-Jan-18")
```

# ulimit

Errors can appear related to opening files or sockets if your user's ulimit is too low for the number of metrics you are saving. Increase your ulimit to avoid this problem.

```
ulimit -n 10000
```
