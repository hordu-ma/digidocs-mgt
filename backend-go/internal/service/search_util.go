package service

import (
	"sort"
	"strconv"
)

func itoa(n int) string { return strconv.Itoa(n) }

func sortEntries[T any](items []T, less func(a, b T) bool) {
	sort.SliceStable(items, func(i, j int) bool { return less(items[i], items[j]) })
}
