//go:build !windows

package main

func winCredGet(name string) string      { return "" }
func winCredSet(name, key string) error  { return nil }
func winCredDelete(name string)          {}
