package src

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
)

type World struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

func ListWorlds(mojangDir string) []World {
	var worlds = make(map[string]World)
	var existingWorldNames = make(map[string]struct{}) // A set with duplicated world names
	var exists = struct{}{}

	worldsPath := path.Join(mojangDir, "minecraftWorlds")
	files, err := ioutil.ReadDir(worldsPath)
	if err != nil {
		Logger.Fatal(err)
		return nil
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
	var result []World
	for _, val := range worlds {
		result = append(result, val)
	}
	return result
}

func FindMojangDir() string {
	if runtime.GOOS != "windows" {
		Logger.Fatal(fmt.Sprintf("unsupported OS '%s'", runtime.GOOS))
		return ""
	}
	result := path.Join(os.Getenv("LOCALAPPDATA"), "Packages", "Microsoft.MinecraftUWP_8wekyb3d8bbwe", "LocalState", "games", "com.mojang")
	if _, err := os.Stat(result); os.IsNotExist(err) {
		Logger.Fatal(err)
		return ""
	}
	return result
}
