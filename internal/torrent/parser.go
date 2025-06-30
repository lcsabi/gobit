package torrent

import (
	"bytes"
	"fmt"
	"os"

	"github.com/lcsabi/gobit/pkg/bencode"
)

const (
	// root-level keys
	keyInfo         = "info"
	keyAnnounce     = "announce"
	keyAnnounceList = "announce-list"
	keyCreationDate = "creation date"
	keyComment      = "comment"
	keyCreatedBy    = "created by"
	keyEncoding     = "encoding"

	// info dictionary keys
	keyName        = "name"
	keyFiles       = "files"
	keyPieceLength = "piece length"
	keyPieces      = "pieces"
	keyPrivate     = "private"

	// file dictionary keys
	keyLength = "length"
	keyPath   = "path"
)

// TODO: reorder struct fields for memory efficiency, visualize with structlayout
// TODO: make sure to parse the required fields first, and the quickest ones from those for efficiency
// TODO: add keys to root level: azureus_properties, add info dict key: source

// File represents the root structure of a .torrent file.
// It includes tracker URLs, metadata, and optional attributes such as comments or encoding.
// Reference: https://wiki.theory.org/BitTorrentSpecification#Metainfo_File_Structure
type File struct {
	Info         InfoDict               // info dictionary that describes the file(s) to be shared (required)
	InfoHash     [20]byte               // SHA-1 hash of the bencoded 'info' dictionary (required)
	Announce     bencode.ByteString     // primary tracker URL (required)
	AnnounceList [][]bencode.ByteString // tiered list of alternative tracker URLs (optional)
	CreationDate bencode.Integer        // creation time as a UNIX timestamp (optional)
	Comment      bencode.ByteString     // free-form comment added by the torrent creator (optional)
	CreatedBy    bencode.ByteString     // name and version of the program that created the torrent (optional)
	Encoding     bencode.ByteString     // used to generate the pieces part of the info dictionary (optional)
}

// InfoDict represents the "info" dictionary in the .torrent file.
// It contains file layout, piece information, and privacy flag.
type InfoDict struct {
	Name        bencode.ByteString // directory name (multi-file mode) or file name (single-file mode) (required)
	Files       []FileInfo         // list of files (single-entry in single-file mode; multiple in multi-file mode)
	PieceLength bencode.Integer    // number of bytes per piece (required)
	Pieces      [][20]byte         // SHA-1 hashes of each piece, sliced into 20-byte blocks (required)
	Private     *bencode.Integer   // if 1, restricts peer discovery to trackers only (optional)
}

// FileInfo represents a file within a multi-file torrent.
// Each file includes its length and a path split into components.
type FileInfo struct {
	Length bencode.Integer      // file size in bytes (required)
	Path   []bencode.ByteString // file path as a slice of components (required)
}

// TODO: implement NumPieces, FullPath, or TotalLength methods
// TODO: create Torrent file linter / validator
// TODO: create Torrent file editor / repair tool

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
	var result *File

	// announce
	if err := result.parseAnnounce(root); err != nil {
		return nil, err
	}

	// info
	if err := result.parseInfo(root); err != nil {
		return nil, err
	}

	// pieces
	// TODO: split the pieces string into [][20]byte

	// parse announce-list
	// TODO: implement order of processing logic
	raw, exists = root["announce-list"]
	if !exists {
		fmt.Println("'announce-list' not found") // TODO: change to log or remove
	} else {
		tierList, ok := raw.([]bencode.BencodeValue)
		if !ok {
			fmt.Printf("parsing 'announce-list': expected list of tiers, got %T\n", raw)
		} else {
			var parsedAnnounceList [][]string
			for _, tier := range tierList {
				tierGroup, ok := tier.([]bencode.BencodeValue)
				if !ok {
					fmt.Printf("parsing tier in 'announce-list': expected list of ByteStrings, got %T\n", tier)
					continue
				}
				var urls []string
				for _, url := range tierGroup {
					s, ok := url.(string)
					if !ok {
						fmt.Printf("parsing URLs in 'announce-list' tier: expected string, got %T\n", url)
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

	return result, nil
}

// =====================================================================================

func (t *File) parseAnnounce(root bencode.Dictionary) error {
	raw, exists := root["announce"]
	if !exists {
		return fmt.Errorf("'announce' key not found")
	}
	announce, ok := raw.(bencode.ByteString)
	if !ok {
		return fmt.Errorf("parsing 'announce': expected bencode.ByteString, got %T", raw)
	}

	t.Announce = announce
	return nil
}

func (t *File) parseInfo(root bencode.Dictionary) error {
	var infoDictionary InfoDict
	raw, exists := root["info"]
	if !exists {
		return fmt.Errorf("'info' key not found")
	}
	info, ok := raw.(bencode.Dictionary)
	if !ok {
		return fmt.Errorf("parsing 'info': expected bencode.Dictionary, got %T", raw)
	}

	// name
	if err := infoDictionary.parseName(info); err != nil {
		return err
	}

	// piece length
	if err := infoDictionary.parsePieceLength(info); err != nil {
		return err
	}

	// files
	if err := infoDictionary.parseFiles(info); err != nil {
		return err
	}

	// pieces
	if err := infoDictionary.parsePieces(info); err != nil {
		return err
	}

	// private
	if err := infoDictionary.parsePrivate(info); err != nil {
		return err
	}

	t.Info = infoDictionary
	return nil
}

func (i *InfoDict) parseName(infoRoot bencode.Dictionary) error {
	raw, exists := infoRoot["name"]
	if !exists {
		return fmt.Errorf("'name' key not found")
	}
	name, ok := raw.(bencode.ByteString)
	if !ok {
		return fmt.Errorf("parsing 'name': expected bencode.ByteString, got %T", raw)
	}

	i.Name = name
	return nil
}

func (i *InfoDict) parseFiles(infoRoot bencode.Dictionary) error {
	var fileInfoList []FileInfo
	raw, exists := infoRoot["files"]
	if !exists {
		// Single-file mode
		fmt.Println("single-file mode torrent") // TODO: change to log or remove
		length, err := parseFileLength(infoRoot)
		if err != nil {
			return fmt.Errorf("parsing single-file mode torrent 'length': %w", err)
		}

		fileInfoList = append(fileInfoList, FileInfo{
			Length: length,
			Path:   []string{i.Name}, // by this point, it's guaranteed that i.Name exists
		})
	} else {
		// Multi-file mode
		fmt.Println("multi-file mode torrent")  // TODO: change to log or remove
		multiFileList, ok := raw.(bencode.List) // contains dictionaries with file path and length
		if !ok {
			return fmt.Errorf("parsing 'files': expected bencode.List, got %T", raw)
		}
		for idx, elem := range multiFileList {
			multiFileDict, ok := elem.(bencode.Dictionary) // contains file path and length keys
			if !ok {
				return fmt.Errorf("parsing entry %d in 'files': expected bencode.Dictionary, got %T", idx, elem)
			}

			length, err := parseFileLength(multiFileDict)
			if err != nil {
				return fmt.Errorf("parsing file length at index %d: %w", idx, err)
			}
			path, err := parseFilePath(multiFileDict)
			if err != nil {
				return fmt.Errorf("parsing file path at index %d: %w", idx, err)
			}

			fileInfoList = append(fileInfoList, FileInfo{
				Length: length,
				Path:   path,
			})
		}
	}

	i.Files = fileInfoList
	return nil
}

func (i *InfoDict) parsePieceLength(infoRoot bencode.Dictionary) error {
	raw, exists := infoRoot["piece length"]
	if !exists {
		return fmt.Errorf("'piece length' key not found")
	}
	pieceLength, ok := raw.(bencode.Integer)
	if !ok {
		return fmt.Errorf("parsing 'piece length': expected bencode.Integer, got %T", raw)
	}

	i.PieceLength = pieceLength
	return nil
}

// ! continue here
// implement parsePieces() here

func (i *InfoDict) parsePrivate(infoRoot bencode.Dictionary) error {
	raw, exists := infoRoot["private"]
	if !exists {
		fmt.Println("'private' key not found") // TODO: change to log or remove
	} else {
		private, ok := raw.(bencode.Integer)
		if !ok {
			return fmt.Errorf("parsing 'private': expected bencode.Integer, got %T", raw)
		} else {
			// we return a pointer just to make sure nil can get handled
			// even though decoding should guarantee no nil value is passed
			i.Private = &private
		}
	}

	return nil
}

func parseFileLength(rootDict bencode.Dictionary) (bencode.Integer, error) {
	raw, exists := rootDict["length"]
	if !exists {
		return 0, fmt.Errorf("'length' key not found")
	}
	length, ok := raw.(bencode.Integer)
	if !ok {
		return 0, fmt.Errorf("parsing 'length': expected bencode.Integer, got %T", raw)
	}

	return length, nil
}

func parseFilePath(rootDict bencode.Dictionary) ([]bencode.ByteString, error) {
	raw, exists := rootDict["path"]
	if !exists {
		return nil, fmt.Errorf("'path' key not found")
	}
	path, ok := raw.([]bencode.ByteString)
	if !ok {
		return nil, fmt.Errorf("parsing 'path': expected []bencode.ByteString, got %T", raw)
	}

	return path, nil
}

// TODO: implement createInfoHash() here, hash the bencoded info dictionary into InfoHash, probably do this before optional fields are parsed

// TODO: AnnounceList

func (t *File) parseCreationDate(root bencode.Dictionary) {
	raw, exists := root["creation date"]
	if !exists {
		fmt.Println("'creation date' not found") // TODO: change to log or remove
	} else {
		creationDate, ok := raw.(bencode.Integer)
		if !ok {
			fmt.Printf("parsing 'creation date': expected bencode.Integer, got %T\n", raw)
		} else {
			t.CreationDate = creationDate
		}
	}
}

func (t *File) parseComment(root bencode.Dictionary) {
	raw, exists := root["comment"]
	if !exists {
		fmt.Println("'comment' not found") // TODO: change to log or remove
	} else {
		comment, ok := raw.(bencode.ByteString)
		if !ok {
			fmt.Printf("parsing 'comment': expected string, got %T\n", raw)
		} else {
			t.Comment = comment
		}
	}
}

func (t *File) parseCreatedBy(root bencode.Dictionary) {
	raw, exists := root["created by"]
	if !exists {
		fmt.Println("'created by' not found") // TODO: change to log or remove
	} else {
		createdBy, ok := raw.(bencode.ByteString)
		if !ok {
			fmt.Printf("parsing 'created by': expected bencode.ByteString, got %T\n", raw)
		} else {
			t.CreatedBy = createdBy
		}
	}
}

func (t *File) parseEncoding(root bencode.Dictionary) {
	raw, exists := root["encoding"]
	if !exists {
		fmt.Println("'encoding' not found") // TODO: change to log or remove
	} else {
		encoding, ok := raw.(bencode.ByteString)
		if !ok {
			fmt.Printf("parsing 'encoding': expected bencode.ByteString, got %T\n", raw)
		} else {
			t.Encoding = encoding
		}
	}
}
