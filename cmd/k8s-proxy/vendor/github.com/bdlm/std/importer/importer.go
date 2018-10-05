package importer

/*
Importer is the interface that wraps the basic Import method.

Importer accepts the empty interface and extracts data from there, returning
an error if the import fails.

Implementations must not retain data. Implementations should not retain any
imported data if returning an error.
*/
type Importer interface {
	// Import imports the given data into the reciever's data structure.
	Import(data interface{}) error
}
