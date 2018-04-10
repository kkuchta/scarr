package main

import (
  "fmt"
  "os"
  "bytes"
  "text/template"
  "io/ioutil"
)

const ConfigTemplateString = 
`domain: "{{.domain}}"
name: "{{.name}}"
`

func generateConfig(domain string, name string) string{
  configTemplate := template.Must(template.New("config").Parse(ConfigTemplateString))
  buffer := &bytes.Buffer{}
  data := map[string]interface{}{
    "name": name,
    "domain": domain}
  check(configTemplate.Execute(buffer, data))

  return buffer.String()
}

func writeFile(path string, content string){
  check(ioutil.WriteFile(path, []byte(content), 0644))
}

func runInit(domain string, name string){
  fmt.Println("Initting")
  err := os.Mkdir(name, 0755)
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  config := generateConfig(domain, name)
  writeFile(name + "/scarr.yml", config)
}
