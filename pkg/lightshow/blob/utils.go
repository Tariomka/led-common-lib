package blob

func U32(data string, offset int) uint32 {
	return uint32(data[offset]) |
		uint32(data[offset+1])<<8 |
		uint32(data[offset+2])<<16 |
		uint32(data[offset+3])<<24
}

func I16(data string, offset int) int {
	return int(int16(uint16(data[offset]) | uint16(data[offset+1])<<8))
}

// Name returns a NUL-padded 16-byte name field without allocating.
func Name(data string, offset int) string {
	end := offset
	for end < offset+nameSize && data[end] != 0 {
		end++
	}
	return data[offset:end]
}

// IndexValue reads element i of an index table as the given type
// (0=u32, 1=i16, 2=u16, 3=i8, 4=u8), like the script GetIndex function.
func IndexValue(index string, i, typ int) int {
	if i < 0 {
		return 0
	}

	switch typ {
	case 0:
		if i < len(index)/4 {
			return int(int32(U32(index, 4*i)))
		}
	case 1:
		if i < len(index)/2 {
			return I16(index, 2*i)
		}
	case 2:
		if i < len(index)/2 {
			return int(uint16(index[2*i]) | uint16(index[2*i+1])<<8)
		}
	case 3:
		if i < len(index) {
			return int(int8(index[i]))
		}
	case 4:
		if i < len(index) {
			return int(index[i])
		}
	}

	return 0
}
