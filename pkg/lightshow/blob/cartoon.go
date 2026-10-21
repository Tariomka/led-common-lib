package blob

// Cartoon types
const (
	Normal = 0
	Script = 1
	Audio  = 2
)

// Prop sizes per element kind, including the start/end frame header
var propSizes = [3]int{56, 20, 56}

// Cartoon is one show of a group.
type Cartoon struct {
	Type int
	Body string
}

func (this Cartoon) Header() FrameHeader {
	if len(this.Body) < 8 {
		return FrameHeader{}
	}

	return FrameHeader{
		FrameCount: int(int32(U32(this.Body, 0))),
		Loop:       I16(this.Body, 4),
		Delay:      int(uint16(this.Body[6]) | uint16(this.Body[7])<<8),
	}
}

// ForEachProp visits every property of every element of a Normal/Audio cartoon.
func (this Cartoon) ForEachProp(callback func(Prop)) {
	for element := 8; element+8 <= len(this.Body); {
		size := int(U32(this.Body, element))
		kind := int(U32(this.Body, element+4))
		if size < 8 || element+size > len(this.Body) {
			return
		}

		if kind >= 0 && kind < len(propSizes) {
			propSize := propSizes[kind]
			for position := element + 8; position+propSize <= element+size; position += propSize {
				callback(Prop{
					Kind:    kind,
					Element: element,
					Start:   I16(this.Body, position), End: I16(this.Body, position+2),
					Data: this.Body[position+4 : position+propSize]})
			}
		}
		element += size
	}
}
