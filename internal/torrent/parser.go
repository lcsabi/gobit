package torrent

import (
	"bytes"
	"errors"
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

func (t *File) IsMultiFile() bool {
	return t.Info.IsMultiFile()
}

func (i *InfoDict) IsMultiFile() bool {
	return len(i.Files) > 1
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
		return nil, errors.New("invalid torrent structure")
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

	// announce-list
	result.parseAnnounceList(root)

	// creation date
	result.parseCreationDate(root)

	// comment
	result.parseComment(root)

	// created by
	result.parseCreatedBy(root)

	// encoding
	result.parseEncoding(root)

	return result, nil
}

// =====================================================================================

func (t *File) parseAnnounce(root bencode.Dictionary) error {
	raw, exists := root[keyAnnounce]
	if !exists {
		return fmt.Errorf("'%s' key not found", keyAnnounce)
	}
	announce, ok := raw.(bencode.ByteString)
	if !ok {
		return fmt.Errorf("parsing '%s': expected bencode.ByteString, got %T", keyAnnounce, raw)
	}

	t.Announce = announce
	return nil
}

func (t *File) parseInfo(root bencode.Dictionary) error {
	var infoDictionary InfoDict
	raw, exists := root[keyInfo]
	if !exists {
		return fmt.Errorf("'%s' key not found", keyInfo)
	}
	info, ok := raw.(bencode.Dictionary)
	if !ok {
		return fmt.Errorf("parsing '%s': expected bencode.Dictionary, got %T", keyInfo, raw)
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
	infoDictionary.parsePrivate(info)

	t.Info = infoDictionary
	return nil
}

func (i *InfoDict) parseName(infoRoot bencode.Dictionary) error {
	raw, exists := infoRoot[keyName]
	if !exists {
		return fmt.Errorf("'%s' key not found", keyName)
	}
	name, ok := raw.(bencode.ByteString)
	if !ok {
		return fmt.Errorf("parsing '%s': expected bencode.ByteString, got %T", keyName, raw)
	}

	i.Name = name
	return nil
}

func (i *InfoDict) parseFiles(infoRoot bencode.Dictionary) error {
	var fileInfoList []FileInfo
	raw, exists := infoRoot[keyFiles]
	if !exists {
		// single-file mode
		fmt.Println("single-file mode torrent") // TODO: change to log or remove
		length, err := parseFileLength(infoRoot)
		if err != nil {
			return fmt.Errorf("parsing single-file mode torrent '%s': %w", keyLength, err)
		}

		fileInfoList = append(fileInfoList, FileInfo{
			Length: length,
			Path:   []string{i.Name}, // by this point, it's guaranteed that i.Name is not nil
		})
	} else {
		// multi-file mode
		fmt.Println("multi-file mode torrent")  // TODO: change to log or remove
		multiFileList, ok := raw.(bencode.List) // contains dictionaries with file path and length
		if !ok {
			return fmt.Errorf("parsing '%s': expected bencode.List, got %T", keyFiles, raw)
		}
		for idx, elem := range multiFileList {
			multiFileDict, ok := elem.(bencode.Dictionary) // contains file path and length keys
			if !ok {
				return fmt.Errorf("parsing entry %d in '%s': expected bencode.Dictionary, got %T", idx, keyFiles, elem)
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
	raw, exists := infoRoot[keyPieceLength]
	if !exists {
		return fmt.Errorf("'%s' key not found", keyPieceLength)
	}

	pieceLength, ok := raw.(bencode.Integer)
	if !ok {
		return fmt.Errorf("parsing '%s': expected bencode.Integer, got %T", keyPieceLength, raw)
	}

	i.PieceLength = pieceLength
	return nil
}

// ! continue here
// implement parsePieces() here

func (i *InfoDict) parsePrivate(infoRoot bencode.Dictionary) {
	raw, exists := infoRoot[keyPrivate]
	if !exists {
		fmt.Printf("'%s' key not found\n", keyPrivate) // TODO: change to log or remove
		return
	}

	private, ok := raw.(bencode.Integer)
	if !ok {
		fmt.Printf("parsing '%s': expected bencode.Integer, got %T\n", keyPrivate, raw) // TODO: change to log or remove
		return
	}

	// we return a pointer just to make sure nil can get handled
	// even though decoding should guarantee no nil value is passed
	i.Private = &private
}

func parseFileLength(root bencode.Dictionary) (bencode.Integer, error) {
	raw, exists := root[keyLength]
	if !exists {
		return 0, fmt.Errorf("'%s' key not found", keyLength)
	}

	length, ok := raw.(bencode.Integer)
	if !ok {
		return 0, fmt.Errorf("parsing '%s': expected bencode.Integer, got %T", keyLength, raw)
	}

	return length, nil
}

func parseFilePath(root bencode.Dictionary) ([]bencode.ByteString, error) {
	raw, exists := root[keyPath]
	if !exists {
		return nil, fmt.Errorf("'%s' key not found", keyPath)
	}

	path, ok := raw.([]bencode.ByteString)
	if !ok {
		return nil, fmt.Errorf("parsing '%s': expected []bencode.ByteString, got %T", keyPath, raw)
	}

	return path, nil
}

// TODO: implement createInfoHash() here, hash the bencoded info dictionary into InfoHash, probably do this before optional fields are parsed

// Reference: https://bittorrent.org/beps/bep_0012.html
func (t *File) parseAnnounceList(root bencode.Dictionary) {
	raw, exists := root[keyAnnounceList]
	if !exists {
		fmt.Printf("'%s' key not found\n", keyAnnounceList) // TODO: change to log or remove
		return
	}

	rawList, ok := raw.(bencode.List)
	if !ok {
		fmt.Printf("parsing '%s': expected bencode.List, got %T\n", keyAnnounceList, raw) // TODO: change to log or remove
		return
	}

	var parsedAnnounceList [][]string
	for tierCount, tierRaw := range rawList {
		tierList, ok := tierRaw.(bencode.List)
		if !ok {
			fmt.Printf("parsing tier #%d: expected bencode.List, got %T\n", tierCount, tierRaw)
			continue
		}

		var urls []string
		for urlCount, urlRaw := range tierList {
			urlStr, ok := urlRaw.(bencode.ByteString)
			if !ok {
				fmt.Printf("parsing URL #%d in tier #%d: expected string, got %T\n", tierCount, urlCount, urlRaw)
				continue
			}
			urls = append(urls, urlStr)
		}

		if len(urls) > 0 {
			parsedAnnounceList = append(parsedAnnounceList, urls)
		}
	}

	t.AnnounceList = parsedAnnounceList
}

func (t *File) parseCreationDate(root bencode.Dictionary) {
	raw, exists := root[keyCreationDate]
	if !exists {
		fmt.Printf("'%s' not found\n", keyCreationDate) // TODO: change to log or remove
		return
	}

	creationDate, ok := raw.(bencode.Integer)
	if !ok {
		fmt.Printf("parsing '%s': expected bencode.Integer, got %T\n", keyCreationDate, raw)
		return
	}

	t.CreationDate = creationDate
}

func (t *File) parseComment(root bencode.Dictionary) {
	raw, exists := root[keyComment]
	if !exists {
		fmt.Printf("'%s' not found\n", keyComment) // TODO: change to log or remove
		return
	}

	comment, ok := raw.(bencode.ByteString)
	if !ok {
		fmt.Printf("parsing '%s': expected string, got %T\n", keyComment, raw) // TODO: change to log or remove
		return
	}

	t.Comment = comment
}

func (t *File) parseCreatedBy(root bencode.Dictionary) {
	raw, exists := root[keyCreatedBy]
	if !exists {
		fmt.Printf("'%s' not found\n", keyCreatedBy) // TODO: change to log or remove
		return
	}

	createdBy, ok := raw.(bencode.ByteString)
	if !ok {
		fmt.Printf("parsing '%s': expected bencode.ByteString, got %T\n", keyCreatedBy, raw) // TODO: change to log or remove
		return
	}

	t.CreatedBy = createdBy
}

func (t *File) parseEncoding(root bencode.Dictionary) {
	raw, exists := root[keyEncoding]
	if !exists {
		fmt.Printf("'%s' not found\n", keyEncoding) // TODO: change to log or remove
		return
	}

	encoding, ok := raw.(bencode.ByteString)
	if !ok {
		fmt.Printf("parsing '%s': expected bencode.ByteString, got %T\n", keyEncoding, raw) // TODO: change to log or remove
		return
	}

	t.Encoding = encoding
}
