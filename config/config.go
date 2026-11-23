package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func Read(r io.Reader) map[string]string {
	m := make(map[string]string)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if i := strings.IndexRune(line, '='); i >= 0 {
			m[line[:i]] = line[i+1:]
		}
	}
	return m
}

func ReadFile(source string) (map[string]string, error) {
	f, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return Read(f), nil
}

func Save(m map[string]string, w io.Writer) error {
	for _, key := range SortedKeys(m) {
		if _, err := fmt.Fprintf(w, "%s=%s\n", key, m[key]); err != nil {
			return err
		}
	}
	return nil
}

func SaveFile(m map[string]string, target string) error {
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return Save(m, f)
}

func Setenv(m map[string]string) error {
	for k, v := range m {
		if v != "" {
			if err := os.Setenv(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func Unsetenv(m map[string]string) error {
	for k := range m {
		if err := os.Unsetenv(k); err != nil {
			return err
		}
	}
	return nil
}

func SortedKeys(m map[string]string) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
