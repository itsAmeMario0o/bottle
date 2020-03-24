package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
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
	concount    int
	config      Config
	credentials Credentials
	tetration   *tetration.H4
}

// Config holds the traffic generator settings
type Config struct {
	Clients  []string  `yaml:"clients"`
	Servers  []int     `yaml:"servers"`
	Tags     []Tag     `yaml:"tags"`
	Packages []Package `yaml:"packages"`
	UI       UI
}

// Tag holds a key value pair applied as annotations in TA
type Tag struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// Package holds the details of an RPM package that will be "installed"
type Package struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Release string `yaml:"release"`
}

// UI is used to create a simple webpage
type UI struct {
	Title      string `yaml:"title"`
	Body       string `yaml:"body"`
	Image      string `yaml:"image"`
	Favicon    string `yaml:"favicon"`
	ClientAddr string
	Hostname   string
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

	envAPIKey, envAPISecret := os.Getenv("BOTTLE_API_KEY"), os.Getenv("BOTTLE_API_SECRET")

	if envAPIKey != "" && envAPISecret != "" {
		c.Key = envAPIKey
		c.Secret = envAPISecret
		log.Println("Using API key provided in environment variables")
		return c
	}

	log.Println("Opening API key expected to be provided in api_credentials.json")
	jsonFile, err := ioutil.ReadFile("api_credentials.json")
	if err != nil {
		log.Fatalf("error opening #%v ", err)
	}
	err = json.Unmarshal(jsonFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	log.Println("Using API key provided in api_credentials.json")
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

	s.setupPackages()

	s.setupTetration()

	log.Printf("ship starting")

	if os.Getenv("BOTTLE_STATS") == "true" {
		log.Println("stats will be logged to collector")
	}

	if os.Getenv("BOTTLE_SENSOR") == "false" {
		log.Println("no sensor will be utilised")
	}

	log.Printf("ship ui starting")
	go s.runUI()
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

func (s *Ship) runUI() {
	ui := s.config.UI

	if (ui != UI{}) {
		log.Println("ui requested, starting")

		tmpl := template.Must(template.ParseFiles("ui.html"))

		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		s.config.UI.Hostname = hostname

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			s.config.UI.ClientAddr = r.RemoteAddr
			tmpl.Execute(w, s.config.UI)
		})

		http.ListenAndServe(":80", nil)
	}
}

func (s *Ship) shortClient(host string, port string) {
	for {

		backends, err := net.LookupHost(host)
		if err != nil {
			log.Printf("[client] DNS lookup error, err=%s", err)
			time.Sleep(20 * time.Second)
			continue
		}

		backend := backends[rand.Intn(len(backends))]
		address := fmt.Sprintf("%s:%s", backend, port)
		log.Printf("[client] service %s resolved to %d hosts, picked %s", host, len(backends), backend)

		log.Printf("[client] connect %d to %s", s.concount, address)
		conn, err := net.DialTimeout("tcp", address, 10*time.Second)
		if err != nil {
			log.Printf("[client] connect error, err=%s", err)
			s.logFailed(os.Getenv("BOTTLE_SHIP"), host)
			time.Sleep(20 * time.Second)
			continue
		}
		s.concount++
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

func (s *Ship) longClient(host string, port string) {
	for {

		backends, err := net.LookupHost(host)
		if err != nil {
			log.Printf("[long-client] DNS lookup error, err=%s", err)
			time.Sleep(20 * time.Second)
			continue
		}

		backend := backends[rand.Intn(len(backends))]
		address := fmt.Sprintf("%s:%s", backend, port)
		log.Printf("[long-client] service %s resolved to %d hosts, picked %s", host, len(backends), backend)

		log.Printf("[long-client] connect %d to %s", s.concount, address)
		conn, err := net.DialTimeout("tcp", address, 10*time.Second)
		if err != nil {
			log.Printf("[long-client] connect error, err=%s", err)
			s.logFailed(os.Getenv("BOTTLE_SHIP"), host)
			time.Sleep(20 * time.Second)
			continue
		}
		s.concount++
		log.Printf("[long-client] connected to %s on %s", conn.RemoteAddr(), conn.LocalAddr())
		s.logComplete(os.Getenv("BOTTLE_SHIP"), host)
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		for {
			fmt.Fprintf(conn, "long hello from "+hostname+"\r\n")
			_, err = bufio.NewReader(conn).ReadString('\n')
			time.Sleep(time.Minute)
		}
	}
}

func (s *Ship) client(address string) {
	rand.Seed(time.Now().Unix())
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		log.Fatalf("[client] %s is not a valid remote host", address)
	}

	go s.shortClient(host, port)

	s.longClient(host, port)
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
	signals := make(chan os.Signal)
	done := make(chan bool, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGUSR1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("sensor cleanup function registered")
		<-signals
		s.cleanup()
		done <- true
	}()
	<-done
	log.Println("process will terminate now")
	os.Exit(0)
}

func (s *Ship) cleanup() {
	log.Println("finishing!")
	s.annotateOnTearDown()
	log.Println("removed annotations")
	if os.Getenv("BOTTLE_SENSOR") != "false" {
		err := s.tetration.Delete("/sensors/"+s.uuid, "")
		if err != nil && !strings.Contains(err.Error(), "Other Error (204)") {
			log.Fatalf("failed to unregister sensor, error=#%v", err)
		}
		log.Println("sensor unregistered")
	}
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

	for _, tag := range s.config.Tags {
		log.Printf("creating custom tag(%s=%s)", tag.Key, tag.Value)
		annotation.Attributes[tag.Key] = tag.Value
	}

	s.annotate(*annotation)
}

func (s *Ship) annotateOnTearDown() {
	annotation := NewAnnotation()
	annotation.Attributes["bottle_lifecycle"] = "terminated"
	s.annotate(*annotation)
}

func (s *Ship) setupTetration() {
	s.credentials.getCredentials()

	var url string
	if os.Getenv("BOTTLE_SENSOR") != "false" {
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
		url = configURL[1]
		url = strings.Trim(url, "\n")
		url = strings.Trim(url, "\"")
	} else {
		url = os.Getenv("BOTTLE_URL")
	}

	s.tetration = tetration.NewH4(url, s.credentials.Secret, s.credentials.Key, "/openapi/v1", false)

	_, err := s.tetration.GetSWAgents()
	if err != nil {
		log.Fatalf("failed reading sw agents (check provided API key has correct privilege) error=%v", err)
	}

	go s.registerCleanup()

	s.annotateOnSetup()
}

func (s *Ship) setupPackages() {
	const packageTemplate = `
Summary: package generated by bottle
Name: {{.Name}}
Version: {{.Version}}
Release: {{.Release}}
License: Public
Group: Applications/System
%description
package generated by bottle
%files`
	templateWriter := template.Must(template.New("packageTemplate").Parse(packageTemplate))

	// loop each package, create blank rpm spec, create the rpm, then install the rpm
	for _, p := range s.config.Packages {
		log.Printf("creating and installing package: %s", p.packageName())

		// create the RPM spec file
		filename := fmt.Sprintf("/tmp/package-%s.spec", p.packageName())
		f, err := os.Create(filename)
		if err != nil {
			log.Fatalf("failed to create temporary package spec file: error=%v ", err)
		}
		err = templateWriter.Execute(f, p)
		if err != nil {
			log.Fatalf("failed to write temporary package spec file: error=%v ", err)
		}
		f.Close()
		log.Printf("created temporary package spec file at %s", filename)

		// create the rpm package
		err = exec.Command("rpmbuild", "-bb", filename).Run()
		if err != nil {
			log.Fatalf("failed to create temporary rpm error=%v", err)
		}
		log.Printf("temporary rpm created")

		// install the rpm package
		// the strange path is generated by the rpmbuild command.
		// rpmbuild likes to place the output in your home directory, "/root" in this case.
		rpmFilename := fmt.Sprintf("/root/rpmbuild/RPMS/x86_64/%s.x86_64.rpm", p.packageName())
		err = exec.Command("rpm", "-Uvh", rpmFilename).Run()
		if err != nil {
			log.Fatalf("failed to install temporary rpm error=%v", err)
		}
		log.Printf("temporary rpm package %s installed", p.packageName())
	}
}

// returns the package name in the format "name-version-release"
func (p *Package) packageName() string {
	return fmt.Sprintf("%s-%s-%s", p.Name, p.Version, p.Release)
}
