# bencode

This package implements a performant Bencode encoder and decoder as defined in the [BitTorrent Specification](https://wiki.theory.org/BitTorrentSpecification#Bencoding).

## Features

- Full support for byte strings, integers, lists, and dictionaries
- Streaming-friendly parser using `bytes.Reader`
- Pretty-print and type-inspection utilities
- Robust validation (e.g. leading zeros, negative zero, max byte string length)

## Usage

### Decoding

```go
f, _ := os.Open("example.torrent")
value, err := bencode.Decode(f)
if err != nil {
	log.Fatal(err)
}

fmt.Println(bencode.ToString(value)) // Pretty print
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

### Debugging

For structured inspection:

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

## Types

- `type BencodeValue = any`
- `type BencodeByteString = string`
- `type BencodeInteger = int64`
- `type BencodeList = []BencodeValue`
- `type BencodeDictionary = map[string]BencodeValue`

## TODO

- Add streaming `Decoder` type
