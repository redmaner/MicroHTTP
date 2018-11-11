package main

import (
	"fmt"
	"io"
	"sync"
)

type metricsData struct {
	sync.Mutex
	enabled       bool
	totalRequests int
	paths         map[int]map[string]int
}

func (md *metricsData) concat(e int, p string) {
	if md.enabled {
		md.Lock()
		if _, ok := md.paths[e]; ok {
			if _, ok := md.paths[e][p]; ok {
				md.paths[e][p]++
			} else {
				md.paths[e][p] = 1
			}
		} else {
			m := make(map[string]int)
			md.paths[e] = m
			md.paths[e][p] = 1
		}
		md.totalRequests++
		md.Unlock()
	}
}

func (md *metricsData) display(o io.Writer) {
	md.Lock()
	io.WriteString(o, fmt.Sprintf("<h1>MicroHTTP metrics</h1><br><b>Total requests:</b> %d<br>", md.totalRequests))
	for k, v := range md.paths {
		io.WriteString(o, fmt.Sprintf("<br><b>%d</b><ul>", k))
		for p, a := range v {
			io.WriteString(o, fmt.Sprintf("<li>Amount: %d - Path: %s</li>", a, p))
		}
		io.WriteString(o, fmt.Sprintf("</ul>"))
	}
	md.Unlock()
}