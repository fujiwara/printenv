package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/fujiwara/ridge"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 8080, "port number")
	flag.VisitAll(func(f *flag.Flag) {
		if s := os.Getenv(strings.ToUpper(f.Name)); s != "" {
			f.Value.Set(s)
		}
	})
	flag.Parse()

	var mux = http.NewServeMux()
	mux.HandleFunc("/", handlePrintenv)
	ridge.Run(fmt.Sprintf(":%d", port), "/", mux)
}

func handlePrintenv(w http.ResponseWriter, r *http.Request) {
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
