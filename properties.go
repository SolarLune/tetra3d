package tetra3d

// Properties is an unordered set of property names to values, representing a means of identifying Nodes or carrying data on Nodes.
type Properties struct {
	props map[string]*Property
}

// NewProperties returns a new Properties object.
func NewProperties() *Properties {
	return &Properties{map[string]*Property{}}
}

func (props *Properties) Clone() *Properties {
	newTags := NewProperties()
	for k, v := range props.props {
		newTags.Get(k).Set(v.Value)
	}
	return newTags
}

// Clear clears the Properties object of all game properties.
func (props *Properties) Clear() {
	props.props = map[string]*Property{}
}

// Remove removes the tag specified from the Properties object.
func (props *Properties) Remove(tag string) {
	delete(props.props, tag)
}

// Has returns true if the Properties object has properties by all of the names specified, and false otherwise.
func (props *Properties) Has(propNames ...string) bool {
	for _, t := range propNames {
		if _, exists := props.props[t]; !exists {
			return false
		}
	}
	return true
}

// Get returns the value associated with the specified property name. If a property with the
// passed name (propName) doesn't exist, Get will return nil.
func (props *Properties) Get(propName string) *Property {
	if _, ok := props.props[propName]; !ok {
		props.props[propName] = &Property{}
	}
	return props.props[propName]
}

// Property represents a game property on a Node or other resource.
type Property struct {
	Value interface{}
}

// Set sets the property's value to the given value.
func (prop *Property) Set(value interface{}) {
	prop.Value = value
}

// IsBool returns true if the Property is a boolean value.
func (prop *Property) IsBool() bool {
	if _, ok := prop.Value.(bool); ok {
		return true
	}
	return false
}

// AsBool returns the value associated with the Property as a bool.
// Note that this does not sanity check to ensure the Property is a bool first.
func (prop *Property) AsBool() bool {
	return prop.Value.(bool)
}

// IsString returns true if the Property is a string.
func (prop *Property) IsString() bool {
	if _, ok := prop.Value.(string); ok {
		return true
	}
	return false
}

// AsString returns the value associated with the Property as a string.
// Note that this does not sanity check to ensure the Property is a string first.
func (prop *Property) AsString() string {
	return prop.Value.(string)
}

// IsFloat returns true if the Property is a float64.
func (prop *Property) IsFloat64() bool {
	if _, ok := prop.Value.(float64); ok {
		return true
	}
	return false
}

// AsFloat returns the value associated with the Property as a float64.
// Note that this does not sanity check to ensure the Property is a float64 first.
func (prop *Property) AsFloat64() float64 {
	return prop.Value.(float64)
}

// IsInt returns true if the Property is an int.
func (prop *Property) IsInt() bool {
	if _, ok := prop.Value.(int); ok {
		return true
	}
	return false
}

// AsInt returns the value associated with the Property as a float.
// Note that this does not sanity check to ensure the Property is an int first.
func (prop *Property) AsInt() int {
	return prop.Value.(int)
}

// IsColor returns true if the Property is a color.
func (prop *Property) IsColor() bool {
	if _, ok := prop.Value.(Color); ok {
		return true
	}
	return false
}

// AsColor returns the value associated with the Property as a Color clone.
// Note that this does not sanity check to ensure the Property is a Color first.
func (prop *Property) AsColor() Color {
	return prop.Value.(Color)
}

// IsVector returns true if the Property is a vector.
func (prop *Property) IsVector() bool {
	if _, ok := prop.Value.(Vector); ok {
		return true
	}
	return false
}

// AsVector returns the value associated with the Property as a 3D position Vector clone.
// The axes are corrected to account for the difference between Blender's axis order and Tetra3D's (i.e.
// Blender's +X, +Y, +Z becomes Tetra3D's +X, +Z, +Y).
// Note that this does not sanity check to ensure the Property is a vector first.
func (prop *Property) AsVector() Vector {
	return prop.Value.(Vector)
}
