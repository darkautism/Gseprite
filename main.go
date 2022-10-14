package gseprite

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"io"
	"log"
	"os"
	"unsafe"
)

func readString(file io.Reader) string {
	var Size uint16
	if _, err := file.Read((*[2]byte)(unsafe.Pointer(&Size))[:]); err != nil {
		log.Println(err)
	}
	buffer := make([]byte, Size)
	if _, err := file.Read(buffer); err != nil {
		log.Println(err)
	}
	return string(buffer)
}

type Header struct {
	FileSize      uint32
	Magic         uint16
	Frames        uint16
	Width         uint16
	Height        uint16
	ColorDepth    uint16
	Flags         uint32
	Speed         uint16 // DEPRECATED
	reserved1     uint32
	reserved2     uint32
	Palette       byte
	ignore        [3]byte
	NumberOfColor uint16
	PixelWidth    byte
	PixelHeight   byte
	reserved3     [92]byte
}

type Frame struct {
	Size           uint32
	Magic          uint16
	NumberOfChunk  uint16
	Duration       uint16
	reserved       [2]byte
	NumberOfChunk2 uint32

	// NonBytesFields
	Chunks   []Chunk
	Gseprite *Gseprite
}

func readFrame(file io.Reader, g *Gseprite) (*Frame, error) {
	var f Frame
	f.Gseprite = g
	file.Read((*[16]byte)(unsafe.Pointer(&f))[:])
	if f.Magic != 0xF1FA {
		return nil, errors.New("Magic code of frame checked failed")
	}
	var maxChunk int
	if f.NumberOfChunk == 0xFFFF {
		maxChunk = int(f.NumberOfChunk2)
	} else {
		maxChunk = int(f.NumberOfChunk)
	}

	for i := 0; i < maxChunk; i++ {
		f.Chunks = append(f.Chunks, readChunk(file, g))
	}

	return &f, nil
}

type ChunkType uint16

const (
	ChunkTypeOldPalette4  ChunkType = 0x0004
	ChunkTypeOldPalette11 ChunkType = 0x0011
	ChunkTypeLayer        ChunkType = 0x2004
	ChunkTypeCel          ChunkType = 0x2005
	ChunkTypeCelExtra     ChunkType = 0x2006
	ChunkTypeColorProfile ChunkType = 0x2007
	ChunkTypeMask         ChunkType = 0x2016 // DEPRECATED
	ChunkTypePath         ChunkType = 0x2017 // Never used.
	ChunkTypeFrameTags    ChunkType = 0x2018
	ChunkTypePalette      ChunkType = 0x2019
	ChunkTypeUserData     ChunkType = 0x2020
	ChunkTypeSlice        ChunkType = 0x2022
)

func (e ChunkType) String() string {
	switch e {
	case ChunkTypeOldPalette4:
		return "OldPalette4"
	case ChunkTypeOldPalette11:
		return "OldPalette11"
	case ChunkTypeLayer:
		return "Layer"
	case ChunkTypeCel:
		return "Cel"
	case ChunkTypeCelExtra:
		return "CelExtra"
	case ChunkTypeColorProfile:
		return "ColorProfile"
	case ChunkTypeMask:
		return "Mask"
	case ChunkTypePath:
		return "Path"
	case ChunkTypeFrameTags:
		return "FrameTags"
	case ChunkTypePalette:
		return "Palette"
	case ChunkTypeUserData:
		return "UserData"
	case ChunkTypeSlice:
		return "Slice"
	default:
		return fmt.Sprintf("%d", int(e))
	}
}

type Chunk interface {
	ChunkType() ChunkType
}

type chunk struct {
	Size uint32
	Type ChunkType
	Data []byte
}

func (c chunk) ChunkType() ChunkType {
	return c.Type
}

func readChunk(file io.Reader, g *Gseprite) Chunk {
	var c chunk
	file.Read((*[6]byte)(unsafe.Pointer(&c))[:])
	c.Data = make([]byte, c.Size-6)
	file.Read(c.Data)
	switch c.Type {
	case ChunkTypePalette:
		return readPalette(bytes.NewReader(c.Data), g)
	case ChunkTypeLayer:
		return readLayer(bytes.NewReader(c.Data), g)
	case ChunkTypeCel:
		return readCel(bytes.NewReader(c.Data), g)
	}
	return c
}

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
	for pi := 0; pi < int(p.Size); pi++ {
		var hasName uint8
		var c NamedColor
		file.Read((*[1]byte)(unsafe.Pointer(&hasName))[:])
		file.Read((*[4]byte)(unsafe.Pointer(&c))[:])
		if int(hasName) == 1 {
			c.Name = readString(file)
			pi += len(c.Name) + 4
		}

		p.Colors = append(p.Colors, c)
	}
	return &p
}

type NamedColor struct {
	R    int8
	G    int8
	B    int8
	A    int8
	Name string
}

func (g NamedColor) RGBA() (uint32, uint32, uint32, uint32) {
	return uint32(g.R), uint32(g.G), uint32(g.B), uint32(g.A)
}

type Gseprite struct {
	Header  Header
	Frames  []*Frame
	Layers  []*Layer
	Palette *Palette

	// For Sprites render
	curtime  float64
	curframe int
}

// Render current image, param is Duration, should be 1000/FPS
func (g *Gseprite) SpritesRender(Duration float64) image.Image {
	g.curtime += Duration
	if g.curtime > float64(g.Frames[g.curframe].Duration) {
		g.curtime -= float64(g.Frames[g.curframe].Duration)
		g.curframe++
		g.curframe %= len(g.Frames)
	}

	return g.Frames[g.curframe].Render()
}

// Get this Aseprite image file Rectangle
func (g Gseprite) Rect() image.Rectangle {
	return image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: int(g.Header.Width), Y: int(g.Header.Height)},
	}
}

// Render frame to single image
func (f *Frame) Render() image.Image {
	rect := f.Gseprite.Rect()
	ret := image.NewNRGBA(rect)
	for _, c := range f.Chunks {
		if c.ChunkType() == ChunkTypeCel {
			var tmp interface{}
			tmp = c
			cel := tmp.(*Cel)
			layer := f.Gseprite.Layers[cel.LayerIndex]
			if layer.Flags&LayerFlagsVisible == 0 {
				continue
			} // This layer is unvisible, skip it

			draw.Draw(ret, rect, cel, image.Point{X: int(-cel.X), Y: int(-cel.Y)}, draw.Over)
		}
	}
	return ret
}

// Create GIF file
func (g Gseprite) GIF() gif.GIF {
	var ret gif.GIF
	for _, frame := range g.Frames {
		img := frame.Render()
		palettedImage := image.NewPaletted(g.Rect(), palette.Plan9)
		draw.Draw(palettedImage, palettedImage.Rect, img, g.Rect().Min, draw.Over)
		ret.Image = append(ret.Image, palettedImage)
		ret.Delay = append(ret.Delay, int(frame.Duration)/10)
	}
	return ret
}

type LayerFlags uint16

const (
	LayerFlagsVisible          LayerFlags = 1
	LayerFlagsEditable         LayerFlags = 2
	LayerFlagsLockMovement     LayerFlags = 4
	LayerFlagsBackground       LayerFlags = 8
	LayerFlagsPreferLinkedCels LayerFlags = 16
	LayerFlagsCollapsedGroup   LayerFlags = 32
	LayerFlagsReference        LayerFlags = 64
)

type Layer struct {
	Flags      LayerFlags
	Type       uint16
	ChildLevel uint16
	width      uint16 //Ignore
	height     uint16 //Ignore
	Blend      uint16 // always 0
	Opacity    uint8
	reserved   [3]byte
	Name       string
}

func (p *Layer) ChunkType() ChunkType {
	return ChunkTypeLayer
}

func readLayer(file io.Reader, g *Gseprite) *Layer {
	var p Layer
	file.Read((*[16]byte)(unsafe.Pointer(&p))[:])
	p.Name = readString(file)

	return &p
}

// Load Aseprite file
func LoadAseprite(filename string) (*Gseprite, error) {
	var g Gseprite
	if f, err := os.Open(filename); err != nil {
		return nil, err
	} else {
		defer f.Close()
		f.Read((*[128]byte)(unsafe.Pointer(&g.Header))[:])
		if g.Header.Magic != 0xA5E0 {
			return nil, errors.New("Magic code check failed")
		}
		for i := 0; i < int(g.Header.Frames); i++ {
			if frame, err := readFrame(f, &g); err != nil {
				return nil, err
			} else {
				g.Frames = append(g.Frames, frame)
			}
		}
		for _, chunk := range g.Frames[0].Chunks {
			var tmp interface{}
			switch chunk.ChunkType() {
			case ChunkTypePalette:
				tmp = chunk
				g.Palette = tmp.(*Palette)
			case ChunkTypeLayer:
				tmp = chunk
				g.Layers = append(g.Layers, tmp.(*Layer))
			}
		}

	}

	return &g, nil
}

type CelType uint16

const (
	CelTypeRaw        CelType = 0
	CelTypeLinked     CelType = 1
	CelTypeCompressed CelType = 2
)

// Cel determine where to put a cel in the specified layer/frame
type Cel struct {
	LayerIndex uint16
	X          int16
	Y          int16
	Opacity    byte
	Type       CelType
	reserved   [7]byte
	Image      image.Image

	// Non structlize field
	ColorDepth uint16
}

func (c *Cel) ChunkType() ChunkType {
	return ChunkTypeCel
}

func (c *Cel) ColorModel() color.Model {
	switch c.ColorDepth {
	case 8:
		return color.AlphaModel
	case 16:
		return color.GrayModel
	case 32:
		return color.NRGBAModel
	}
	return color.NRGBAModel
}

func (c *Cel) Bounds() image.Rectangle {
	return c.Image.Bounds()
}

func (c *Cel) At(x, y int) color.Color {
	return c.Image.At(x, y)
}

func readCel(file io.Reader, g *Gseprite) *Cel {
	var p Cel
	file.Read((*[16]byte)(unsafe.Pointer(&p))[:])
	p.ColorDepth = g.Header.ColorDepth
	switch p.Type {
	case CelTypeRaw, CelTypeCompressed:
		var Height, Width uint16
		file.Read((*[2]byte)(unsafe.Pointer(&Width))[:])
		file.Read((*[2]byte)(unsafe.Pointer(&Height))[:])
		img := image.NewRGBA(image.Rect(0, 0, int(Height), int(Width)))
		p.Image = img
		switch g.Header.ColorDepth {
		case 8:
			img.Pix = make([]byte, Height*Width*4)
			buffer := make([]byte, Height*Width)
			r, _ := zlib.NewReader(file)
			r.Read(buffer)
			r.Close()
			img.Stride = int(Width) * 4
			img.Rect = image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: int(Width), Y: int(Height)},
			}
			for i := 0; i < int(Height*Width); i++ {
				img.Pix[i*4+3] = buffer[i]
			}
		case 16:
			img.Pix = make([]byte, Height*Width*4)
			buffer := make([]byte, Height*Width*2)
			r, _ := zlib.NewReader(file)
			r.Read(buffer)
			r.Close()
			img.Stride = int(Width) * 4
			img.Rect = image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: int(Width), Y: int(Height)},
			}
			for i := 0; i < int(Height*Width)*2; i += 2 {
				img.Pix[i*4] = buffer[i*2]
				img.Pix[i*4+1] = buffer[i*2]
				img.Pix[i*4+2] = buffer[i*2]
				img.Pix[i*4+3] = buffer[i*2+1]
			}
		case 32:
			img.Pix = make([]byte, Height*Width*4)
			r, _ := zlib.NewReader(file)
			r.Read(img.Pix)
			r.Close()
			img.Stride = int(Width) * 4
			img.Rect = image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: int(Width), Y: int(Height)},
			}
		}
	case CelTypeLinked:
	}

	return &p
}
