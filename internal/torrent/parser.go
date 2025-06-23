package torrent

import (
	"bytes"
	"fmt"
	"os"

	"github.com/lcsabi/gobit/pkg/bencode"
)

// Reference: https://wiki.theory.org/BitTorrentSpecification#Metainfo_File_Structure
type TorrentFile struct {
	Info         InfoDict   // a dictionary that describes the file(s) of the torrent
	Announce     string     // tracker URL
	AnnounceList [][]string // offers backwards compatibility, optional
	CreationDate int64      // standard UNIX epoch format, optional
	Comment      string     // free-form textual comments of the author, optional
	CreatedBy    string     // name and version of the program used to create the .torrent, optional
	Encoding     string     // the string encoding format used to generate the pieces part of the info dictionary in the .torrent metafile, optional
}

type InfoDict struct {
	PieceLength int64      // number of bytes in each piece
	Pieces      [][20]byte // concatenation of all 20-byte SHA1 hash values, one per piece
	Private     *int       // if set to "1", the client MUST publish its presence to get other peers ONLY via the trackers explicitly described in the metainfo file
	Files       []FileInfo // a list of dictionaries, one for each file
	Name        string     // the name of the directory in which to store all the files
}

type FileInfo struct {
	Length int64    // length of the file in bytes
	Path   []string // a list containing one or more string elements that together represent the path and filename
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

	// Parse announce URL
	announce := root["announce"]
	fmt.Println(announce)

	return nil, nil
}
