package src

import (
	"fmt"

	getter "github.com/hashicorp/go-getter"
)

func DownloadFileTest() {
	fmt.Println("HELLO WORD!")
	fileUrl := "https://github.com/Bedrock-OSS/regolith-filters/tree/master/texture_list"

	getter.Get(".regolith/cache", fileUrl)
}
