package main

import (
	"math"
	"time"
)

type flashcard struct {
	Simplified  string
	Traditional string
	Pinyin      string
	English     []string
	LastSeen    time.Time
	Correct     int
	Wrong       int
}

type MasterCard struct {
	Id          int
	Simplified  string
	Traditional string
	Pinyin      string
	English     []string
}

type UserCard struct {
	Master   MasterCard
	LastSeen time.Time
	Wrong    int
	Correct  int
}

func (f UserCard) Points() float64 {
	weightTime := math.Sqrt(time.Since(f.LastSeen).Hours())
	pctWrong := float64(f.Wrong+1) / float64(f.Wrong+f.Correct)
	if f.Correct < 1 {
		if f.Wrong > 0 {
			return pctWrong * weightTime * 100
		}
		return Benchmark
	}
	weightCorrect := float64(f.Correct) / 100
	return (pctWrong * weightTime) / weightCorrect
}

type ByTime struct {
	u []UserCard
}

func (f ByTime) Len() int           { return len(f.u) }
func (f ByTime) Swap(i, j int)      { f.u[i], f.u[j] = f.u[j], f.u[i] }
func (f ByTime) Less(i, j int) bool { return f.u[i].LastSeen.Before(f.u[j].LastSeen) }

type ByPoints []UserCard

func (f ByPoints) Len() int           { return len(f) }
func (f ByPoints) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f ByPoints) Less(i, j int) bool { return f[i].Points() > f[j].Points() }

type ById struct {
	m []MasterCard
}

func (f ById) Len() int           { return len(f.m) }
func (f ById) Swap(i, j int)      { f.m[i], f.m[j] = f.m[j], f.m[i] }
func (f ById) Less(i, j int) bool { return f.m[i].Id > f.m[j].Id }
