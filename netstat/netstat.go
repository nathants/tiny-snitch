package netstat

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/nathants/tinysnitch/lib"
	"github.com/nathants/tinysnitch/log"
	"net"
	"os"
	"regexp"
	"strconv"
)

type Entry struct {
	Proto   string
	SrcIP   net.IP
	SrcPort uint
	DstIP   net.IP
	DstPort uint
	UserId  int
	INode   int
}

func NewEntry(proto string, srcIP net.IP, srcPort uint, dstIP net.IP, dstPort uint, userId int, iNode int) Entry {
	return Entry{
		Proto:   proto,
		SrcIP:   srcIP,
		SrcPort: srcPort,
		DstIP:   dstIP,
		DstPort: dstPort,
		UserId:  userId,
		INode:   iNode,
	}
}

func FindEntry(proto string, srcIP net.IP, srcPort uint, dstIP net.IP, dstPort uint) *Entry {
	entries, err := Parse(proto)
	if err != nil {
		log.Warning("Error while searching for %s netstat entry: %s", proto, err)
		return nil
	}
	for _, entry := range entries {
		if srcIP.Equal(entry.SrcIP) && srcPort == entry.SrcPort && dstIP.Equal(entry.DstIP) && dstPort == entry.DstPort {
			return &entry
		}
	}
	return nil
}

var (
	parser = regexp.MustCompile(`(?i)` +
		`\d+:\s+` + // sl
		`([a-f0-9]{8,32}):([a-f0-9]{4})\s+` + // local_address
		`([a-f0-9]{8,32}):([a-f0-9]{4})\s+` + // rem_address
		`[a-f0-9]{2}\s+` + // st
		`[a-f0-9]{8}:[a-f0-9]{8}\s+` + // tx_queue rx_queue
		`[a-f0-9]{2}:[a-f0-9]{8}\s+` + // tr tm->when
		`[a-f0-9]{8}\s+` + // retrnsmt
		`(\d+)\s+` + // uid
		`\d+\s+` + // timeout
		`(\d+)\s+` + // inode
		`.+`) // stuff we don't care about
)

func decToInt(n string) int {
	d, err := strconv.ParseInt(n, 10, 64)
	if err != nil {
		log.Fatal("Error while parsing %s to int: %s", n, err)
	}
	return int(d)
}

func hexToInt(h string) uint {
	d, err := strconv.ParseUint(h, 16, 64)
	if err != nil {
		log.Fatal("Error while parsing %s to int: %s", h, err)
	}
	return uint(d)
}

func hexToInt2(h string) (uint, uint) {
	if len(h) > 16 {
		d, err := strconv.ParseUint(h[:16], 16, 64)
		if err != nil {
			log.Fatal("Error while parsing %s to int: %s", h[16:], err)
		}
		d2, err := strconv.ParseUint(h[16:], 16, 64)
		if err != nil {
			log.Fatal("Error while parsing %s to int: %s", h[16:], err)
		}
		return uint(d), uint(d2)
	} else {
		d, err := strconv.ParseUint(h, 16, 64)
		if err != nil {
			log.Fatal("Error while parsing %s to int: %s", h[16:], err)
		}
		return uint(d), 0
	}
}

func hexToIP(h string) net.IP {
	n, m := hexToInt2(h)
	var ip net.IP
	if m != 0 {
		ip = make(net.IP, 16)
		// TODO: Check if this depends on machine endianness?
		binary.LittleEndian.PutUint32(ip, uint32(n>>32))
		binary.LittleEndian.PutUint32(ip[4:], uint32(n))
		binary.LittleEndian.PutUint32(ip[8:], uint32(m>>32))
		binary.LittleEndian.PutUint32(ip[12:], uint32(m))
	} else {
		ip = make(net.IP, 4)
		binary.LittleEndian.PutUint32(ip, uint32(n))
	}
	return ip
}

func Parse(proto string) ([]Entry, error) {
	filename := fmt.Sprintf("/proc/net/%s", proto)
	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	entries := make([]Entry, 0)
	scanner := bufio.NewScanner(fd)
	for lineno := 0; scanner.Scan(); lineno++ {
		// skip column names
		if lineno == 0 {
			continue
		}
		line := lib.Trim(scanner.Text())
		m := parser.FindStringSubmatch(line)
		if m == nil {
			log.Warning("Could not parse netstat line from %s: %s", filename, line)
			continue
		}
		entries = append(entries, NewEntry(
			proto,
			hexToIP(m[1]),
			hexToInt(m[2]),
			hexToIP(m[3]),
			hexToInt(m[4]),
			decToInt(m[5]),
			decToInt(m[6]),
		))
	}
	return entries, nil
}
