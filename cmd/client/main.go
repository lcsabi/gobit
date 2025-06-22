// https://bittorrent.org/beps/bep_0003.html

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/lcsabi/gobit/pkg/bencode"
)

func main() {
	f, _ := os.Open("D:\\devstuff\\projects\\gobit\\cmd\\client\\example.torrent")
	value, err := bencode.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(bencode.ToString(value))
}
