package main

import (
  "fmt"
  "os"
)

func generateConfig(domain string, name string) string{
  return `
  foo
  bar
  `
}

func runInit(domain string, name string){
  fmt.Println("Initting")
  err := os.Mkdir(name, 0755)
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  fmt.Println(generateConfig(domain, name))
}
