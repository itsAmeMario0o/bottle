package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"

	yaml "gopkg.in/yaml.v2"
)

func main() {
	bottle := &Bottle{}
	bottle.Run()
}

// Bottle runs to generate network traffic
type Bottle struct {
	image    string
	scope    string
	scenario Scenario
}

// Scenario holds the desired traffic generation parameters
type Scenario struct {
	Name  string          `yaml:"name"`
	Ships map[string]Ship `yaml:"ships"`
}

// Ship holds the state of one traffic generator
type Ship struct {
	Replicas int      `yaml:"replicas"`
	Clients  []string `yaml:"clients"`
	Servers  []int    `yaml:"servers"`
}

// Run the bottle traffic generator
func (b *Bottle) Run() {
	scenarioFilename := flag.String("f", "scenario.yaml", "scenario file to deploy")
	image := flag.String("i", "bottle:latest", "container image to deploy")
	scope := flag.String("s", "Default", "scope to use when creating annotations")

	flag.Parse()

	b.image = *image
	b.scope = *scope
	b.scenario.getScenario(*scenarioFilename)
	b.create()
}

func (b *Bottle) create() {
	fmt.Printf("Deploying scenario \"%s\" with container image \"%s\"\n", b.scenario.Name, b.image)

	for name, ship := range b.scenario.Ships {
		fmt.Println("\nTier: " + name)
		fmt.Println(" replicas: " + strconv.Itoa(ship.Replicas))
		fmt.Println(" clients:  " + strconv.Itoa(len(ship.Clients)))
		for _, client := range ship.Clients {
			fmt.Println("  " + client)
		}
		fmt.Println(" servers:  " + strconv.Itoa(len(ship.Servers)))
		for _, server := range ship.Servers {
			fmt.Println("  " + strconv.Itoa(server))

		}
	}
}

func (s *Scenario) getScenario(filename string) *Scenario {

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("error opening scenario #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, s)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return s
}
