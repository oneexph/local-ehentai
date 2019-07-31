package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/firefoxchan/local-ehentai/pkg/ehloader"
	"github.com/firefoxchan/local-ehentai/pkg/websvr"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	var (
		jsonPath   string
		thumbsPath string
		format     string
		host       string
		pprofHost  string
		mode       string
	)
	flag.StringVar(&jsonPath, "j", "", "path to eh api json file")
	flag.StringVar(&thumbsPath, "t", "", "path to thumbs dir")
	flag.StringVar(&format, "f", "dense", "output format. dense, json")
	flag.StringVar(&host, "h", "127.0.0.1:8080", "http listen addr")
	flag.StringVar(&pprofHost, "p", "127.0.0.1:8081", "pprof http listen addr")
	flag.StringVar(&mode, "m", "http", "start mode. cmd, http")
	flag.Parse()
	if jsonPath == "" {
		if _, e := os.Stat("gdata.json"); e == nil {
			jsonPath = "gdata.json"
		}
	}
	if thumbsPath == "" {
		if fi, e := os.Stat("thumbs"); e == nil {
			if fi.IsDir() {
				thumbsPath = "thumbs"
			}
		}
	}
	if jsonPath == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	if e := ehloader.IndexJson(jsonPath); e != nil {
		panic(e)
	}
	switch mode {
	case "http":
		http(host, pprofHost, thumbsPath)
	case "cmd":
		cmd(format)
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func http(host string, pprofHost string, thumbs string) {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	ctx, ctxCancel := context.WithCancel(context.Background())
	go func() {
		<-signals
		ctxCancel()
	}()
	if e := websvr.HTTPServer(ctx, websvr.Config{
		Host:      host,
		PProfHost: pprofHost,
		Thumbs:    thumbs,
	}); e != nil {
		fmt.Printf("HTTPServer Error: %s\n", e)
	}
}

func cmd(format string) {
	// search
	scanner := bufio.NewScanner(os.Stdin)
	printHint := func() {
		fmt.Printf("Search format: offset, limit, tag1:value1, tag2:value2, ...\n")
		fmt.Printf("Example: 0, 10, category:doujinshi, parody:the idolmaster\n")
		fmt.Printf("Quit format: quit\n")
		fmt.Printf("Please Input: ")
	}
	printHint()
	for scanner.Scan() {
		in := scanner.Text()
		switch in {
		case "exit", "quit":
			fmt.Printf("Bye\n")
			return
		}
		params := strings.SplitN(in, ",", 3)
		if len(params) != 3 {
			printHint()
			continue
		}
		offset, e := strconv.ParseInt(strings.TrimSpace(params[0]), 10, 64)
		if e != nil {
			fmt.Println("Unable to parse offset:", e)
			printHint()
			continue
		}
		limit, e := strconv.ParseInt(strings.TrimSpace(params[1]), 10, 64)
		if e != nil {
			fmt.Println("Unable to parse limit:", e)
			printHint()
			continue
		}
		search := strings.TrimSpace(params[2])
		kvs := strings.Split(search, ",")
		searchTags := map[ehloader.TagK]map[ehloader.TagV]struct{}{}
		for _, kv := range kvs {
			key, value := ehloader.BuildKV(kv, ehloader.TagKAll)
			if _, ok := searchTags[key]; !ok {
				searchTags[key] = map[ehloader.TagV]struct{}{}
			}
			searchTags[key][value] = struct{}{}
		}
		ret := ehloader.Search(searchTags, int(offset), int(limit))
		fmt.Printf("Search: %d, %d, %v -> %v\n", offset, limit, search, searchTags)
		if format == "dense" {
			for _, g := range ret {
				b, e := json.Marshal(g)
				if e != nil {
					fmt.Printf("  %+v\n", e)
				} else {
					fmt.Printf("  %s\n", string(b))
				}
			}
		} else {
			b, e := json.MarshalIndent(ret, "  ", "  ")
			if e != nil {
				fmt.Printf("  %+v\n", e)
			} else {
				fmt.Printf("  %s\n", string(b))
			}
		}
		fmt.Printf("\nPlease Input: ")
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}
}
