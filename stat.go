package main

import (
	"encoding/json"
	"io"
)

type Kind int

const (
	Addition Kind = iota
	Deletion
	Modification
	Unknown
)

func (k Kind) String() string {
	switch k {
	case Addition:
		return "insert"
	case Deletion:
		return "delete"
	case Modification:
		return "modify"
	default:
		return "unknown"
	}
}

func (k Kind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// Stat represents the stats for a file in a commit.
type Stat struct {
	Kind Kind   `json:"kind"`
	Path string `json:"path"`
}

type Stats []Stat

type Result struct {
	Repo  string `json:"repo,omitempty"`
	Stats Stats  `json:"stats,omitempty"`
}

func (r Result) Print(w io.Writer) error {
	return json.NewEncoder(w).Encode(&r)
}

func (ss *Stats) Filter(f func(Stat) bool) Stats {
	stats := make([]Stat, 0)
	for _, stat := range *ss {
		if f(stat) {
			stats = append(stats, stat)
		}
	}
	return stats
}

func (ss *Stats) Map(f func(Stat) Stat) Stats {
	stats := make([]Stat, len(*ss))
	for i, stat := range *ss {
		stats[i] = f(stat)
	}
	return stats
}

func (ss *Stats) Unique() Stats {
	m := make(map[Stat]bool)
	stats := make([]Stat, 0)
	for _, stat := range *ss {
		if !m[stat] {
			m[stat] = true
			stats = append(stats, stat)
		}
	}
	return stats
}
