package model

/*
ModelType defines a type for use by models.
*/
type ModelType int

const (
	// ModelTypeList causes the model to behave as a list (keys are unsigned,
	// contiguous integers beginning at 0).
	ModelTypeList ModelType = iota
	// ModelTypeHash causes the model to behave as a hash (keys are strings,
	// order is static).
	ModelTypeHash
)

/*
Model is a list or a map of Values.

This interface defines data storage and access methods for data models in
order to provide a consistent interface for communicating messages between
instances. This allows several abstractions on and recursions into
multidimensional untyped data structures.
*/
type Model interface {
	// Delete removes a value from this model.
	Delete(key interface{}) error
	// Filter filters elements of the data using a callback function and
	// returns the result.
	Filter(callback func(Value) Model) Model
	// Get returns the specified data value in this model.
	Get(key interface{}) (Value, error)
	// GetID returns returns this model's id.
	GetID() interface{}
	// GetType returns the model type.
	GetType() ModelType
	// Has tests to see of a specified data element exists in this model.
	Has(key interface{}) bool
	// Lock marks this model as read-only. There is no Unlock.
	Lock()
	// Map applies a callback to all elements in this model and returns the
	// result.
	Map(callback func(Value) Model) Model
	// Merge merges data from any Model into this Model.
	Merge(Model) error
	// Push a value to the end of the internal data store.
	Push(value interface{}) error
	// Reduce iteratively reduces the data set to a single value using a
	// callback function and returns the result.
	Reduce(callback func(Value) bool) Value
	// Set stores a value in the internal data store. All values must be
	// identified by key.
	Set(key interface{}, value interface{}) error
	// SetData replaces the current data stored in the model with the
	// provided data.
	SetData(data interface{}) error
	// SetID sets this Model's identifier property.
	SetID(id interface{})
	// SetType sets the model type. If any data is stored in this model,
	// this property becomes read-only.
	SetType(typ ModelType) error
}
