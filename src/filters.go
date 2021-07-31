package src

import (
	"log"
	"os"
	"os/exec"
)

func RunProfile(profile string) {
	log.Println("Configuration:", profile)
	var project Project = LoadConfig()
	log.Println(project)
}

func RunChildProccess() {
	cmd := exec.Command("python", "./hello.py")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Println(cmd.Run())
}
