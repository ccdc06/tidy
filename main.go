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
)

func main() {
	code := truemain()
	fmt.Println("Execution complete. Press Enter to exit.")
	ScanLine()
	os.Exit(code)
}

var scanner = bufio.NewScanner(os.Stdin)

func truemain() int {
	fmt.Println("(Press Ctrl+C to exit at any time)")

	answer := ReadYesNo("Download the lists of files and collections from https://github.com/ccdc06/metadata/tree/master?")
	if !answer {
		fmt.Println("Download cancelled")
		return 0
	}

	expected, err := DownloadFileList()
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}

	var filesCount int
	var collections []string
	for key, files := range expected {
		filesCount += len(files)
		collections = append(collections, key)
	}

	fmt.Println("Download complete")
	fmt.Printf("List: %d files in %d collections\n", filesCount, len(collections))

	var rootDir string
	var foundCollections []string
	for {
		rootDir = ReadDirectory(fmt.Sprintf("Enter the path to the directory where the downloaded collections (like '%s') are located:", collections[0]))

		for _, collection := range collections {
			check := filepath.Join(rootDir, collection)

			if DirExists(check) {
				foundCollections = append(foundCollections, collection)
			}
		}

		if len(foundCollections) != 0 {
			break
		}
		fmt.Printf("No collection directories were found in '%s'\n", rootDir)
	}

	fmt.Printf("Collection directories found locally: %d\n", len(foundCollections))

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
		}
	}

	if len(incompleteCollections) > 0 {
		Hr()
		fmt.Println("There are missing files in the following collections:")

		for collection, count := range incompleteCollections {
			if count == 1 {
				fmt.Printf("%s: 1 missing file\n", collection)
			} else {
				fmt.Printf("%s: %d missing files\n", collection, count)
			}
		}

		answer := ReadYesNo("Would you like to see the list of missing files?")
		if answer {
			Hr()
			for _, file := range missingFiles {
				fmt.Printf("MISSING: %s\n", file)
			}
		}
	}

	if len(extraFiles) > 0 {
		Hr()
		done := false
		for {
			if done {
				break
			}
			options := map[string]string{"s": "Show me the list of files then ask again", "d": "Permanently delete", "m": "Move to another folder", "n": "Nothing"}

			answer := ReadOptions(fmt.Sprintf("What would you like to do with the extra %d cbz files found?", len(extraFiles)), options)

			switch answer {
			case "d":
				if ReadYesNo("This action can't be undone. Are you sure?") {
					for _, file := range extraFiles {
						err := os.Remove(file)

						if err != nil {
							fmt.Printf("ERROR: %s (file: %s)\n", err, file)
						} else {
							fmt.Printf("DELETED: %s\n", file)
						}
					}
					done = true
				}

			case "m":
				dest := ReadDirectory("Enter the folder where you want to move the files:")
				for cn, from := range extraFiles {
					var err error

					to := filepath.Join(dest, cn)
					toDir := filepath.Dir(to)
					if !DirExists(toDir) {
						err = os.MkdirAll(toDir, 0777)
					}

					if err != nil {
						fmt.Printf("ERROR: %s (file: %s)\n", err, from)
						continue
					}

					err = os.Rename(from, to)

					if err != nil {
						fmt.Printf("ERROR: %s (file: %s)\n", err, from)
					} else {
						fmt.Printf("MOVED: %s\n", from)
					}
				}
				done = true

			case "s":
				for _, file := range extraFiles {
					fmt.Printf("EXTRA: %s\n", file)
				}

			case "n":
				done = true
			}
		}
	}

	if len(extraFiles) == 0 && len(incompleteCollections) == 0 {
		fmt.Printf("No missing or extra files were found. Your collections are up to date!")
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
		fmt.Printf("%s [%s]\n", msg, validAnswersText)
		for option, desc := range options {
			fmt.Printf("%s: %s\n", option, desc)
		}
		answer = strings.ToLower(ScanLine())

		if len(answer) == 1 {
			_, ok := options[answer]
			if ok {
				return answer
			}
		}

		fmt.Printf("Valid answers: [%s]\n", validAnswersText)
	}
}

func ReadYesNo(msg string) bool {
	for {
		fmt.Printf("%s [y/n]\n", msg)
		answer := strings.ToLower(ScanLine())

		switch answer {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}

		fmt.Println("Valid answers: y, n, yes, no")
	}
}

func ReadDirectory(msg string) string {
	for {
		fmt.Println(msg)
		answer := ScanLine()

		stat, err := os.Stat(answer)

		if os.IsNotExist(err) {
			fmt.Println("Directory does no exists")
			continue
		}

		if err != nil {
			fmt.Println(err)
			continue
		}

		if !stat.IsDir() {
			fmt.Println("This is not a directory")
			continue
		}

		return answer
	}
}

func DownloadFileList() (map[string][]string, error) {
	var ret map[string][]string = make(map[string][]string)

	url := `https://raw.githubusercontent.com/ccdc06/metadata/master/indexes/list.csv`

	fmt.Println("Downloading list of files")
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
	fmt.Println("-----------------------------------")
}

func ScanLine() string {
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}
