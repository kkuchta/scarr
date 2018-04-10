package main

import (
  "fmt"
  "os"
  "flag"

  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/s3"
)

func check(e error) {
  if e != nil {
    panic(e)
  }
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func main(){
  initCommand := flag.NewFlagSet("init", flag.ExitOnError)
  deployCommand := flag.NewFlagSet("deploy", flag.ExitOnError)

  domainPtr := initCommand.String("domain", "", "the domain this site will live at")
  namePtr := initCommand.String("name", "", "the name of this project")

  if len(os.Args) < 2 {
    fmt.Println("Missing command")
    os.Exit(1)
  }

  command := os.Args[1]

  switch command {
  case "-h":
    fmt.Println("Available commands:\n  init\n  deploy")
  case "init":
    initCommand.Parse(os.Args[2:])
  case "deploy":
    deployCommand.Parse(os.Args[2:])
  default:
    fmt.Println("Unknown command ", command)
    flag.PrintDefaults()
    os.Exit(1)
  }

  if initCommand.Parsed() {
    runInit(*domainPtr, *namePtr)
    fmt.Println("init parsed", *domainPtr, *namePtr)
  } else if deployCommand.Parsed() {
    fmt.Println("deploying")
  }

  os.Exit(0)

  sess := session.Must(session.NewSession(&aws.Config{
    Region: aws.String("us-west-2") }))
  s3Session := s3.New(sess)
  result, err := s3Session.ListBuckets(nil)

  if err != nil {
    exitErrorf("Unable to list buckets, %v", err)
  }

  fmt.Println("result", result)

	fmt.Println("done")
}
