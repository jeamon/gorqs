# gorqs

`gorqs` means *Go Runnable Queue Service*. This is a multi-features go-based concurrent-safe library to **queue & execute** jobs and records their execution result. You can start the Queue service into synchronous or asynchronous mode.
The mode defines wether each added job should be processed *synchronously* or *asynchronously*. Be aware that adding a job to the Queue system is always a non-blocking operation and returns the job id on success. 

## Features

`gorqs.New(gorqs.MODE_SYNC | gorqs.TRACK_JOBS)` or `gorqs.New(gorqs.MODE_ASYNC | gorqs.TRACK_JOBS)` method provides a Queue object which implements the `Queuer` interface with below actions.

| Action | Description |
|:------ | :-------------------------------------- |
| Start(context.Context) error | starts the jobs queue |
| Stop(context.Context) error | stops the jobs queue |
| Push(context.Context, Runner) (int64, error) | adds a job to the queue asynchronously |
| Result(context.Context, int64) error | gets result execution of given job |
| Clear | delete all jobs results records |

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
