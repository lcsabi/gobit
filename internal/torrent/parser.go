package torrent

// Reference: https://wiki.theory.org/BitTorrentSpecification#Metainfo_File_Structure
type TorrentFile struct {
	Info         InfoDict   // a dictionary that describes the file(s) of the torrent
	Announce     string     // tracker URL
	AnnounceList [][]string // offers backwards compatibility, optional
	CreationDate int64      // standard UNIX epoch format, optional
	Comment      string     //  free-form textual comments of the author, optional
	CreatedBy    string     // name and version of the program used to create the .torrent, optional
	Encoding     string     // the string encoding format used to generate the pieces part of the info dictionary in the .torrent metafile, optional
}

type InfoDict struct {
	PieceLength int64      // number of bytes in each piece
	Pieces      [][20]byte // concatenation of all 20-byte SHA1 hash values, one per piece
	Private     *int       // Optional: 1 means private tracker
	Files       []FileInfo // Always populated, even in single-file torrents
	Name        string     // Top-level directory or file name
}

type FileInfo struct {
	Length int64    // length of the file in bytes
	Path   []string // a list containing one or more string elements that together represent the path and filename
}

func (t TorrentFile) IsMultiFile() bool {
	return len(t.Info.Files) > 1
}
