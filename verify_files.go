package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

func verifyFiles() error {
	answer := ReadYesNo("Download the lists of files and collections from https://github.com/ccdc06/metadata/tree/master?")
	if !answer {
		color.Yellow("Download cancelled")
		return nil
	}

	expected, err := DownloadFileList()
	if err != nil {
		color.Red(err.Error())
		return err
	}
	color.Green("Download complete")

	collections := listExpectedCollections(expected)

	Hr()

	rootDir, foundCollections := scanLocalCollections(collections)

	found, _ := scanCbzFiles(foundCollections, rootDir)

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

	return nil
}

func DownloadFileList() (map[string][]string, error) {
	var ret map[string][]string = make(map[string][]string)

	url := `https://raw.githubusercontent.com/ccdc06/metadata/master/indexes/list.csv`

	color.Green("Downloading list of files")
	response, err := http.Get(url)
	if err != nil {
		return ret, err
	}

	return ReadFileList(response.Body)
}
