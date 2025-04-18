package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fujiwara/ridge"
	"github.com/mashiike/accesslogger"
)

type Latency struct {
	duration  time.Duration
	randomize bool
}

func (l *Latency) Sleep() {
	if l.duration == 0 {
		return
	}
	var s time.Duration
	if l.randomize {
		s = time.Duration(rand.NormFloat64() * float64(l.duration))
	} else {
		s = l.duration
	}
	time.Sleep(s)
}

var commonLatency = &Latency{}

func main() {
	var port int

	flag.IntVar(&port, "port", 8080, "port number")
	flag.DurationVar(&commonLatency.duration, "latency", 0, "average latency")
	flag.BoolVar(&commonLatency.randomize, "randomize", false, "randomize latency")
	flag.VisitAll(func(f *flag.Flag) {
		if s := os.Getenv(strings.ToUpper(f.Name)); s != "" {
			f.Value.Set(s)
		}
	})
	flag.Parse()
	log.Println("port:", port)
	log.Printf("latency: avg:%s randomize:%v", commonLatency.duration, commonLatency.randomize)

	var mux = http.NewServeMux()
	mux.HandleFunc("/", handlePrintenv)
	mux.HandleFunc("/headers", handleHeaders)
	ridge.Run(
		fmt.Sprintf(":%d", port),
		"/",
		accesslogger.Wrap(mux, accesslogger.JSONLogger(os.Stdout)),
	)
}

func newLatencyFromRequest(r *http.Request) (*Latency, error) {
	s := r.URL.Query().Get("latency")
	if s == "" {
		return commonLatency, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return nil, fmt.Errorf("invalid latency: %s", err)
	}
	return &Latency{
		duration:  d,
		randomize: commonLatency.randomize,
	}, nil
}

func handlePrintenv(w http.ResponseWriter, r *http.Request) {
	if l, err := newLatencyFromRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	} else {
		l.Sleep()
	}
	ac := r.Header.Get("Accept")
	if strings.Contains(ac, "application/json") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		envs := make(map[string]string, len(os.Environ()))
		for _, v := range os.Environ() {
			kv := strings.SplitN(v, "=", 2)
			envs[kv[0]] = kv[1]
		}
		json.NewEncoder(w).Encode(envs)
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		envs := os.Environ()
		sort.SliceStable(envs, func(i, j int) bool {
			return envs[i] < envs[j]
		})
		for _, v := range envs {
			fmt.Fprintln(w, v)
		}
	}
}

func handleHeaders(w http.ResponseWriter, r *http.Request) {
	if l, err := newLatencyFromRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	} else {
		l.Sleep()
	}
	headers := make(map[string]string, len(r.Header))
	keys := make([]string, 0, len(r.Header))
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ",")
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ac := r.Header.Get("Accept")
	if strings.Contains(ac, "application/json") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(headers)
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		for _, k := range keys {
			fmt.Fprintf(w, "%s: %s\n", k, headers[k])
		}
	}
}
