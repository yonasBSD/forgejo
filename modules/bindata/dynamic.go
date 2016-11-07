// +build !bindata

package bindata

import (
	"fmt"
	"os"
	"path/filepath"
	"io/ioutil"
	"strings"
)

var rootDir string
var binData map[string] []byte
var dirData map[string] []string

func setBinData(key string, data []byte) {
	if binData == nil {
		binData = make(map[string] []byte)
	}
	binData[key] = data
}

func setDirData(key string, data []string) {
	if dirData == nil {
		dirData = make(map[string] []string)
	}
	dirData[key] = data
}

func getRootDir() (string, error) {
	if rootDir != "" {
		return rootDir, nil
	}

	dir := os.Getenv("GITEA_ROOT")
	if dir == "" {
		dir = "."
	}

	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("%v", err)
	}
	for {
		_, err = ioutil.ReadDir(filepath.Join(dir, "conf"))
		if err == nil {
			// TODO: check if the file is a directory
			break
		}
		// TODO: check that the error is a "file not found" error ?
		newdir := filepath.Join(dir, "..")
		if dir == newdir {
			return "", fmt.Errorf("Could not find directory containing 'conf', try setting GITEA_ROOT")
		}
		dir = newdir
	}

	fmt.Println("WARNING: this deveopment build of Gitea depends on a directory tree, we'll be using the one in ", dir)

	rootDir = dir
	return dir, nil
}

func resolveName(name string) (string, error) {

	name = strings.Replace(name, "\\", "/", -1) // needed ?

	dir, err := getRootDir()
	if err != nil {
		return "", fmt.Errorf("%v", err)
	}

	return filepath.Join(dir,name), nil
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {

	canonicalName, err := resolveName(name)
	if err != nil {
		return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
	}

	dat := binData[canonicalName]
	if dat != nil { return dat, nil }

	dat, err = ioutil.ReadFile(canonicalName)
	if err != nil {
		return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
	}

	setBinData(canonicalName, dat)
	return dat, nil
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	canonicalName, err := resolveName(name)
	if err != nil {
		return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
	}

	rv := dirData[canonicalName]
	if rv != nil { return rv, nil }

	files, err := ioutil.ReadDir(canonicalName)
	if err != nil {
		return nil, fmt.Errorf("Error reading directory %s: ", err)
	}

	rv = make([]string, 0, len(files))
	for _, f := range files {
		rv = append(rv, f.Name())
	}

	setDirData(canonicalName, rv)
	return rv, nil
}
