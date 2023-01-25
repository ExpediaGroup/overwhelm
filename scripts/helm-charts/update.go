package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	d, err := os.ReadFile("charts/tmp.yaml")
	if err != nil {
		panic(err)
	}

	data := string(d)
	//if condition string patch. The kustomize created a sub element to spec element since we cant add this block to the spec key directly. This bit
	//will remove the sub element and indent this block back to the spec element

	ifString := regexp.MustCompile("{{- if(.|\n)+{{- end }}").FindString(data)
	ifIndentPatch := regexp.MustCompile("\n\\s\\s").ReplaceAllString(ifString, "\n")
	data = regexp.MustCompile("\"\": \\|-\\s+").ReplaceAllString(data, "")
	data = regexp.MustCompile("{{- if(.|\n)+({{- end }})").ReplaceAllString(data, ifIndentPatch)

	data = strings.ReplaceAll(data, "'{{", "{{")
	data = strings.ReplaceAll(data, "}}'", "}}")
	data = strings.ReplaceAll(data, "{{ toYaml", "\n{{ toYaml")
	data = regexp.MustCompile("{{\\s+").ReplaceAllString(data, "{{ ")
	data = regexp.MustCompile("\\s+}}").ReplaceAllString(data, " }}")

	if err = os.WriteFile("charts/tmp.yaml", []byte(data), 0644); err != nil {
		panic(err)
	}

	c, err := os.ReadFile("charts/overwhelm/Chart.yaml")
	if err != nil {
		panic(err)
	}
	chart := string(c)
	chartVersion := regexp.MustCompile("(version: )(.+)(\n)").FindSubmatch(c)[2]
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("The current chart version is %s. Provide the new version in the form of x.x.x if applicable or type NA if not upgrade is needed \n", chartVersion)
	scanner.Scan()
	text := scanner.Text()
	if strings.ToLower(text) != "na" {
		fmt.Printf("The new version will be version: %s. Type Yes to confirm: ", text)
		scanner.Scan()
		confirm := scanner.Text()
		if strings.ToLower(confirm) != "yes" {
			fmt.Printf("Ok, abandoning chart upgrade \n")
		} else {
			newVersion := "version: " + text
			chart = strings.ReplaceAll(chart, "version: "+string(chartVersion), newVersion)
			if err = os.WriteFile("charts/overwhelm/Chart.yaml", []byte(chart), 0644); err != nil {
				panic(err)
			}
			fmt.Printf("Chart version upgraded to %s \n", text)
		}
	}
}
