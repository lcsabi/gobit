# bencode

This package implements a performant Bencode encoder and decoder as defined in the [BitTorrent Specification](https://wiki.theory.org/BitTorrentSpecification#Bencoding).

## Features

- Full support for byte strings, integers, lists, and dictionaries
- Streaming-friendly parser using `bytes.Reader`
- Pretty-print and type-inspection utilities
- Input validation (e.g. leading zeros, negative zero, max byte string length)

## Usage

```go
import (
    "os"
    "github.com/lcsabi/gobit/pkg/bencode"
)

func main() {
    f, _ := os.Open("example.torrent")
    value, err := bencode.Decode(f)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(bencode.ToString(value))
}
