package blob

type Display struct {
	X, Y, Z  int
	Dir      int
	CubeType int // 1=8x8x8, 2=8x16x8, 3=16x16x16, 4=8x32x8
}

func (this Display) Size() (length, height, width int) {
	switch this.CubeType {
	case 2:
		return 8, 16, 8
	case 3:
		return 16, 16, 16
	case 4:
		return 8, 32, 8
	case 1:
		fallthrough
	default:
		return 8, 8, 8
	}
}
