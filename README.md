# Gobit

**Gobit** is a lightweight, modular BitTorrent client written in Go, designed for clarity, performance, and strict protocol compliance. It is built with the goals of educational value, practical portfolio demonstration, and potential extensibility into a fully-featured torrent client.

---

## 1. Features

### Implemented
- [x] Parse bencode:
    - [x] Decode bencoded strings
    - [x] Decode bencoded integers
    - [x] Decode bencoded lists
    - [x] Decode bencoded dicitonaries
- [x] Parse torrent file:
    - [x] Parse announce tracker
    - [x] Parse announce-list trackers for Multitracker Metadata Extension
    - [x] Parse info dictionary (name, piece length, pieces)
    - [x] Calculate info hash
    - [x] Parse creation date
    - [x] Parse comment
    - [x] Parse created by
    - [x] Parse encoding

### In Progress

#### Tracker Communication
- [ ] Implement HTTP tracker request (BEP 0003)
- [ ] Parse tracker response (`peers` list in binary or dictionary format)
- [ ] Support UDP trackers (BEP 0015)

### Planned

#### Peer Protocol
- [ ] TCP connection handling to peers
- [ ] BitTorrent handshake exchange
- [ ] Implement basic peer messages:
  - [ ] `choke` / `unchoke`
  - [ ] `interested` / `not interested`
  - [ ] `have`, `bitfield`
  - [ ] `request`, `piece`, `cancel`
- [ ] Maintain peer state (choked/interested, pieces owned, etc.)
- [ ] Request and download pieces from peers
- [ ] Assemble and verify pieces using SHA-1

#### Storage & Piece Management
- [ ] Store downloaded pieces to disk
- [ ] Validate piece hashes against `info` dictionary
- [ ] Resume partially downloaded torrents

#### Basic CLI
- [ ] Load `.torrent` file from command line
- [ ] Start/stop torrent download
- [ ] Show basic status (progress, speed, connected peers)

*Once MVP is stable, potential additions include:*
#### User Interfaces
- [ ] **TUI** (Terminal User Interface) for headless server control
- [ ] **GUI** (Graphical User Interface) for desktop users

#### Performance & Networking
- [ ] Optimistic unchoking & choking algorithms
- [ ] Piece selection strategies (rarest first, sequential)
- [ ] Peer exchange (BEP 0011)
- [ ] DHT (BEP 0005) for trackerless peer discovery
- [ ] Local peer discovery (BEP 0014)
- [ ] uTP transport (BEP 0029)
- [ ] Swarm health checking
- [ ] Torrent health checking

#### File Management
- [ ] Selective file downloading in multi-file torrents
- [ ] File priority settings
- [ ] Preallocation & sparse files

#### Security & Privacy
- [ ] Protocol encryption (BEP 0009)
- [ ] IP filtering
- [ ] Private torrent support enforcement

#### Quality of Life
- [ ] Bandwidth throttling
- [ ] Detailed session statistics
- [ ] Configurable settings file
- [ ] Magnet link support (BEP 0009)

---

## 2. Architecture

---

## 3. Usage


---

## 4. Specification References
[BitTorrent Specification Theory (unofficial)](https://wiki.theory.org/BitTorrentSpecification)
[BEP 0003: The BitTorrent Protocol Specification](https://bittorrent.org/beps/bep_0003.html)
[BEP 0012: Multitracker Metadata Extension](https://www.bittorrent.org/beps/bep_0012.html)
