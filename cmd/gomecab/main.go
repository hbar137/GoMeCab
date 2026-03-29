package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gomecab"
)

func main() {
	dictDir := flag.String("d", "", "path to MeCab dictionary directory")
	flag.Parse()

	if *dictDir == "" {
		*dictDir = findDictDir()
		if *dictDir == "" {
			fmt.Fprintln(os.Stderr, "error: no dictionary directory specified and none found automatically")
			fmt.Fprintln(os.Stderr, "usage: gomecab -d /path/to/mecab/dict")
			os.Exit(1)
		}
	}

	t, err := gomecab.New(*dictDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			fmt.Println("EOS")
			continue
		}
		tokens := t.Tokenize(line)
		for _, tok := range tokens {
			fmt.Printf("%s\t%s\n", tok.Surface, tok.Feature)
		}
		fmt.Println("EOS")
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}
}

func findDictDir() string {
	// Try mecab-config
	out, err := exec.Command("mecab-config", "--dicdir").Output()
	if err == nil {
		base := strings.TrimSpace(string(out))
		// Check for UTF-8 subdirectories
		for _, sub := range []string{"mecab-ipadic-utf8", "ipadic-utf8", "ipadic", ""} {
			dir := filepath.Join(base, sub)
			if _, err := os.Stat(filepath.Join(dir, "sys.dic")); err == nil {
				return dir
			}
		}
	}

	// Try common paths
	paths := []string{
		"/var/lib/mecab/dic/mecab-ipadic-utf8",
		"/var/lib/mecab/dic/ipadic-utf8",
		"/usr/lib/x86_64-linux-gnu/mecab/dic/mecab-ipadic-utf8",
		"/usr/local/lib/mecab/dic/ipadic-utf8",
		"/usr/local/lib/mecab/dic/ipadic",
	}
	for _, p := range paths {
		if _, err := os.Stat(filepath.Join(p, "sys.dic")); err == nil {
			return p
		}
	}

	return ""
}
