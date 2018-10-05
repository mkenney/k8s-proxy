package iterator

/*
Iterator defines a simple interator interface.
*/
type Iterator interface {
	// Cur reads the key and value at the current cursor postion into pK and
	// pV respectively. Cur will return false if no iteration has begun,
	// including following calls to Reset.
	Cur(pK, pV *interface{}) bool

	// Next moves the cursor forward one position before reading the key and
	// value at the cursor position into pK and pV respectively. If data is
	// available at that position and was written to pK and pV then Next
	// returns true, else false to signify the end of the data and resets
	// the cursor postion to the beginning of the data set (-1).
	Next(pK, pV *interface{}) bool

	// Prev moves the cursor backward one position before reading the key
	// and value at the cursor position into pK and pV respectively. If data
	// is available at that position and was written to pK and pV then Prev
	// returns true, else false to signify the beginning of the data.
	Prev(pK, pV *interface{}) bool

	// Reset sets the iterator position to the beginning of the data set.
	Reset()

	// Seek sets the iterator cursor position to the location of key. key is
	// expected to be the array or map index of the desired location.
	Seek(key interface{}) error
}
