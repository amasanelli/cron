package cron

import (
	"testing"
	"time"
)

func BenchmarkNext(b *testing.B) {
	c := MustParse("59 23 1 * 1", time.UTC)
	for i := 0; i < b.N; i++ {
		c.Next(time.Now())
	}
}
