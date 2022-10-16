package gseprite

import (
	"io"
	"unsafe"
)

type FrameTags struct {
	Counts   uint16
	reserved [8]byte

	Tags []Tag
	M    map[string]Tag
}

func (p *FrameTags) ChunkType() ChunkType {
	return ChunkTypeFrameTags
}

type Tag struct {
	From     uint16
	To       uint16
	Loop     byte
	reserved [8]byte
	RGB      [3]byte // Deprecated
	Extra    byte
	Name     string
}

func readFrameTags(file io.Reader, g *Gseprite) *FrameTags {
	var p FrameTags
	file.Read((*[10]byte)(unsafe.Pointer(&p))[:])
	p.M = make(map[string]Tag)
	for i := 0; i < int(p.Counts); i++ {
		var tag Tag
		file.Read((*[17]byte)(unsafe.Pointer(&tag))[:])

		tag.Name = readString(&file)
		p.M[tag.Name] = tag
		p.Tags = append(p.Tags, tag)
	}
	g.FrameTags = &p
	return &p
}
