// +build !go1.13

package main

type nullTime struct {
	Time  time.Time
	Valid bool
}
