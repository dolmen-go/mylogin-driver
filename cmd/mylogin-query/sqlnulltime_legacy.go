// +build !go1.13

package main

import "time"

type nullTime struct {
	Time  time.Time
	Valid bool
}
