package main

import "gopkg.in/cheggaaa/pb.v1"

type ESHSessionConfig struct {
	Name string
	Hostname string
	Port string
	Username string
	Password string
	KeyPath string
	IsCurrentSession bool
	WorkingDir string
}

type ProgressTracker struct{
	Length int64
	ProgressInt int
	Name string
	Progress *pb.ProgressBar
}
