// https://bittorrent.org/beps/bep_0003.html

package main

import (
	"fmt"

	"github.com/lcsabi/gobit/internal/torrent"
)

func main() {
	_, err := torrent.Parse("D:\\devstuff\\projects\\gobit\\cmd\\client\\single.example.torrent")
	if err != nil {
		fmt.Println(err)
	}
}
