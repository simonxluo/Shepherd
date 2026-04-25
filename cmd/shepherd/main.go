package main

import (
	"github.com/shepherd-project/shepherd/Shepherd/internal/cli"
)

var (
	Version   string
	BuildTime string
	GitCommit string
)

func main() {
	cli.Version = Version
	cli.BuildTime = BuildTime
	cli.GitCommit = GitCommit
	cli.Execute()
}
