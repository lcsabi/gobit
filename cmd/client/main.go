// https://bittorrent.org/beps/bep_0003.html

package main

import (
	"fmt"
	"os"

	"github.com/lcsabi/gobit/pkg/bencode"
)

func main() {
	fmt.Println("Hello, world!")
	file, err := os.Open("D:\\devstuff\\projects\\gobit\\cmd\\client\\multifile_test.torrent")
	if err != nil {
		fmt.Println(err)
	}
	content, err := bencode.Decode(file)
	if err != nil {
		fmt.Println(err)
	}
	contents := content.(map[string]bencode.BencodeValue)
	infoDict := contents["info"].(map[string]bencode.BencodeValue)
	fmt.Printf("%#v\n", infoDict["files"])
}
