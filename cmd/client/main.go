// https://bittorrent.org/beps/bep_0003.html

package main

import (
	"fmt"

	"github.com/lcsabi/gobit/internal/torrent"
)

func main() {
	file, err := torrent.Parse("D:\\devstuff\\projects\\gobit\\cmd\\client\\single.example.torrent")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%v\n", file.Announce)
	fmt.Printf("%v\n", file.AnnounceList)
	fmt.Printf("%v\n", file.Info.Name)
	fmt.Printf("%v\n", file.Info.PieceLength)
	fmt.Printf("%v\n", file.Info.Files)
	fmt.Printf("%v\n", *file.Info.Private)
}
