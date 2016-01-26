# siphon
Siphon off and archive CloudWatch Metrics.

```
$ siphon -h
Usage of siphon:
  -baseDir string
    	Base directory to store datapoint file structure. (default "/tmp/cloudwatch")
  -duration string
    	Subtract duration from Now for the metric search. (default "24h")
  -endDate string
    	End date for datapoint collection. (ex. 2006-Jan-02)
  -period int
    	Period is the length of time associated with a specific CloudWatch statistic. (default 300)
  -region string
    	AWS Region to siphon metrics. (default "us-east-1")
  -startDate string
    	Start date for datapoint collection. (ex. 2006-Jan-02)
```

# installation

```
go get -u github.com/linuturk/siphon
```

# ulimit

Errors can appear related to opening files or sockets if your user's ulimit is too low for the number of metrics you are saving. Increase your ulimit to avoid this problem.

```
ulimit -n 10000
```
