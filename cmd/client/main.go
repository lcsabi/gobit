// https://bittorrent.org/beps/bep_0003.html

package main

import (
	"github.com/lcsabi/gobit/internal/torrent"
)

func main() {
	torrent.Parse("D:\\devstuff\\projects\\gobit\\cmd\\client\\single.example.torrent")
}
