package src

import (
	"fmt"
	"log"
	"os/exec"
)

func RunChildProc() {
	out, err := exec.Command("date").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("The date is %s\n", out)
}
