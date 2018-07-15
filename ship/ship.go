package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tetration "github.com/remiphilippe/go-h4"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	ship := &Ship{}
	ship.Run()
}

// Ship runs to generate network traffic
type Ship struct {
	uuid        string
	config      Config
	credentials Credentials
	tetration   *tetration.H4
}

// Config holds the traffic generator settings
type Config struct {
	Clients []string `yaml:"clients"`
	Servers []int    `yaml:"servers"`
}

// Credentials holds the tetration API keys
type Credentials struct {
	Key    string `json:"api_key"`
	Secret string `json:"api_secret"`
}

// Annotation will be sent to Tetration to identify this ship
type Annotation struct {
	IP         string            `json:"ip"`
	Attributes map[string]string `json:"attributes"`
}

// NewAnnotation creates an initialized annotation
func NewAnnotation() *Annotation {
	var annotation Annotation
	annotation.Attributes = make(map[string]string)
	return &annotation
}

func (c *Credentials) getCredentials() *Credentials {

	jsonFile, err := ioutil.ReadFile("api_credentials.json")
	if err != nil {
		log.Fatalf("error opening #%v ", err)
	}
	err = json.Unmarshal(jsonFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func (c *Config) getConfig() *Config {

	yamlFile, err := ioutil.ReadFile("conf.yaml")
	if err != nil {
		log.Printf("error opening #%v ", err)
		yamlFile, err = ioutil.ReadFile("/etc/ship/conf.yaml")
		if err != nil {
			log.Printf("error opening #%v ", err)
		}
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

// Run the ship traffic generator
func (s *Ship) Run() {
	s.config.getConfig()

	s.setupTetration()

	log.Printf("ship starting")

	if os.Getenv("BOTTLE_STATS") == "true" {
		log.Println("stats will be logged to collector")
	}

	log.Printf("ship servers starting")
	go s.runServers()
	time.Sleep(2 * time.Second)
	log.Printf("ship clients starting")
	go s.runClients()
	select {}
}

func (s *Ship) runClients() {
	for _, address := range s.config.Clients {
		go s.client(address)
	}
}

func (s *Ship) runServers() {
	for _, port := range s.config.Servers {
		go s.server(port)
	}

}

func (s *Ship) logComplete(source string, target string) {
	if os.Getenv("BOTTLE_STATS") == "true" {
		resp, err := http.Get(fmt.Sprintf("http://svc-%s-stats:8080/log/complete/%s:%s", os.Getenv("BOTTLE_SCENARIO"), source, target))
		if err != nil {
			log.Printf("[client] failed to log a complete connection, err=%s", err)
			return
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

func (s *Ship) logFailed(source string, target string) {
	if os.Getenv("BOTTLE_STATS") == "true" {
		resp, err := http.Get(fmt.Sprintf("http://svc-%s-stats:8080/log/failed/%s:%s", os.Getenv("BOTTLE_SCENARIO"), source, target))
		if err != nil {
			log.Printf("[client] failed to log a failed connection, err=%s", err)
			return
		}

		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

func (s *Ship) client(address string) {
	rand.Seed(time.Now().Unix())
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		log.Fatalf("[client] %s is not a valid remote host", address)
	}

	connection := 1
	for {

		backends, err := net.LookupHost(host)
		if err != nil {
			log.Printf("[client] DNS lookup error, err=%s", err)
			time.Sleep(20 * time.Second)
			continue
		}

		backend := backends[rand.Intn(len(backends))]
		address = fmt.Sprintf("%s:%s", backend, port)
		log.Printf("[client] service %s resolved to %d hosts, picked %s", host, len(backends), backend)

		log.Printf("[client] connect %d to %s", connection, address)
		conn, err := net.DialTimeout("tcp", address, 10*time.Second)
		if err != nil {
			log.Printf("[client] connect error, err=%s", err)
			s.logFailed(os.Getenv("BOTTLE_SHIP"), host)
			time.Sleep(20 * time.Second)
			continue
		}
		connection++
		log.Printf("[client] connected to %s on %s", conn.RemoteAddr(), conn.LocalAddr())
		s.logComplete(os.Getenv("BOTTLE_SHIP"), host)
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		fmt.Fprintf(conn, "hello from "+hostname+"\r\n")
		_, err = bufio.NewReader(conn).ReadString('\n')
		conn.Close()
		log.Printf("[client] closed connection to %s on %s", conn.RemoteAddr(), conn.LocalAddr())
		time.Sleep(30 * time.Second)
	}
}

func (s *Ship) server(port int) {
	log.Printf("[server] serving on %d", port)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("[server] listen error, err=%s", err)
	}
	accepted := 0
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalf("[server] accept error, err=%s", err)
		}
		accepted++
		go s.handleConnection(conn)
		log.Printf("[server] connection %d accepted from %s to %s", accepted, conn.RemoteAddr(), conn.LocalAddr())
	}
}

func (s *Ship) handleConnection(conn net.Conn) {
	bufr := bufio.NewReader(conn)
	buf := make([]byte, 1024)

	for {
		readBytes, err := bufr.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("handle connection error, err=%s", err)
			}
			conn.Close()
			return
		}
		// log.Printf("<->\n%s", hex.Dump(buf[:readBytes]))
		conn.Write([]byte("server says: " + string(buf[:readBytes])))
	}
}

func (s *Ship) registerCleanup() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		log.Println("sensor cleanup function registered")
		<-c
		s.cleanup()
		os.Exit(1)
	}()
}

func (s *Ship) cleanup() {
	log.Println("finishing! will unregister sensor")
	s.annotateOnTearDown()
	err := s.tetration.Delete("/sensors/"+s.uuid, "")
	if err != nil && !strings.Contains(err.Error(), "Other Error (204)") {
		log.Fatalf("failed to unregister sensor, error=#%v", err)
	}
	log.Println("sensor unregistered")
}

func (s *Ship) annotate(annotation Annotation) {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Panicln("no interfaces found to annotate")
		return
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Printf("could not find any IP addresses to annotate on interface %s", i.Name)
			return
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			scope := os.Getenv("BOTTLE_SCOPE")
			annotation.IP = ip.String()
			payload, err := json.Marshal(annotation)
			if err != nil {
				log.Printf("could not create annotation payload #error=%v", err)
				return
			}
			_, err = s.tetration.Post("/inventory/tags/"+scope, string(payload))
			if err != nil {
				log.Printf("could not post annotation #error=%v", err)
				return
			}
			log.Printf("annotations saved to cluster (ip=%s)", ip.String())
		}
	}
}

func (s *Ship) annotateOnSetup() {
	annotation := NewAnnotation()
	annotation.Attributes["bottle"] = "true"
	annotation.Attributes["bottle_lifecycle"] = "active"
	annotation.Attributes["bottle_scenario"] = os.Getenv("BOTTLE_SCENARIO")
	annotation.Attributes["bottle_ship"] = os.Getenv("BOTTLE_SHIP")
	s.annotate(*annotation)
}

func (s *Ship) annotateOnTearDown() {
	annotation := NewAnnotation()
	annotation.Attributes["bottle_lifecycle"] = "terminated"
	s.annotate(*annotation)
}

func (s *Ship) setupTetration() {
	s.credentials.getCredentials()

	var uuid string

	for i := 1; i <= 12; i++ {
		result, err := ioutil.ReadFile("/usr/local/tet/sensor_id")
		if err != nil {
			log.Printf("attempt %d, no sensor uuid (check if sensor is running) error=#%v", i, err)
			time.Sleep(10 * time.Second)
		} else {
			uuid = string(result)
			break
		}
	}

	if uuid == "" {
		log.Fatalf("could not open sensor_id file")
	}

	if strings.Contains(string(uuid), "uuid-") {
		log.Fatalf("sensor is not registered %s", uuid)
	}

	log.Printf("sensor is registered with uuid=%s", uuid)

	s.uuid = uuid

	result, err := ioutil.ReadFile("/usr/local/tet/site.cfg")
	if err != nil {
		log.Fatalf("could not obtain site configuration, error=#%v", err)
	}

	configURL := strings.Split(string(result), "=")
	url := configURL[1]
	url = strings.Trim(url, "\n")
	url = strings.Trim(url, "\"")

	s.tetration = tetration.NewH4(url, s.credentials.Secret, s.credentials.Key, "/openapi/v1", false)

	_, err = s.tetration.GetSWAgents()
	if err != nil {
		log.Fatalf("failed reading sw agents (check provided API key has correct privilege) error=%v", err)
	}

	s.registerCleanup()

	s.annotateOnSetup()
}
