package randb

// Chunk represents rows interval.
type Chunk struct {
	Offset int
	Limit  int
}

type Chunker interface {
	Chunk(int) []Chunk
}

type chunker int

// TODO: use real chunking for big values.
func (chunker) Chunk(count int) []Chunk{
	return []Chunk{{Offset: 0, Limit: count}}
}
