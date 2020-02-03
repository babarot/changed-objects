package main

// Stat represents the stats for a file in a commit.
type Stat struct {
	Kind string
	Path string
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
