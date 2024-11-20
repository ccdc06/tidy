package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fatih/color"
)

var cacheBaseDir string
var cacheDir string
var cacheFile string

func updateYamlFiles() error {
	var err error

	cacheBaseDir, err = os.UserCacheDir()
	if err != nil {
		return err
	}
	cacheDir = filepath.Join(cacheBaseDir, "TidyTool")
	cacheFile = filepath.Join(cacheDir, "master.zip")

	cacheDirStat, err := os.Stat(cacheDir)
	if os.IsNotExist(err) {
		fmt.Printf("Creating cache directory %s\n", cacheDir)
		err = os.MkdirAll(cacheDir, 0755)
		if err != nil {
			return err
		}
	} else if !cacheDirStat.IsDir() {
		fmt.Printf("Removing %s\n", cacheDir)
		err = os.Remove(cacheDir)
		if err != nil {
			return err
		}

		fmt.Printf("Creating cache directory %s\n", cacheDir)
		err = os.MkdirAll(cacheDir, 0755)
		if err != nil {
			return err
		}
	}

	cacheFileStat, err := os.Stat(cacheFile)
	if os.IsNotExist(err) {
		answer := ReadYesNo(fmt.Sprintf("Cache file %s not found. Download it now?", cacheFile))
		if !answer {
			err = fmt.Errorf("operation cancelled")
			return err
		}
		err = DownloadRelease()
	} else {
		answer := ReadYesNo(fmt.Sprintf("Cache file %s already exists. It was downloaded on %s. Download it again?", cacheFile, cacheFileStat.ModTime().Format("2006-01-02 15:04 (-0700)")))
		if answer {
			err = DownloadRelease()
		} else {
			err = nil
		}
	}

	if err != nil {
		return err
	}

	zipReeader, err := zip.OpenReader(cacheFile)
	if err != nil {
		return err
	}

	listFile, err := zipReeader.Open("metadata-master/indexes/list.csv")
	if err != nil {
		return err
	}
	defer listFile.Close()

	expected, err := ReadFileList(listFile)
	if err != nil {
		return err
	}

	collections := listExpectedCollections(expected)

	rootDir, foundCollections := scanLocalCollections(collections)

	fmt.Println("Listing cbz files")
	cbzFound, _ := scanCbzFiles(foundCollections, rootDir)

	fmt.Println("Listing yaml files")
	yamlFound, _ := scanYamlFiles(foundCollections, rootDir)

	updateCreate := ReadYesNo("Are you sure you want to create or update paired yaml files?")

	if !updateCreate {
		return nil
	}

	for collection := range cbzFound {
		for _, cbzName := range cbzFound[collection] {
			yamlName := strings.TrimSuffix(cbzName, ".cbz") + ".yaml"

			yamlFn := filepath.Join(rootDir, collection, yamlName)

			zipYamlFile, err := zipReeader.Open(path.Join("metadata-master", collection, yamlName))
			if err != nil {
				return err
			}
			targetYamlFile, err := os.Create(yamlFn)
			if err != nil {
				return err
			}

			_, err = io.Copy(targetYamlFile, zipYamlFile)
			zipYamlFile.Close()
			targetYamlFile.Close()

			if err != nil {
				fmt.Println("Error: " + err.Error())
				continue
			}

			i := slices.Index(yamlFound[collection], yamlName)
			if i != -1 {
				yamlFound[collection] = slices.Delete(yamlFound[collection], i, i+1)
			}

			fmt.Printf("%s written\n", yamlFn)
		}
	}

	var unknownYamlFiles []string
	for collection := range yamlFound {
		for _, yamlName := range yamlFound[collection] {
			unknownYamlFiles = append(unknownYamlFiles, filepath.Join(rootDir, collection, yamlName))
		}
	}

	if len(unknownYamlFiles) == 0 {
		fmt.Println("No unknown yaml files were found in the collections directories")
		return nil
	}

	var deleteUnknown bool
	if len(unknownYamlFiles) == 1 {
		deleteUnknown = ReadYesNo("There is 1 unknown yaml file in a collection directory. Delete it?")
	} else {
		deleteUnknown = ReadYesNo(fmt.Sprintf("There are %d unknown yaml files in the collections directories. Delete them?", len(unknownYamlFiles)))
	}

	if deleteUnknown {
		for _, fileName := range unknownYamlFiles {
			err = os.Remove(fileName)
			if err != nil {
				fmt.Println("Error: " + err.Error())
				continue
			}
		}
	}

	return nil
}

func DownloadRelease() error {
	var err error
	url := `https://github.com/ccdc06/metadata/archive/refs/heads/master.zip`

	color.Green("Downloading release from %s", url)

	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	color.Green("Done!")

	out, err := os.Create(cacheFile)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, response.Body)
	if err != nil {
		return err
	}

	return nil
}
