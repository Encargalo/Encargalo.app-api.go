package repo

import (
	"reflect"
	"strings"
)

// postgresMaxQueryParams is the protocol limit for a single query.
const postgresMaxQueryParams = 65535

func olapInsertColumns[T any]() int {
	var zero T
	t := reflect.TypeOf(zero)
	n := 0
	for f := range t.Fields() {
		if f.Anonymous {
			continue
		}
		tag := f.Tag.Get("bun")
		if tag == "-" || strings.Contains(tag, "rel:") {
			continue
		}
		n++
	}
	if n < 1 {
		return 1
	}
	return n
}

func insertChunkSize(columns int) int {
	size := postgresMaxQueryParams / columns
	if size < 1 {
		return 1
	}
	return size
}

func insertInChunks[T any](rows []T, fn func(chunk []T) error) error {
	if len(rows) == 0 {
		return nil
	}
	size := insertChunkSize(olapInsertColumns[T]())
	for i := 0; i < len(rows); i += size {
		end := min(i+size, len(rows))
		if err := fn(rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}
