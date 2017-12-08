package main

import (
	"bytes"
	"fmt"
	"github.com/intrip/simple_balancer/common"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	bind, balance        string
	port, maxConnections int
	readTimeout          = 10
	writeTimeout         = 10
	backends             []common.Backend
)

func init() {
	loadConfig("config")
}

// loads config from ./config.yaml
func loadConfig(config string) {
	viper.SetConfigType("yaml")
	viper.SetConfigName(config)
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Error in config file: %s \n", err))
	}

	server := viper.GetStringMapString("server")
	// read port
	if v, ok := server["port"]; ok {
		port, err = strconv.Atoi(v)
		if err != nil {
			panic(fmt.Errorf("Server port is not valid: %s \n", err))
		}
	} else {
		panic(fmt.Errorf("Server port is required"))
	}
	// listen
	if v, ok := server["bind"]; ok {
		bind = v
	} else {
		panic(fmt.Errorf("Server bind is required"))
	}
	// maxConnections
	if v, ok := server["maxconnections"]; ok {
		maxConnections, err = strconv.Atoi(v)
		if err != nil {
			panic(fmt.Errorf("Server maxConnections is not valid: %s \n", err))
		}
	} else {
		panic(fmt.Errorf("Server maxConnections is required"))
	}

	// timeout
	if v, ok := server["readtimeout"]; ok {
		readTimeout, err = strconv.Atoi(v)
		if err != nil {
			panic(fmt.Errorf("server readtimeout is not valid: %s \n", err))
		}
	}
	if v, ok := server["writetimeout"]; ok {
		writeTimeout, err = strconv.Atoi(v)
		if err != nil {
			panic(fmt.Errorf("server writetimeout is not valid: %s \n", err))
		}
	}

	balance = viper.GetString("balancers")
	backends = parseBalance(balance)
}

func main() {
	s := &http.Server{
		Addr:           serverUrl(),
		Handler:        common.NewLimitHandler(maxConnections, &Proxy{}),
		ReadTimeout:    time.Duration(readTimeout) * time.Second,
		WriteTimeout:   time.Duration(writeTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	s.ListenAndServe()
}

func serverUrl() string {
	return fmt.Sprintf("%s:%d", bind, port)
}

type Proxy struct{}

func (h *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backendStruct := &common.Backends{0, backends}
	next := common.NextRoundRobin(backendStruct)
	doBalance(w, r, &next)
}

func doBalance(w http.ResponseWriter, r *http.Request, backend *common.Backend) {
	u, err := url.Parse(backend.Url + r.RequestURI)
	if err != nil {
		log.Panic("Error parsing backend Url: ", err)
	}

	client := &http.Client{}
	req := &http.Request{Method: r.Method, URL: u, Body: r.Body, Host: backend.Url, Header: make(map[string][]string)}
	// sets forwarded header
	forwarded := fmt.Sprintf("by=%s; for=%s; host=%s; proto=%s", serverUrl(), r.RemoteAddr, r.Host, r.Proto)
	req.Header.Set("Forwarded", forwarded)
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Panic("Error reading response: ", err)
	}

	buffer := bytes.NewBuffer(bodyBytes)
	for k, v := range res.Header {
		w.Header().Set(k, strings.Join(v, ";"))
	}
	w.WriteHeader(res.StatusCode)

	io.Copy(w, buffer)
}

func parseBalance(balancers string) (backends []common.Backend) {
	urls := strings.Split(balancers, ",")
	backends = make([]common.Backend, len(urls))

	for index, backend := range urls {
		backends[index] = common.Backend{Url: backend, ActiveConnections: 0}
	}

	return
}
