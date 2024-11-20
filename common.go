package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fatih/color"
)

// List all items in a that aren't in b
func Diff[T comparable](a []T, b []T) []T {
	var ret []T

	for _, item := range a {
		if !slices.Contains(b, item) {
			ret = append(ret, item)
		}
	}

	return ret
}

func ReadOptions(msg string, options map[string]string) string {
	var answer string
	var validAnswers []string
	for option := range options {
		validAnswers = append(validAnswers, option)
	}
	validAnswersText := strings.Join(validAnswers, "/")

	for {
		color.White("%s [%s]", msg, validAnswersText)
		for option, desc := range options {
			color.White("%s: %s", option, desc)
		}
		answer = strings.ToLower(ScanLine())
		if len(answer) == 0 {
			continue
		}

		if len(answer) == 1 {
			_, ok := options[answer]
			if ok {
				return answer
			}
		}

		color.White("Valid answers: [%s]\n", validAnswersText)
	}
}

func ReadYesNo(msg string) bool {
	for {
		color.White("%s [y/n]\n", msg)
		answer := strings.ToLower(ScanLine())

		switch answer {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}

		color.White("Valid answers: y, n, yes, no")
	}
}

func ReadDirectory(msg string, tryCreate bool) string {
	firstLoop := true

	for {
		color.White(msg)
		if tryCreate && firstLoop {
			color.White("(If the directory does not exist, it will be created)")
		}
		firstLoop = false

		answer := ScanLine()
		if len(answer) == 0 {
			continue
		}

		stat, err := os.Stat(answer)

		if os.IsNotExist(err) {
			if tryCreate {
				err = os.MkdirAll(answer, 0777)
				if err != nil {
					color.Yellow(err.Error())
					continue
				}
				color.Green("directory %s created", answer)
				return answer
			}

			color.Yellow("Directory does no exists")
			continue
		}

		if err != nil {
			color.Yellow(err.Error())
			continue
		}

		if !stat.IsDir() {
			color.Yellow("This is not a directory")
			continue
		}

		return answer
	}
}

func DirExists(dir string) bool {
	stat, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return false
	}

	if !stat.IsDir() {
		return false
	}
	return true
}

func Hr() {
	color.White("-----------------------------------")
}

var scanner = bufio.NewScanner(os.Stdin)

func ScanLine() string {
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func ReadFileList(r io.Reader) (map[string][]string, error) {
	var ret map[string][]string = make(map[string][]string)

	reader := csv.NewReader(r)
	rows, err := reader.ReadAll()
	if err != nil {
		return ret, err
	}

	for _, row := range rows {
		// skip empty rows
		if len(row) != 2 {
			continue
		}

		// skip headers
		if !strings.Contains(row[1], "/") {
			continue
		}

		// skip rows where the second column isn't a cbz file
		if !strings.HasSuffix(row[1], ".cbz") {
			continue
		}

		split := strings.SplitN(row[1], "/", 2)

		// skip rows where the second column isn't in the format <collection>/<gallery>
		if len(split) != 2 {
			continue
		}

		ret[split[0]] = append(ret[split[0]], split[1])
	}

	if len(ret) == 0 {
		return ret, fmt.Errorf("empty list of files")
	}

	return ret, err
}

func listExpectedCollections(expected map[string][]string) []string {
	var filesCount int
	var collections []string
	for key, files := range expected {
		filesCount += len(files)
		collections = append(collections, key)
	}
	color.Green("List: %d files in %d collections", filesCount, len(collections))
	return collections
}

func scanLocalFilesByPattern(foundCollections []string, rootDir string, pattern string) (map[string][]string, int) {
	var found map[string][]string = make(map[string][]string)
	total := 0
	for _, collection := range foundCollections {
		pattern := filepath.Join(rootDir, collection, pattern)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			total++
			f := filepath.Base(match)

			found[collection] = append(found[collection], f)
		}
	}
	return found, total
}

func scanCbzFiles(foundCollections []string, rootDir string) (map[string][]string, int) {
	return scanLocalFilesByPattern(foundCollections, rootDir, "*.cbz")
}

func scanYamlFiles(foundCollections []string, rootDir string) (map[string][]string, int) {
	return scanLocalFilesByPattern(foundCollections, rootDir, "*.yaml")
}

func scanLocalCollections(collections []string) (string, []string) {
	var rootDir string
	var foundCollections []string

	for {
		rootDir = ReadDirectory(fmt.Sprintf("Enter the path to the directory where the downloaded collections (like '%s') are located:", collections[0]), false)

		for _, collection := range collections {
			check := filepath.Join(rootDir, collection)

			if DirExists(check) {
				foundCollections = append(foundCollections, collection)
			}
		}

		if len(foundCollections) != 0 {
			break
		}
		color.Yellow("No collection directories were found in '%s'", rootDir)
	}

	color.Green("Collection directories found locally: %d of %d", len(foundCollections), len(collections))
	return rootDir, foundCollections
}
