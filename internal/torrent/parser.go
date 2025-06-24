package torrent

import (
	"bytes"
	"fmt"
	"os"

	"github.com/lcsabi/gobit/pkg/bencode"
)

// Reference: https://wiki.theory.org/BitTorrentSpecification#Metainfo_File_Structure
type TorrentFile struct {
	Info         InfoDict   // a dictionary that describes the file(s) of the torrent, required
	Announce     string     // tracker URL, required
	AnnounceList [][]string // offers backwards compatibility, optional
	CreationDate int64      // standard UNIX epoch format, optional
	Comment      string     // free-form textual comments of the author, optional
	CreatedBy    string     // name and version of the program used to create the .torrent, optional
	Encoding     string     // the string encoding format used to generate the pieces part of the info dictionary in the .torrent metafile, optional
}

type InfoDict struct {
	PieceLength int64      // number of bytes in each piece, required
	Pieces      [][20]byte // ? concatenation of all 20-byte SHA1 hash values, one per piece, required MAYBE STRING
	Private     *int       // if set to "1", the client MUST publish its presence to get other peers ONLY via the trackers explicitly described in the metainfo file, optional
	Files       []FileInfo // a list of dictionaries, one for each file
	Name        string     // the name of the directory in which to store all the files or the file name if single-file mode, required
}

type FileInfo struct {
	Length int64    // length of the file in bytes, required
	Path   []string // a list containing one or more string elements that together represent the path and filename, required
}

func (t TorrentFile) IsMultiFile() bool {
	return len(t.Info.Files) > 1
}

func Parse(path string) (*TorrentFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	decodedData, err := bencode.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	root, ok := decodedData.(bencode.BencodeDictionary)
	if !ok {
		return nil, fmt.Errorf("invalid torrent structure")
	}

	// parse announce URL
	raw, exists := root["announce"]
	if !exists {
		return nil, fmt.Errorf("announce URL not found")
	}
	announce, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("announce URL is invalid: %T (%v)", announce, announce)
	}

	// parse info dictionary
	raw, exists = root["info"]
	if !exists {
		return nil, fmt.Errorf("info dictionary not found")
	}
	infoDictionary, ok := raw.(map[string]bencode.BencodeValue)
	if !ok {
		return nil, fmt.Errorf("info dictionary is invalid: %T (%v)", announce, announce)
	}

	// create pieces
	/*
		infoBuf := new(bytes.Buffer)
		_ = util.Bencode(infoBuf, infoMap)

		infoHash := sha1.Sum(infoBuf.Bytes())
	*/

	// parse piece length
	raw, exists = infoDictionary["piece length"]
	if !exists {
		return nil, fmt.Errorf("piece length not found")
	}
	pieceLength, ok := raw.(int64)
	if !ok {
		return nil, fmt.Errorf("piece length is invalid: %T (%v)", pieceLength, pieceLength)
	}

	// I would like to always create an array and specify a length + path even for single-file mode

	// parse name

	// decide if single or multi-file

	// parse files dictionary if multi-file

	// populate info dictionary

	return &TorrentFile{
		Announce: announce,
	}, nil
}
