package gseprite

import (
	"image/color"
	"io"
	"unsafe"
)

type Palette struct {
	Size            uint32
	FirstColorIndex uint32
	LastColorIndex  uint32
	reserved        [8]byte
	Colors          []color.Color
}

func (p *Palette) ChunkType() ChunkType {
	return ChunkTypePalette
}

func readPalette(file io.Reader, g *Gseprite) *Palette {
	var p Palette
	file.Read((*[20]byte)(unsafe.Pointer(&p))[:])
	for pi := 0; pi < int(p.LastColorIndex)+1; pi++ {
		var hasName uint16
		var c NamedColor
		file.Read((*[2]byte)(unsafe.Pointer(&hasName))[:])
		file.Read((*[4]byte)(unsafe.Pointer(&c))[:])
		if int(hasName) == 1 {
			c.Name = readString(&file)
		}

		p.Colors = append(p.Colors, c)
	}
	g.Palette = &p
	return &p
}

type NamedColor struct {
	R    uint8
	G    uint8
	B    uint8
	A    uint8
	Name string
}

func (c NamedColor) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R)
	r |= r << 8
	g = uint32(c.G)
	g |= g << 8
	b = uint32(c.B)
	b |= b << 8
	a = uint32(c.A)
	a |= a << 8
	return
}

type OldPalette struct {
	PacketsLength uint16
	Colors        []color.Color
}

type OldPalettePackets struct {
	Skip         byte
	ColorsLength byte
	Colors       []color.Color
}

func (p *OldPalette) ChunkType() ChunkType {
	return ChunkTypeOldPalette4
}

func readOldPalette4(file io.Reader, g *Gseprite) *OldPalette {
	var p OldPalette
	file.Read((*[2]byte)(unsafe.Pointer(&p))[:])
	for i := 0; i < int(p.PacketsLength); i++ {
		var opp OldPalettePackets
		file.Read((*[2]byte)(unsafe.Pointer(&opp))[:])
		for j := 0; j < int(opp.ColorsLength); j++ {
			var nc NamedColor
			nc.A = 255
			file.Read((*[2]byte)(unsafe.Pointer(&nc))[:])
			p.Colors = append(p.Colors, nc)
		}
	}
	return &p
}
