package model

/*
Value represents a single data value. Used as an interface to values stored
in Model nodes.
*/
type Value interface {
	// Bool returns the boolean representation of the value of this node, or
	// an error if the type conversion is not possible.
	Bool() (bool, error)
	// Float returns the float64 representation of the value of this node,
	// or an error if the type conversion is not possible.
	Float() (float64, error)
	// Float32 returns the float32 representation of the value of this node,
	// or an error if the type conversion is not possible.
	Float32() (float32, error)
	// Float64 returns the float64 representation of the value of this node,
	// or an error if the type conversion is not possible.
	Float64() (float64, error)
	// Int returns the int representation of the value of this node, or an
	// error if the type conversion is not possible.
	Int() (int, error)
	// List returns the array of Values stored in this node, or an error if
	// the type conversion is not possible.
	List() ([]Value, error)
	// Map returns the map[string]Value data stored in this node, or an
	// error if the type conversion is not possible.
	Map() (map[string]Value, error)
	// Model returns the Model stored at this node, or an error if the value
	// does not implement Model.
	Model() (Model, error)
	// String returns the boolean representation of the value, or an error
	// if the type conversion is not possible.
	String() (string, error)
	// Value returns the untyped value.
	Value() interface{}
}
