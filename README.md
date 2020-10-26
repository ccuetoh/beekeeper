
[![Go Report Card][go-report-shield]][go-report-url]
[![MIT License][license-shield]][license-url]
[![Documentation][docs-shield]][docs-url]

<!-- PROJECT LOGO -->
<br />
<p align="center">
  <a href="https://github.com/CamiloHernandez/beekeeper">
    <img src="https://beekeeper.dev/logo.svg" alt="Logo" width="80" height="80">
  </a>

  <h3 align="center">Beekeeper</h3>

  <p align="center">
    Batteries-included cluster computing for Go
    <br />
    <a href="https://beekeeper.dev/documentation"><strong>Explore the docs »</strong></a>
  </p>
</p>

<!-- TABLE OF CONTENTS -->
## Table of Contents
* [About](#about)
* [Installation](#installation)
	* [Installing the Go library](#installing-the-go-library)
	* [Installing the CLI tool](#installing-the-cli-tool)
* [Usage](#usage)
	* [Starting a Worker](#starting-a-worker)
	* [Creating a Job](#creating-a-job)
	* [Running a Task](#running-a-task)
	* [Further reading](#further-reading)
* [Contributing](#contributing)
* [License](#license)

<!-- ABOUT -->
## About
**This project is currently in preview and is not yet intended for production use. Breaking changes could be made.**

Beekeeper is a batteries-included cluster computing solution written in Go. Be it a homemade cluster with your old computers or an enterprise-grade deployment, Beekeeper makes it quick and painless.

The idea behind Beekeeper is simple: Take a bunch of computers and make them work together for a faster and more reliable system (yay, teamwork!).

<!-- GETTING STARTED -->
## Installation
Beekeeper is split into two installs. The library is used to create new jobs and distribute them among the nodes while the CLI tools will help you get your nodes up and running.

### Installing the Go library
To install the Go library run the following command in the command line:
```bash
go get github.com/CamiloHernandez/beekeeper/lib
```

### Installing the CLI tool
The command-line tool can be installed using go get as follows:
```bash
go get github.com/CamiloHernandez/beekeeper/bee
```

<!-- Usage -->
## Usage

**Drop by our [Quickstart page](https://beekeeper.dev/documentation/quickstart) for an in-depth onboarding!**

### Starting a Worker
To run a worker server execute `bee start worker` in the command prompt. This command won't do anything until we create and send a job to be worked on.

### Creating a Job
To create a new Job we need to define a function implementing the `func (t *beekeeper.Task)` prototype. The Task struct will hold the information needed to run the Job, such as additional data and fields to return our result.

As an example we'll cluster a job that finds a bunch of primes numbers and returns them to the primary node:
```go 
func RandomPrime(t *beekeeper.Task) {
	var primes []int64
	r := rand.New(rand.NewSource(time.Now().Unix()))

	for len(primes) < 10000 {
		n := r.Int63n(10000000000000000)

		if isPrime(n) {
			primes = append(primes, n)
		}
	}

	t.Returns["primes"] = primes
}
  
func isPrime(n int64) bool {  
   return big.NewInt(n).ProbablyPrime(0)  
}
```

### Running a Task
To run a task first we need primary server to handle our task. This is blocking, so we run it inside a goroutine.
```go
go func() {  
   err := beekeeper.StartPrimary()  
   if err != nil{  
      panic(err)  
   }  
}()
```
Now we need a list of the available nodes in our network. To do this we use `ScanLocal`.
```go
workers, err := beekeeper.ScanLocal(beekeeper.DefaultScanTime)  
if err != nil{  
   panic(err)
}
```
`workers` will be a slice containing the workers we are about to use. Since our job is not yet present on our nodes we need to distribute it.
```go
err = workers.DistributeJob("github.com/user/myFirstCluster", "RandomPrime")  
if err != nil{  
   panic(err)
}
```
All is left to do is create a new Task and send it to the workers. Since we are not going to send any arguments we'll use an empty Task.
```go
task := beekeeper.NewTask()

results, err := workers.Execute(task)  
if err != nil{  
   panic(err) 
}
```
The execution will now block until all nodes have finished their task. Now we can see the results.
```go
var primes []int64  
for _, result := range results{  
   print(result.Task.Returns["primes"].[]int64)
}
```

### Further reading
You can check out the official [documentation](https://beekeeper.dev/documentation) to learn more about the advanced features of Beekeeper.

<!-- CONTRIBUTING -->
## Contributing
[![Contributor Covenant][covenant-shield]][covenant-url]
Contributions are always welcome! If you want to help Beekeeper please create a fork of this repository, make your changes to your fork and do a Pull Request. Keep in mind that suggestions must have a strong case for their addition, and must keep to the structure and quality of the code.

<!-- LICENSE -->
## License
Beekeeper is distributed under the MIT License as free and open-source. See the `LICENSE`file for more information.

<!-- MARKDOWN LINKS -->
[go-report-shield]: https://goreportcard.com/badge/github.com/CamiloHernandez/beekeeper
[go-report-url]: https://goreportcard.com/report/github.com/CamiloHernandez/beekeeper
[license-shield]: https://img.shields.io/github/license/CamiloHernandez/beekeeper
[license-url]: https://github.com/CamiloHernandez/beekeeper/blob/master/LICENSE
[docs-shield]: https://godoc.org/github.com/CamiloHernandez/beekeeper?status.svg
[docs-url]: https://godoc.org/github.com/CamiloHernandez/beekeeper

[covenant-shield]: https://img.shields.io/badge/Contributor%20Covenant-v2.0-green
[covenant-url]: https://github.com/CamiloHernandez/beekeeper/blob/master/CODE_OF_CONDUCT.md
