package torrent

import (
	"bytes"
	"fmt"
	"os"

	"github.com/lcsabi/gobit/pkg/bencode"
)

// TODO: reorder struct fields for memory efficiency, visualize with structlayout
// TODO: make sure to parse the required fields first, and the quickest ones from those for efficiency

// File represents the root structure of a .torrent file.
// It includes tracker URLs, metadata, and optional attributes such as comments or encoding.
// Reference: https://wiki.theory.org/BitTorrentSpecification#Metainfo_File_Structure
type File struct {
	Info         InfoDict   // info dictionary that describes the file(s) to be shared (required)
	InfoHash     [20]byte   // SHA-1 hash of the bencoded 'info' dictionary used as the unique identifier (required)
	Announce     string     // primary tracker URL (required)
	AnnounceList [][]string // tiered list of alternative tracker URLs (optional)
	CreationDate int64      // creation time as a UNIX timestamp (optional)
	Comment      string     // free-form comment added by the torrent creator (optional)
	CreatedBy    string     // name and version of the program that created the torrent (optional)
	Encoding     string     // used to generate the pieces part of the info dictionary (optional)
}

// InfoDict represents the "info" dictionary in the .torrent file.
// It contains file layout, piece information, and privacy flag.
type InfoDict struct {
	Name        string     // directory name (multi-file mode) or file name (single-file mode) (required)
	Files       []FileInfo // list of files (single-entry in single-file mode; multiple in multi-file mode)
	PieceLength int64      // number of bytes per piece (required)
	Pieces      [][20]byte // SHA-1 hashes of each piece, sliced into 20-byte blocks (required)
	Private     *int64     // if 1, restricts peer discovery to trackers only (optional)
}

// FileInfo represents a file within a multi-file torrent.
// Each file includes its length and a path split into components.
type FileInfo struct {
	Length int64    // file size in bytes (required)
	Path   []string // file path as a slice of components (required)
}

// TODO: implement NumPieces, FullPath, or TotalLength methods

func (i *InfoDict) IsMultiFile() bool {
	return len(i.Files) > 1
}

func (t *File) IsMultiFile() bool {
	return t.Info.IsMultiFile()
}

func Parse(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decodedData, err := bencode.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	root, ok := decodedData.(bencode.Dictionary)
	if !ok {
		return nil, fmt.Errorf("invalid torrent structure")
	}
	result := &File{}

	// announce
	if err := result.parseAnnounce(root); err != nil {
		return nil, err
	}

	// info
	if err := result.parseInfo(root); err != nil {
		return nil, err
	}

	// TODO: hash the bencoded info dictionary into InfoHash, probably do this last after torrent is parsed

	// pieces
	// TODO: split the pieces string into [][20]byte

	// files
	filesList := []FileInfo{}
	raw, exists = rootDict["files"]
	if !exists {
		fmt.Println("single-file torrent")

		// parse single-file
		raw, exists = rootDict["length"]
		if !exists {
			return nil, fmt.Errorf("'length' key not found")
		}
		length, ok := raw.(int64)
		if !ok {
			return nil, fmt.Errorf("expected 'length' value to be an int64, got %T", raw)
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

	return result, nil
}

func (t *File) parseAnnounce(root bencode.Dictionary) error {
	raw, exists := root["announce"]
	if !exists {
		return fmt.Errorf("'announce' key not found")
	}
	announce, ok := raw.(string)
	if !ok {
		return fmt.Errorf("expected 'announce' value to be a string, got %T", raw)
	}
	t.Announce = announce

	return nil
}

func (t *File) parseInfo(root bencode.Dictionary) error {
	infoDictionary := InfoDict{}
	raw, exists := root["info"]
	if !exists {
		return fmt.Errorf("'info' key not found")
	}
	info, ok := raw.(bencode.Dictionary)
	if !ok {
		return fmt.Errorf("expected 'info' value to be a dictionary, got %T", raw)
	}

	// piece length
	if err := infoDictionary.parsePieceLength(info); err != nil {
		return err
	}

	// name
	if err := infoDictionary.parseName(info); err != nil {
		return err
	}

	// files
	if err := infoDictionary.parseFiles(info); err != nil {
		return err
	}

	// pieces

	// private
	if err := infoDictionary.parsePrivate(info); err != nil {
		return err
	}

	t.Info = infoDictionary

	return nil
}

func (i *InfoDict) parsePieceLength(infoRoot bencode.Dictionary) error {
	raw, exists := infoRoot["piece length"]
	if !exists {
		return fmt.Errorf("'piece length' key not found")
	}
	pieceLength, ok := raw.(int64)
	if !ok {
		return fmt.Errorf("expected 'piece length' value to be an int64, got %T", raw)
	}
	i.PieceLength = pieceLength

	return nil
}

func (i *InfoDict) parseName(infoRoot bencode.Dictionary) error {
	raw, exists := infoRoot["name"]
	if !exists {
		return fmt.Errorf("'name' key not found")
	}
	name, ok := raw.(string)
	if !ok {
		return fmt.Errorf("expected 'name' value to be a string, got %T", raw)
	}
	i.Name = name

	return nil
}

func (i *InfoDict) parseFiles(infoRoot bencode.Dictionary) error {
	fileInfoList := []FileInfo{}
	// ! continue here
	i.Files = fileInfoList
	return nil
}

func (i *InfoDict) parsePrivate(infoRoot bencode.Dictionary) error {
	raw, exists := infoRoot["private"]
	if !exists {
		fmt.Println("'private' key not found")
	} else {
		private, ok := raw.(int64)
		if !ok {
			return fmt.Errorf("expected 'private' value to be an int64, got %T", raw)
		} else {
			i.Private = &private
		}
	}

	return nil
}
