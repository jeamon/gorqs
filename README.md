# gorqs

[![Build Status](https://github.com/jeamon/gorqs/actions/workflows/tests.yml/badge.svg?branch=main)](https://github.com/jeamon/gorqs/actions)
[![godoc](https://godoc.org/github.com/jeamon/gorqs?status.svg)](https://godoc.org/github.com/jeamon/gorqs)
[![Go Report Card](https://goreportcard.com/badge/github.com/jeamon/gorqs)](https://goreportcard.com/report/github.com/jeamon/gorqs)
[![codecov](https://codecov.io/gh/jeamon/gorqs/graph/badge.svg?token=AKQ6PV9N90)](https://codecov.io/gh/jeamon/gorqs)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/jeamon/gorqs)
[![MIT License](https://img.shields.io/github/license/jeamon/gorqs)](https://github.com/jeamon/gorqs/blob/main/LICENSE)

`gorqs` means *Go Runnable Queue Service*. This is a multi-features go-based concurrent-safe library to **queue & execute** jobs and records their execution result. You can start the Queue service into synchronous or asynchronous mode.
The mode defines wether each added job should be processed *synchronously* or *asynchronously*. Be aware that adding a job to the Queue system is always a non-blocking operation and returns the job id on success. 

## Features

`gorqs.New(gorqs.SyncMode | gorqs.TrackJobs)` or `gorqs.New(gorqs.AsyncMode | gorqs.TrackJobs)` method provides a Queue object which implements the `Queuer` interface with below actions.

| Action | Description |
|:------ | :-------------------------------------- |
| Start(context.Context) error | starts the jobs queue |
| Stop(context.Context) error | stops the jobs queue |
| Push(context.Context, Runner) (int64, error) | adds a job to the queue asynchronously |
| Fetch(context.Context, int64) error | gets result execution of given job |
| Clear | delete all jobs results records |
| IsRunning() bool | provides queue service status |

*An acceptable runnable job should implement the `Runner` interface defined as below :*

```go
type Runner interface {
	Run() error
}
```

## Installation

Just import the `gorqs` library as external package to start using it into your project. There are some examples into the *examples* folder. 

**[Step 1] -** Download the package

```shell
$ go get github.com/jeamon/gorqs
```


**[Step 2] -** Import the package into your project

```shell
$ import "github.com/jeamon/gorqs"
```


**[Step 3] -** Optional: Clone the library to run some examples

```shell
$ git clone https://github.com/jeamon/gorqs.git
$ cd gorqs
$ go run examples/sync-mode/example.go
$ go run examples/async-mode/example.go
```

## Contact

Feel free to [reach out to me](https://blog.cloudmentor-scale.com/contact) before any action. Feel free to connect on [Twitter](https://twitter.com/jerome_amon) or [linkedin](https://www.linkedin.com/in/jeromeamon/)
