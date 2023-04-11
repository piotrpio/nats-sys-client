package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/nats-io/nats.go"
	nsys "github.com/piotrpio/nats-sys-client/pkg/sys"
)

type StreamDetail struct {
	StreamName string
	Account    string
	RaftGroup  string
	State      nats.StreamState
	Cluster    *nats.ClusterInfo
}

func main() {
	log.SetFlags(0)
	var urls, sname string
	flag.StringVar(&urls, "s", nats.DefaultURL, "The NATS server URLs (separated by comma)")
	flag.StringVar(&sname, "stream", "", "Select a single stream")
	flag.Parse()

	nc, err := nats.Connect(urls)
	if err != nil {
		log.Fatal(err)
	}
	sys := nsys.NewSysClient(nc)

	servers, err := sys.JszPing(nsys.JszEventOptions{
		JszOptions: nsys.JszOptions{
			Streams:    true,
			RaftGroups: true,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	header := fmt.Sprintf("Servers: %d", len(servers))
	fmt.Println(header)

	streams := make(map[string]map[string]*StreamDetail)
	// Collect all info from servers.
	for _, resp := range servers {
		server := resp.Server
		jsz := resp.JSInfo
		for _, acc := range jsz.AccountDetails {
			for _, stream := range acc.Streams {
				var ok bool
				var m map[string]*StreamDetail
				if m, ok = streams[stream.Name]; !ok {
					m = make(map[string]*StreamDetail)
					streams[stream.Name] = m
				}
				m[server.Name] = &StreamDetail{
					StreamName: stream.Name,
					Account:    acc.Name,
					RaftGroup:  stream.RaftGroup,
					State:      stream.State,
					Cluster:    stream.Cluster,
				}
			}
		}
	}
	keys := make([]string, 0)
	for k := range streams {
		for kk := range streams[k] {
			keys = append(keys, fmt.Sprintf("%s/%s", k, kk))
		}
	}
	sort.Strings(keys)

	line := strings.Repeat("-", 220)
	fmt.Printf("Streams: %d\n", len(keys))
	fmt.Println()

	fields := []any{"STREAM REPLICA", "RAFT", "ACCOUNT", "NODE", "MESSAGES", "BYTES", "STATUS"}
	fmt.Printf("%-20s %-15s %-10s %-15s %-15s %-15s %-30s\n", fields...)

	var prev string
	for i, k := range keys {
		v := strings.Split(k, "/")
		streamName, serverName := v[0], v[1]
		if sname != "" && streamName != sname {
			continue
		}
		if i > 0 && prev != streamName {
			fmt.Println(line)
		}

		stream := streams[streamName]
		replica := stream[serverName]
		status := "IN SYNC"

		// Make comparisons against other peers.
		for _, peer := range stream {
			if peer.State.Msgs != replica.State.Msgs && peer.State.Bytes != replica.State.Bytes {
				status = "UNSYNCED"
			}
		}

		sf := make([]any, 0)
		sf = append(sf, replica.StreamName)
		sf = append(sf, replica.RaftGroup)
		sf = append(sf, replica.Account)

		// Mark it in case it is a leader.
		var suffix string
		if serverName == replica.Cluster.Leader {
			suffix = "*"
		}
		s := fmt.Sprintf("%s%s", serverName, suffix)
		sf = append(sf, s)
		sf = append(sf, replica.State.Msgs)
		sf = append(sf, replica.State.Bytes)
		sf = append(sf, status)

		sf = append(sf, replica.Cluster.Leader)
		var replicasInfo string
		for _, r := range replica.Cluster.Replicas {
			info := fmt.Sprintf("%s(current=%-5v,offline=%v)", r.Name, r.Current, r.Offline)
			replicasInfo = fmt.Sprintf("%-40s %s", info, replicasInfo)
		}
		sf = append(sf, replicasInfo)
		fmt.Printf("%-20s %-15s %-10s %-15s %-15d %-15d| %-10s | leader: %s | peers: %s\n", sf...)

		prev = streamName
	}
}
