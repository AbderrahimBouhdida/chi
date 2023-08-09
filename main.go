package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	chclient "github.com/AbderrahimBouhdida/chi/cl"
	chshare "github.com/AbderrahimBouhdida/chi/share"
	"github.com/AbderrahimBouhdida/chi/share/cos"
	chserver "github.com/AbderrahimBouhdida/chi/srv"
)

var help = ``

func main() {

	version := flag.Bool("version", false, "")
	v := flag.Bool("v", false, "")
	flag.Bool("help", false, "")
	flag.Bool("h", false, "")
	flag.Usage = func() {}
	flag.Parse()

	if *version || *v {
		fmt.Println(chshare.BuildVersion)
		os.Exit(0)
	}

	args := flag.Args()

	subcmd := ""
	if len(args) > 0 {
		subcmd = args[0]
		args = args[1:]
	}

	switch subcmd {
	case "srv":
		srv(args)
	case "cl":
		cl(args)
	default:
		fmt.Print(help)
		os.Exit(0)
	}
}

var commonHelp = ``

func generatePidFile() {
	pid := []byte(strconv.Itoa(os.Getpid()))
	if err := ioutil.WriteFile("chisel.pid", pid, 0644); err != nil {
		log.Fatal(err)
	}
}

var serverHelp = `` + commonHelp

func srv(args []string) {

	flags := flag.NewFlagSet("srv", flag.ContinueOnError)

	config := &chserver.Config{}
	flags.StringVar(&config.KeySeed, "key", "", "")
	flags.StringVar(&config.AuthFile, "authfile", "", "")
	flags.StringVar(&config.Auth, "auth", "", "")
	flags.DurationVar(&config.KeepAlive, "keepalive", 25*time.Second, "")
	flags.StringVar(&config.Proxy, "proxy", "", "")
	flags.StringVar(&config.Proxy, "backend", "", "")
	flags.BoolVar(&config.Socks5, "socks5", false, "")
	flags.BoolVar(&config.Reverse, "reverse", false, "")
	flags.StringVar(&config.TLS.Key, "tls-key", "", "")
	flags.StringVar(&config.TLS.Cert, "tls-cert", "", "")
	flags.Var(multiFlag{&config.TLS.Domains}, "tls-domain", "")
	flags.StringVar(&config.TLS.CA, "tls-ca", "", "")

	host := flags.String("host", "", "")
	p := flags.String("p", "", "")
	port := flags.String("port", "", "")
	pid := flags.Bool("pid", false, "")
	verbose := flags.Bool("v", false, "")

	flags.Usage = func() {
		fmt.Print(serverHelp)
		os.Exit(0)
	}
	flags.Parse(args)

	if *host == "" {
		*host = os.Getenv("HOST")
	}
	if *host == "" {
		*host = "0.0.0.0"
	}
	if *port == "" {
		*port = *p
	}
	if *port == "" {
		*port = os.Getenv("PORT")
	}
	if *port == "" {
		*port = "8080"
	}
	if config.KeySeed == "" {
		config.KeySeed = os.Getenv("CHISEL_KEY")
	}
	s, err := chserver.NewServer(config)
	if err != nil {
		log.Fatal(err)
	}
	s.Debug = *verbose
	if *pid {
		generatePidFile()
	}
	go cos.GoStats()
	ctx := cos.InterruptContext()
	if err := s.StartContext(ctx, *host, *port); err != nil {
		log.Fatal(err)
	}
	if err := s.Wait(); err != nil {
		log.Fatal(err)
	}
}

type multiFlag struct {
	values *[]string
}

func (flag multiFlag) String() string {
	return strings.Join(*flag.values, ", ")
}

func (flag multiFlag) Set(arg string) error {
	*flag.values = append(*flag.values, arg)
	return nil
}

type headerFlags struct {
	http.Header
}

func (flag *headerFlags) String() string {
	out := ""
	for k, v := range flag.Header {
		out += fmt.Sprintf("%s: %s\n", k, v)
	}
	return out
}

func (flag *headerFlags) Set(arg string) error {
	index := strings.Index(arg, ":")
	if index < 0 {
		return fmt.Errorf(` (%s). "`, arg)
	}
	if flag.Header == nil {
		flag.Header = http.Header{}
	}
	key := arg[0:index]
	value := arg[index+1:]
	flag.Header.Set(key, strings.TrimSpace(value))
	return nil
}

var clientHelp = `` + commonHelp

func cl(args []string) {
	flags := flag.NewFlagSet("cl", flag.ContinueOnError)
	config := chclient.Config{Headers: http.Header{}}
	flags.StringVar(&config.Fingerprint, "fp", "", "")
	flags.StringVar(&config.Auth, "auth", "", "")
	flags.DurationVar(&config.KeepAlive, "keepalive", 25*time.Second, "")
	flags.IntVar(&config.MaxRetryCount, "max-retry-count", -1, "")
	flags.DurationVar(&config.MaxRetryInterval, "max-retry-interval", 0, "")
	flags.StringVar(&config.Proxy, "proxy", "", "")
	flags.StringVar(&config.TLS.CA, "tls-ca", "", "")
	flags.BoolVar(&config.TLS.SkipVerify, "tls-skip-verify", false, "")
	flags.StringVar(&config.TLS.Cert, "tls-cert", "", "")
	flags.StringVar(&config.TLS.Key, "tls-key", "", "")
	flags.Var(&headerFlags{config.Headers}, "header", "")
	hostname := flags.String("hostname", "", "")
	sni := flags.String("sni", "", "")
	pid := flags.Bool("pid", false, "")
	verbose := flags.Bool("v", false, "")
	flags.Usage = func() {
		fmt.Print(clientHelp)
		os.Exit(0)
	}
	flags.Parse(args)
	//pull out options, put back remaining args
	args = flags.Args()
	if len(args) < 2 {
		log.Fatalf("")
	}
	config.Server = args[0]
	config.Remotes = args[1:]
	//default auth
	if config.Auth == "" {
		config.Auth = os.Getenv("AUTH")
	}
	//move hostname onto headers
	if *hostname != "" {
		config.Headers.Set("Host", *hostname)
		config.TLS.ServerName = *hostname
	}

	if *sni != "" {
		config.TLS.ServerName = *sni
	}

	//ready
	c, err := chclient.NewClient(&config)
	if err != nil {
		log.Fatal(err)
	}
	c.Debug = *verbose
	if *pid {
		generatePidFile()
	}
	go cos.GoStats()
	ctx := cos.InterruptContext()
	if err := c.Start(ctx); err != nil {
		log.Fatal(err)
	}
	if err := c.Wait(); err != nil {
		log.Fatal(err)
	}
}
