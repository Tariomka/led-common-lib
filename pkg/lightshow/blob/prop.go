package blob

// Prop is one keyframed property of an element; Data starts after the start/end frame fields.
type Prop struct {
	Kind       int
	Element    int // offset of the owning element within the cartoon body
	Start, End int
	Data       string
}
