package main

import (
	"encoding/json"
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
	Kind     Kind   `json:"kind"`
	Path     string `json:"path"`
	File     string `json:"file"`
	Dir      string `json:"dir"`
	DirExist bool   `json:"dir-exist"`
	Output   string `json:"-"`
}

type Stats []Stat

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
	m := make(map[string]bool)
	stats := make([]Stat, 0)
	for _, stat := range *ss {
		if !m[stat.Path] {
			m[stat.Path] = true
			stats = append(stats, stat)
		}
	}
	return stats
}
