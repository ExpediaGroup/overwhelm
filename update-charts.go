package main

import (
	"os"
	"strings"
)


func main() {
	d, err := os.ReadFile(".charts/tmp.yaml")
	if err != nil {
		panic(err)
	}
	data := string(d)
	data = strings.ReplaceAll(data, "{{", "\\{\\{")
	data = strings.ReplaceAll(data, "<GOTEMPLATE>", "{{")
	data = strings.ReplaceAll(data, "</GOTEMPLATE>", "}}")
	data = strings.ReplaceAll(data, "{{ toYaml", "\n{{ toYaml")
	if err = os.WriteFile(".charts/tmp.yaml", []byte(data), 0644); err != nil {
		panic(err)
	}
}
