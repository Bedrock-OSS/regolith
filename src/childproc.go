package src

import (
	"log"
	"os"
	"os/exec"
)

func RunChildProc() {
	cmd := exec.Command("python", "./hello.py")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Println(cmd.Run())
}
