package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"os"

	"github.com/fatih/color"
)

func main() {
	code := truemain()
	if code == 0 {
		color.Green("Execution complete. Press Enter to exit.")
	} else {
		color.Yellow("Execution complete (error code %d). Press Enter to exit.", code)
	}
	ScanLine()
	os.Exit(code)
}

var scanner = bufio.NewScanner(os.Stdin)

func truemain() int {
	color.Green("(Press Ctrl+C to exit at any time)")

	answer := ReadYesNo("Download the lists of files and collections from https://github.com/ccdc06/metadata/tree/master?")
	if !answer {
		color.Yellow("Download cancelled")
		return 0
	}

	expected, err := DownloadFileList()
	if err != nil {
		color.Red(err.Error())
		return 1
	}

	var filesCount int
	var collections []string
	for key, files := range expected {
		filesCount += len(files)
		collections = append(collections, key)
	}

	color.Green("Download complete")
	color.Green("List: %d files in %d collections", filesCount, len(collections))

	Hr()

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

	color.Green("Collection directories found locally: %d", len(foundCollections))

	var found map[string][]string = make(map[string][]string)
	for _, collection := range foundCollections {
		pattern := filepath.Join(rootDir, collection, "*.cbz")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			f := filepath.Base(match)

			found[collection] = append(found[collection], f)
		}
	}

	var extraFiles map[string]string = make(map[string]string)
	var extraCollections map[string]int = make(map[string]int)
	var incompleteCollections map[string]int = make(map[string]int)
	var missingFiles []string

	for collection, foundFiles := range found {
		expectedFiles := expected[collection]

		extra := Diff(foundFiles, expectedFiles)
		missing := Diff(expectedFiles, foundFiles)

		if len(missing) > 0 {
			incompleteCollections[collection] = len(missing)

			for _, n := range missing {
				missingFiles = append(missingFiles, filepath.Join(rootDir, collection, n))
			}
		}

		for _, n := range extra {
			cn := collection + "/" + n
			extraFiles[cn] = filepath.Join(rootDir, collection, n)
			extraCollections[collection]++
		}
	}

	Hr()

	if len(incompleteCollections) > 0 {
		color.Yellow("There are missing files in the following collections:")

		for collection, count := range incompleteCollections {
			color.Yellow("%s: %d of %d galleries (%d missing)", collection, len(expected[collection])-count, len(expected[collection]), count)
		}
		Hr()

		answer := ReadYesNo("Would you like to see the list of missing files?")
		if answer {
			Hr()
			for _, file := range missingFiles {
				color.Yellow("MISSING: %s", file)
			}
		}
	} else {
		color.Green("No missing files found")
	}

	Hr()
	if len(extraFiles) > 0 {
		color.Yellow("There are extra galleries in the following collections:")
		for collection, count := range extraCollections {
			if count == 1 {
				color.Yellow("%s (1 extra gallery)", collection)
			} else {
				color.Yellow("%s (%d extra galleries)", collection, count)
			}
		}

		done := false
		for {
			if done {
				break
			}
			Hr()
			options := map[string]string{"s": "Show me the list of galleries then ask again", "d": "Permanently delete", "m": "Move to another directory", "n": "Nothing"}

			answer := ReadOptions(fmt.Sprintf("What would you like to do with the extra %d cbz file(s) found?", len(extraFiles)), options)

			switch answer {
			case "d":
				if ReadYesNo("This action can't be undone. Are you sure?") {
					Hr()
					for _, file := range extraFiles {
						err := os.Remove(file)

						if err != nil {
							color.Yellow("ERROR: %s (file: %s)", err, file)
						} else {
							color.Green("DELETED: %s", file)
						}
					}
					done = true
				}

			case "m":
				dest := ReadDirectory("Enter the directory where you want to move the files:", true)
				Hr()
				for cn, from := range extraFiles {
					var err error

					to := filepath.Join(dest, cn)
					toDir := filepath.Dir(to)
					if !DirExists(toDir) {
						err = os.MkdirAll(toDir, 0777)
					}

					if err != nil {
						color.Yellow("ERROR: %s (file: %s)", err, from)
						continue
					}

					err = os.Rename(from, to)

					if err != nil {
						color.Yellow("ERROR: %s (file: %s)", err, from)
					} else {
						color.Green("MOVED: %s => %s", from, to)
					}
				}
				done = true

			case "s":
				Hr()
				for _, file := range extraFiles {
					color.Yellow("EXTRA: %s", file)
				}

			case "n":
				Hr()
				done = true
			}
		}
		Hr()
	} else {
		color.Green("No extra files found")
		Hr()
	}

	if len(extraFiles) == 0 && len(incompleteCollections) == 0 {
		color.Green("Your collections are up to date!")
	}

	return 0
}

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

func DownloadFileList() (map[string][]string, error) {
	var ret map[string][]string = make(map[string][]string)

	url := `https://raw.githubusercontent.com/ccdc06/metadata/master/indexes/list.csv`

	color.Green("Downloading list of files")
	response, err := http.Get(url)
	if err != nil {
		return ret, err
	}

	reader := csv.NewReader(response.Body)
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

func ScanLine() string {
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}
