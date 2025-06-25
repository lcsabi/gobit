package torrent

import (
	"bytes"
	"fmt"
	"os"

	"github.com/lcsabi/gobit/pkg/bencode"
)

// TODO: reorder struct fields for memory efficiency, visualize with structlayout

// TorrentFile represents the root structure of a .torrent file.
// It includes tracker URLs, metadata, and optional attributes such as comments or encoding.
// Reference: https://wiki.theory.org/BitTorrentSpecification#Metainfo_File_Structure
type TorrentFile struct {
	Info         InfoDict   // info dictionary that describes the file(s) to be shared (required)
	Announce     string     // primary tracker URL (required)
	InfoHash     [20]byte   // SHA-1 hash of the bencoded 'info' dictionary. Used as the unique identifier of the torrent
	AnnounceList [][]string // tiered list of alternative tracker URLs (optional)
	CreationDate int64      // creation time as a UNIX timestamp (optional)
	Comment      string     // free-form comment added by the torrent creator (optional)
	CreatedBy    string     // name and version of the program that created the torrent (optional)
	Encoding     string     // text encoding used for strings (optional)
}

// InfoDict represents the "info" dictionary in the .torrent file.
// It contains file layout, piece information, and privacy flag.
type InfoDict struct {
	PieceLength int64      // number of bytes per piece (required)
	Pieces      [][20]byte // SHA-1 hashes of each piece, sliced into 20-byte blocks (required)
	Name        string     // directory name (multi-file mode) or file name (single-file mode) (required)
	Files       []FileInfo // list of files (single-entry in single-file mode; multiple in multi-file mode)
	Private     *int64     // if 1, restricts peer discovery to trackers only (optional)
}

// FileInfo represents a file within a multi-file torrent.
// Each file includes its length and a path split into components.
type FileInfo struct {
	Length int64    // file size in bytes (required)
	Path   []string // file path as a slice of components (e.g., ["dir", "subdir", "file.ext"]) (required)
}

// TODO: implement NumPieces, FullPath, or TotalLength methods
func (d *InfoDict) IsMultiFile() bool {
	return len(d.Files) > 1
}

func (t *TorrentFile) IsMultiFile() bool {
	return t.Info.IsMultiFile()
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

	result := TorrentFile{} // the return value that needs to be populated

	// announce
	raw, exists := root["announce"]
	if !exists {
		return nil, fmt.Errorf("'announce' not found")
	}
	announce, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("expected 'announce' to be a string, got %T", raw)
	}
	result.Announce = announce // append to result

	// info
	infoDictionary := InfoDict{}
	raw, exists = root["info"]
	if !exists {
		return nil, fmt.Errorf("'info' dictionary not found")
	}
	rootDict, ok := raw.(bencode.BencodeDictionary)
	if !ok {
		return nil, fmt.Errorf("expected 'info' to be a dictionary, got %T", raw)
	}

	// TODO: hash the bencoded info dictionary into InfoHash

	// piece length
	raw, exists = rootDict["piece length"]
	if !exists {
		return nil, fmt.Errorf("'piece length' not found")
	}
	pieceLength, ok := raw.(int64)
	if !ok {
		return nil, fmt.Errorf("expected 'piece length' to be an int64, got %T", raw)
	}
	infoDictionary.PieceLength = pieceLength

	// pieces
	// TODO: split the pieces string into [][20]byte

	// name
	raw, exists = rootDict["name"]
	if !exists {
		return nil, fmt.Errorf("'name' not found")
	}
	name, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("expected 'name' to be a string, got %T", raw)
	}
	infoDictionary.Name = name

	// files
	filesList := []FileInfo{}
	raw, exists = rootDict["files"]
	if !exists {
		fmt.Println("single-file torrent")

		// parse single-file
		raw, exists = rootDict["length"]
		if !exists {
			return nil, fmt.Errorf("'length' not found")
		}
		length, ok := raw.(int64)
		if !ok {
			return nil, fmt.Errorf("expected 'length' to be an int64, got %T", raw)
		}
		filesList = append(filesList, FileInfo{
			Length: length,
			Path:   []string{name},
		})
	} else {
		fmt.Println("multi-file torrent")

		// parse multi-file

	}
	infoDictionary.Files = filesList

	// parse private
	raw, exists = rootDict["private"]
	if !exists {
		fmt.Println("'private' not found")
	} else {
		private, ok := raw.(int64)
		if !ok {
			return nil, fmt.Errorf("expected 'private' to be an int64, got %T", raw)
		} else {
			infoDictionary.Private = &private
		}
	}
	result.Info = infoDictionary // append to result

	// parse announce-list
	// TODO: implement order of processing logic
	raw, exists = root["announce-list"]
	if !exists {
		fmt.Println("'announce-list' not found")
	} else {
		tierList, ok := raw.([]bencode.BencodeValue)
		if !ok {
			fmt.Printf("expected 'announce-list' to be a list of tiers, got %T\n", raw)
		} else {
			var parsedAnnounceList [][]string
			for _, tier := range tierList {
				tierGroup, ok := tier.([]bencode.BencodeValue)
				if !ok {
					fmt.Printf("expected tier in 'announce-list' to be a list, got %T\n", tier)
					continue
				}
				var urls []string
				for _, url := range tierGroup {
					s, ok := url.(string)
					if !ok {
						fmt.Printf("expected URL in 'announce-list' to be a string, got %T\n", url)
						continue
					}
					urls = append(urls, s)
				}
				if len(urls) > 0 {
					parsedAnnounceList = append(parsedAnnounceList, urls)
				}
			}
			result.AnnounceList = parsedAnnounceList
		}
	}

	// parse creation date
	raw, exists = root["creation date"]
	if !exists {
		fmt.Println("'creation date' not found")
	} else {
		creationDate, ok := raw.(int64)
		if !ok {
			fmt.Printf("expected 'creation date' to be an int64, got %T\n", raw)
		} else {
			result.CreationDate = creationDate
		}
	}

	// parse comment
	raw, exists = root["comment"]
	if !exists {
		fmt.Println("'comment' not found")
	} else {
		comment, ok := raw.(string)
		if !ok {
			fmt.Printf("expected 'comment' to be a string, got %T\n", raw)
		} else {
			result.Comment = comment
		}
	}

	// parse created by
	raw, exists = root["created by"]
	if !exists {
		fmt.Println("'created by' not found")
	} else {
		createdBy, ok := raw.(string)
		if !ok {
			fmt.Printf("expected 'created by' to be a string, got %T\n", raw)
		} else {
			result.CreatedBy = createdBy
		}
	}

	// parse encoding
	raw, exists = root["encoding"]
	if !exists {
		fmt.Println("'encoding' not found")
	} else {
		encoding, ok := raw.(string)
		if !ok {
			fmt.Printf("expected 'encoding' to be a string, got %T\n", raw)
		} else {
			result.Encoding = encoding
		}
	}

	return &result, nil
}
