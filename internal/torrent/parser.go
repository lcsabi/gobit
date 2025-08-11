package torrent

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lcsabi/gobit/pkg/bencode"
)

// store dictionary keys
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

const MaxTorrentSize = 10 * 1024 * 1024 // 10 MB

// TODO: reorder struct fields for memory efficiency, visualize with structlayout
// TODO: make sure to parse the required fields first, and the quickest ones from those for efficiency
// TODO: add keys to root level: azureus_properties, add info dict key: source
// TODO: add ToString() method

// MetaInfo represents the root structure of a .torrent file.
// It includes tracker URLs, metadata, and optional attributes such as comments or encoding.
// Reference: https://wiki.theory.org/BitTorrentSpecification#Metainfo_File_Structure
type MetaInfo struct {
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
// TODO: consider creating debug builds for logging

func (t *MetaInfo) IsMultiFile() bool {
	return t.Info.IsMultiFile()
}

func (i *InfoDict) IsMultiFile() bool {
	return len(i.Files) > 1
}

func Parse(path string) (*MetaInfo, error) {
	data, path, err := readTorrentFile(path)
	if err != nil {
		return nil, err
	}

	decodedData, err := bencode.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	root, err := bencode.AsDictionary(decodedData)
	if err != nil {
		return nil, fmt.Errorf("expected bencoded dictionary at top-level of %s", path)
	}
	result := MetaInfo{}

	// announce
	if err := result.parseAnnounce(root); err != nil {
		return nil, err
	}

	// info
	if err := result.parseInfo(root); err != nil {
		return nil, err
	}

	// create information hash
	infoHash, err := createInfoHash(root)
	if err != nil {
		return nil, err
	}
	result.InfoHash = infoHash

	result.parseAnnounceList(root)
	result.parseCreationDate(root)
	result.parseComment(root)
	result.parseCreatedBy(root)
	result.parseEncoding(root)

	return &result, nil
}

// =====================================================================================

func readTorrentFile(path string) ([]byte, string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, "", errors.New("empty path provided")
	}

	extension := filepath.Ext(path)
	if strings.ToLower(extension) != ".torrent" {
		return nil, "", fmt.Errorf("invalid file extension: expected .torrent, got: %q", extension)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	cleaned := filepath.Clean(absPath)

	// TODO: add logging
	info, err := os.Stat(cleaned)
	if err != nil {
		return nil, "", fmt.Errorf("failed to stat file: %w", err)
	}
	if info.Size() > MaxTorrentSize {
		return nil, "", fmt.Errorf("torrent file too large (%d bytes), max allowed is %d bytes", info.Size(), MaxTorrentSize)
	}

	data, err := os.ReadFile(cleaned)
	if err != nil {
		return nil, "", err
	}
	return data, cleaned, nil
}

func (t *MetaInfo) parseAnnounce(root bencode.Dictionary) error {
	raw, exists := root[keyAnnounce]
	if !exists {
		return fmt.Errorf("'%s' key not found", keyAnnounce)
	}

	announce, err := bencode.AsByteString(raw)
	if err != nil {
		return fmt.Errorf("parsing '%s': %w", keyAnnounce, err)
	}

	t.Announce = announce
	return nil
}

func (t *MetaInfo) parseInfo(root bencode.Dictionary) error {
	var infoDictionary InfoDict
	raw, exists := root[keyInfo]
	if !exists {
		return fmt.Errorf("'%s' key not found", keyInfo)
	}

	info, err := bencode.AsDictionary(raw)
	if err != nil {
		return fmt.Errorf("parsing '%s': %w", keyInfo, err)
	}

	// piece length
	if err := infoDictionary.parsePieceLength(info); err != nil {
		return err
	}

	// pieces
	if err := infoDictionary.parsePieces(info); err != nil {
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

	name, err := bencode.AsByteString(raw)
	if err != nil {
		return fmt.Errorf("parsing '%s': %w", keyName, err)
	}

	i.Name = filepath.Clean(name) // remvove any unwanted garbage
	return nil
}

func (i *InfoDict) parseFiles(infoRoot bencode.Dictionary) error {
	var fileInfoList []FileInfo
	raw, exists := infoRoot[keyFiles]
	if !exists {
		// single-file mode
		fmt.Println("detected single-file mode torrent") // TODO: change to log or remove
		length, err := parseFileLength(infoRoot)
		if err != nil {
			return fmt.Errorf("parsing single-file mode torrent '%s': %w", keyLength, err)
		}

		fileInfoList = append(fileInfoList, FileInfo{
			Length: length,
			Path:   []string{i.Name}, // by this point, it's guaranteed i.Name is not nil
		})
	} else {
		// multi-file mode
		fmt.Println("detected multi-file mode torrent") // TODO: change to log or remove
		multiFileList, err := bencode.AsList(raw)       // contains dictionaries with file path and length
		if err != nil {
			return fmt.Errorf("parsing '%s': %w", keyFiles, err)
		}
		for idx, elem := range multiFileList {
			multiFileDict, err := bencode.AsDictionary(elem) // contains file path and length keys
			if err != nil {
				return fmt.Errorf("parsing entry %d in '%s': %w", idx, keyFiles, err)
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

	pieceLength, err := bencode.AsInteger(raw)
	if err != nil {
		return fmt.Errorf("parsing '%s': %w", keyPieceLength, err)
	}

	// avoid potential division by zero or buffers with zero length
	if pieceLength <= 0 {
		return fmt.Errorf("invalid '%s': must be non-negative, got %d", keyPieceLength, pieceLength)
	}

	i.PieceLength = pieceLength
	return nil
}

func (i *InfoDict) parsePieces(infoRoot bencode.Dictionary) error {
	raw, exists := infoRoot[keyPieces]
	if !exists {
		return fmt.Errorf("'%s' key not found", keyPieces)
	}

	piecesByteString, err := bencode.AsByteString(raw)
	if err != nil {
		return fmt.Errorf("parsing '%s': %w", keyPieces, err)
	}

	if len(piecesByteString)%20 != 0 {
		return fmt.Errorf("invalid '%s' length: not divisible by 20", keyPieces)
	}

	pieceCount := len(piecesByteString) / 20 // prealloacate for large files
	completeList := make([][20]byte, 0, pieceCount)
	for i := 0; i < len(piecesByteString); i += 20 {
		var chunk [20]byte
		end := i + 20
		copy(chunk[:], piecesByteString[i:end])
		completeList = append(completeList, chunk)
	}

	i.Pieces = completeList
	return nil
}

func (i *InfoDict) parsePrivate(infoRoot bencode.Dictionary) {
	raw, exists := infoRoot[keyPrivate]
	if !exists {
		fmt.Printf("'%s' key not found\n", keyPrivate) // TODO: change to log or remove
		return
	}

	private, err := bencode.AsInteger(raw)
	if err != nil {
		fmt.Printf("parsing '%s': %v\n", keyPrivate, err) // TODO: change to log or remove
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

	length, err := bencode.AsInteger(raw)
	if err != nil {
		return 0, fmt.Errorf("parsing '%s': %w", keyLength, err)
	}

	if length < 0 {
		return 0, fmt.Errorf("invalid '%s': must be non-negative, got %d", keyLength, length)
	}

	return length, nil
}

func parseFilePath(root bencode.Dictionary) ([]bencode.ByteString, error) {
	raw, exists := root[keyPath]
	if !exists {
		return nil, fmt.Errorf("'%s' key not found", keyPath)
	}

	paths, err := bencode.AsList(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing '%s': %w", keyPath, err)
	}

	result, err := bencode.ConvertListToByteStrings(paths)
	if err != nil {
		return nil, fmt.Errorf("parsing file list: %w", err)
	}

	return result, nil
}

// TODO: test somehow
// do not modify 'infoDict' before encoding because info_hash depends on exact byte structure
func createInfoHash(root bencode.Dictionary) ([20]byte, error) {
	raw, exists := root[keyInfo]
	if !exists {
		return [20]byte{}, fmt.Errorf("'%s' key not found", keyInfo)
	}

	infoDict, err := bencode.AsDictionary(raw)
	if err != nil {
		return [20]byte{}, fmt.Errorf("'%s' is not a dictionary: %w", keyInfo, err)
	}

	encoded, err := bencode.Encode(infoDict)
	if err != nil {
		return [20]byte{}, fmt.Errorf("encoding '%s': %w", keyInfo, err)
	}

	return sha1.Sum(encoded), nil
}

// Reference: https://bittorrent.org/beps/bep_0012.html
func (t *MetaInfo) parseAnnounceList(root bencode.Dictionary) {
	raw, exists := root[keyAnnounceList]
	if !exists {
		fmt.Printf("'%s' key not found\n", keyAnnounceList) // TODO: change to log or remove
		return
	}

	rawList, err := bencode.AsList(raw)
	if err != nil {
		fmt.Printf("parsing '%s': %+v\n", keyAnnounceList, err) // TODO: change to log or remove
		return
	}

	var announceList [][]bencode.ByteString

	for tierIdx, tierRaw := range rawList {
		tier, err := bencode.AsList(tierRaw)
		if err != nil {
			fmt.Printf("tier %d: %+v\n", tierIdx, err)
			continue
		}

		var urls []bencode.ByteString

		for urlIdx, urlRaw := range tier {
			url, err := bencode.AsByteString(urlRaw)
			if err != nil {
				fmt.Printf("tier %d, url %d: %+v\n", tierIdx, urlIdx, err)
				continue
			}
			urls = append(urls, url)
		}

		if len(urls) > 0 {
			announceList = append(announceList, urls)
		}
	}

	t.AnnounceList = announceList
}

// TODO: add conversion function to display human-readable date
func (t *MetaInfo) parseCreationDate(root bencode.Dictionary) {
	raw, exists := root[keyCreationDate]
	if !exists {
		fmt.Printf("'%s' not found\n", keyCreationDate) // TODO: change to log or remove
		return
	}

	creationDate, err := bencode.AsInteger(raw)
	if err != nil {
		fmt.Printf("parsing '%s': %+v\n", keyCreationDate, err) // TODO: change to log or remove
		return
	}

	t.CreationDate = creationDate
}

func (t *MetaInfo) parseComment(root bencode.Dictionary) {
	raw, exists := root[keyComment]
	if !exists {
		fmt.Printf("'%s' not found\n", keyComment) // TODO: change to log or remove
		return
	}

	comment, err := bencode.AsByteString(raw)
	if err != nil {
		fmt.Printf("parsing '%s': %+v\n", keyComment, err) // TODO: change to log or remove
		return
	}

	t.Comment = comment
}

func (t *MetaInfo) parseCreatedBy(root bencode.Dictionary) {
	raw, exists := root[keyCreatedBy]
	if !exists {
		fmt.Printf("'%s' not found\n", keyCreatedBy) // TODO: change to log or remove
		return
	}

	createdBy, err := bencode.AsByteString(raw)
	if err != nil {
		fmt.Printf("parsing '%s': %+v\n", keyCreatedBy, err) // TODO: change to log or remove
		return
	}

	t.CreatedBy = createdBy
}

func (t *MetaInfo) parseEncoding(root bencode.Dictionary) {
	raw, exists := root[keyEncoding]
	if !exists {
		fmt.Printf("'%s' not found\n", keyEncoding) // TODO: change to log or remove
		return
	}

	encoding, err := bencode.AsByteString(raw)
	if err != nil {
		fmt.Printf("parsing '%s': %+v\n", keyEncoding, err) // TODO: change to log or remove
		return
	}

	t.Encoding = encoding
}
