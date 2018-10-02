package sorter

// SortFlag provides a type for sort flags
type SortFlag int

const (
	// SortByKey - sort hash data by key
	SortByKey SortFlag = iota
)

// Sorter describes a sorter.
type Sorter interface {
	// Reverse reverses the order of the data set.
	Reverse(SortFlag) error
	// Sort sorts the model data.
	Sort(SortFlag) error
	// Len returns the number of items stored in this model.
	Len() int
}
