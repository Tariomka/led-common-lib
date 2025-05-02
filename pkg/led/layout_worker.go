package led

import "iter"

type LayoutWorker interface {
	LedSingleWorker
	LedRowIndividualWorker
	LedRowWorker
	LedLayerWorker
	LedBlockWorker
	Slicer
}

type LedSingleWorker interface {
	ChangeSingle(x, y, z uint8, c Color) error
	SetSingle(x, y, z uint8, c Color) error
	ResetSingle(x, y, z uint8) error
}

type LedRowWorker interface {
	ChangeRow(y, z uint8, c Color) error
	SetRow(y, z uint8, c Color) error
	ResetRow(y, z uint8) error
}

type LedRowIndividualWorker interface {
	ChangeRowIndividual(y, z uint8, c Color, values byte) error
	SetRowIndividual(y, z uint8, c Color, values byte) error
	ResetRowIndividual(y, z uint8, values byte) error
}

type LedLayerWorker interface {
	ChangeLayer(z uint8, c Color) error
	SetLayer(z uint8, c Color) error
	ResetLayer(z uint8) error
}

type LedBlockWorker interface {
	ChangeBlock(c Color)
	SetBlock(c Color)
	ResetBlock()
}

type Slicer interface {
	IterateSlices() iter.Seq2[uint8, []byte]
}

type Frame func(LayoutWorker) // Single frame of a light show
type LightShow []Frame        // Collection of light show frames
