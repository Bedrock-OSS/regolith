package src

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
)

type World struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func ListWorlds(mojangDir string) []World {
	var result []World
	worldsPath := path.Join(mojangDir, "minecraftWorlds")
	files, err := ioutil.ReadDir(worldsPath)
	if err != nil {
		Logger.Fatal(err)
		return nil
	}
	for _, f := range files {
		if f.IsDir() {
			worldPath := path.Join(worldsPath, f.Name())
			worldname, _ := ioutil.ReadFile(path.Join(worldPath, "levelname.txt"))
			result = append(result, World{
				Name: string(worldname),
				Path: worldPath,
			})
		}
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
