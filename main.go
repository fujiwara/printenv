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
	accesslog "github.com/mash/go-accesslog"
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

type Logger struct {
	enc *json.Encoder
}

func (l Logger) Log(record accesslog.LogRecord) {
	l.enc.Encode(record)
}

var latency = &Latency{}

func main() {
	var port int
	logger := Logger{enc: json.NewEncoder(os.Stdout)}

	flag.IntVar(&port, "port", 8080, "port number")
	flag.DurationVar(&latency.duration, "latency", 0, "average latency")
	flag.BoolVar(&latency.randomize, "randomize", false, "randomize latency")
	flag.VisitAll(func(f *flag.Flag) {
		if s := os.Getenv(strings.ToUpper(f.Name)); s != "" {
			f.Value.Set(s)
		}
	})
	flag.Parse()
	log.Println("port:", port)
	log.Printf("latency: avg:%s randomize:%v", latency.duration, latency.randomize)

	var mux = http.NewServeMux()
	mux.HandleFunc("/", handlePrintenv)
	ridge.Run(
		fmt.Sprintf(":%d", port),
		"/",
		accesslog.NewLoggingHandler(mux, logger),
	)
}

func handlePrintenv(w http.ResponseWriter, r *http.Request) {
	latency.Sleep()
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
