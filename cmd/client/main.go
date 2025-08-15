package main

import (
	"fmt"

	"github.com/lcsabi/gobit/internal/torrent"
)

func main() {
	file, err := torrent.Parse("D:\\devstuff\\projects\\gobit\\cmd\\client\\info_hash2.torrent")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%v\n", file.Announce)
	fmt.Printf("%v\n", file.AnnounceList)
	fmt.Printf("%v\n", file.Info.Name)
	fmt.Printf("%v\n", file.Info.PieceLength)
	fmt.Printf("%v\n", file.Info.Files)
	fmt.Printf("%x\n", file.InfoHash)
}
