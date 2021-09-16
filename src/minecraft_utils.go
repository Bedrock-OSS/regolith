package src

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
)

type World struct {
	Id   string `json:"id"`   // The name of the world directory
	Name string `json:"name"` // The name of the world in levelname.txt
	Path string `json:"path"`
}

func ListWorlds(mojangDir string) ([]*World, error) {
	var worlds = make(map[string]World)
	var existingWorldNames = make(map[string]struct{}) // A set with duplicated world names
	var exists = struct{}{}

	worldsPath := path.Join(mojangDir, "minecraftWorlds")
	files, err := ioutil.ReadDir(worldsPath)
	if err != nil {
		return nil, wrapError("Failed to list files inside worlds dir", err)
	}
	for _, f := range files {
		if f.IsDir() {
			worldPath := path.Join(worldsPath, f.Name())
			worldname, err := ioutil.ReadFile(path.Join(worldPath, "levelname.txt"))
			if err != nil {
				Logger.Warnf("Unable to read levelname.txt from %q", worldPath)
				continue
			}
			_, ok := existingWorldNames[string(worldname)]
			existingWorldNames[string(worldname)] = exists
			if ok { // The world with this name already exists
				delete(worlds, string(worldname))
				Logger.Warnf("Duplicated world name %q", worldname)
				continue
			}
			worlds[string(worldname)] = World{
				Name: string(worldname),
				Id:   f.Name(),
				Path: worldPath,
			}
		}
	}
	// Convert result to list
	var result []*World
	for _, val := range worlds {
		result = append(result, &val)
	}
	return result, nil
}

func FindMojangDir() (string, error) {
	if runtime.GOOS != "windows" {
		return "", errors.New(fmt.Sprintf("unsupported OS '%s'", runtime.GOOS))
	}
	result := path.Join(os.Getenv("LOCALAPPDATA"), "Packages", "Microsoft.MinecraftUWP_8wekyb3d8bbwe", "LocalState", "games", "com.mojang")
	if _, err := os.Stat(result); os.IsNotExist(err) {
		return "", wrapError(fmt.Sprintf("Failed to find file %s", result), err)
	}
	return result, nil
}
