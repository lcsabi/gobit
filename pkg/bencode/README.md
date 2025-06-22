# bencode

This package implements a performant Bencode encoder and decoder as defined in the [BitTorrent Specification](https://wiki.theory.org/BitTorrentSpecification#Bencoding).

## Features

- Full support for bencoded types:
  - Byte strings (`BencodeByteString`)
  - Integers (`BencodeInteger`)
  - Lists (`BencodeList`)
  - Dictionaries (`BencodeDictionary`)
- Streaming-friendly parser using `bytes.Reader` for scalability
- Pretty-printer (`ToString`) for human-readable debugging
- Type introspection utility (`TypeOf`)
- Secure and robust decoding:
  - Enforces integer format (no leading zeros or negative zero)
  - Rejects malformed or unknown types
  - Limits byte string length to prevent memory exhaustion (default: 10MB)
- Deterministic dictionary encoding (keys are sorted)
- Allocates efficiently using reusable buffers (via `EncodeTo`)
- Idiomatic Go API for general-purpose use beyond `.torrent` files

## Usage

### Decoding

```go
f, _ := os.Open("example.torrent")
value, err := bencode.Decode(f)
if err != nil {
	log.Fatal(err)
}

fmt.Println(bencode.ToString(value))
```

For structured inspection or for debugging purposes:

```go
fmt.Println(bencode.ToString(data))
```

Prints:

```
dictionary:
  key: "announce"
    string: "http://tracker.example.com/announce"
  key: "info"
    dictionary:
      key: "length"
        integer: 12345
      key: "name"
        string: "example.txt"
```

### Encoding

You can encode Go data into bencoded format using `Encode` or `EncodeTo`.

```go
data := bencode.BencodeDictionary{
	"announce": bencode.BencodeByteString("http://tracker.example.com/announce"),
	"info": bencode.BencodeDictionary{
		"name":   bencode.BencodeByteString("example.txt"),
		"length": bencode.BencodeInteger(12345),
	},
}

encoded, err := bencode.Encode(data)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Bencoded output: %s\n", encoded)
```

### Output

```
Bencoded output: d8:announce36:http://tracker.example.com/announce4:infod6:lengthi12345e4:name11:example.txte
```

## Types

- `type BencodeValue = any`
- `type BencodeByteString = string`
- `type BencodeInteger = int64`
- `type BencodeList = []BencodeValue`
- `type BencodeDictionary = map[string]BencodeValue`

## TODO

- Add streaming `Decoder` type for parsing torrent files that exceed 1MB
