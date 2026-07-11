package tetra3d

import "strings"

// Properties is an ordered set of property names to values, representing a means of identifying or carrying data on resources.
type Properties struct {
	data []*Property
}

// NewProperties returns a new Properties object.
func NewProperties() *Properties {
	return &Properties{}
}

// Clone clones the Properties object.
func (props *Properties) Clone() *Properties {
	newProps := NewProperties()
	for _, v := range props.data {
		newProps.Add(v.name).Set(v.Value)
	}
	return newProps
}

// Clear clears the Properties object of all game properties.
func (props *Properties) Clear() {
	for i := range props.data {
		props.data[i] = nil
	}
	props.data = props.data[:0]
}

// CopyFrom copies the properties from the source Properties set (src).
// It does not clear the properties set on props.
// If overwrite is true, then all properties on src will be copied over to the destination calling Properties set.
// If overwrite is false, then all properties on src that have not been already set on the destination calling Properties set will be copied over.
func (props *Properties) CopyFrom(src *Properties, overwrite bool) {
	for _, prop := range src.data {
		if overwrite || !props.Has(prop.name) {
			props.Add(prop.name).Set(prop.Value)
		}
	}
}

// Remove removes the tag specified from the Properties object.
func (props *Properties) Remove(key string) {
	for i, p := range props.data {
		if p.name == key {
			props.data[i] = nil
			props.data = append(props.data[:i], props.data[i+1:]...)
		}
	}
}

// Has returns true if the Properties object has a property by the name specified.
func (props *Properties) Has(propName string) bool {
	return props.GetByName(propName) != nil
}

// Add adds a property to the Properties map using the given name.
// If a property of the specified name already exists, it will return that property instead.
func (props *Properties) Add(propName string) *Property {
	for _, p := range props.data {
		if p.name == propName {
			return p
		}
	}
	newProp := &Property{
		name: propName,
	}
	props.data = append(props.data, newProp)
	return newProp
}

// Returns the value associated with the specified property name. If a property with the
// passed name (propName) doesn't exist, the function will return nil.
func (props *Properties) GetByName(propName string) *Property {
	for _, p := range props.data {
		if p.name == propName {
			return p
		}
	}
	return nil
}

// Returns the value associated with the property of the given index.
// If the property doesn't exist, the function will return nil.
func (props *Properties) Get(propIndex int) *Property {
	if propIndex >= 0 && propIndex < len(props.data) {
		return props.data[propIndex]
	}
	return nil
}

// Returns if the Properties object has a property of the given name with the given value.
func (props *Properties) HasNameAndValue(propName string, value any) bool {
	for _, p := range props.data {
		if p.name == propName {
			return p.name == propName && p.Value == value
		}
	}
	return false
}

// Returns if the Properties object has a property of the given name with the given value.
func (props *Properties) HasNameAndBitfieldValue(propName string, value Bitfield) bool {
	for _, p := range props.data {
		if p.name == propName {
			return p.name == propName && p.IsBitfield() && p.AsBitfield().Contains(value)
		}
	}
	return false
}

// Returns if the Properties object has a property of the given name with the given value.
func (props *Properties) HasBitfieldValue(value Bitfield) bool {
	for _, p := range props.data {
		return p.IsBitfield() && p.AsBitfield().Contains(value)
	}
	return false
}

// Iterates over each Property in the Properties set, in sequence.
func (props *Properties) ForEach(forEach func(p *Property, index int) bool) {
	for i, p := range props.data {
		if !forEach(p, i) {
			return
		}
	}
}

// Set sets the given value to the property name, creating it if it doesn't exist.
func (props *Properties) Set(propName string, value any) *Property {
	prop := props.Add(propName)
	prop.Set(value)
	return prop
}

// Count returns the number of properties in the Properties set.
func (props *Properties) Count() int {
	return len(props.data)
}

// Property represents a game property on a Node or other resource.
type Property struct {
	name  string
	Value any
}

func (prop *Property) Name() string {
	return prop.name
}

// Set sets the property's value to the given value.
func (prop *Property) Set(value any) {
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

// Returns the property as two strings - a scene name and the object name.
func (prop *Property) AsStringSceneObjectPair() (sceneName, objectName string) {
	results := strings.Split(prop.Value.(string), ":")
	if len(results) == 0 {
		return "", ""
	}
	if len(results) == 1 {
		return "", results[0]
	}
	return results[0], results[1]
}

// Returns if the Property contains a bitfield value.
func (prop *Property) IsBitfield() bool {
	if _, ok := prop.Value.(Bitfield); ok {
		return true
	}
	return false
}

// Returns the property as a Bitfield value.
func (prop *Property) AsBitfield() Bitfield {
	return prop.Value.(Bitfield)
}

// IsFloat returns true if the Property is a float32.
func (prop *Property) IsFloat32() bool {
	if _, ok := prop.Value.(float32); ok {
		return true
	}
	return false
}

// AsFloat returns the value associated with the Property as a float32.
// Note that this does not sanity check to ensure the Property is a float32 first.
func (prop *Property) AsFloat32() float32 {
	return prop.Value.(float32)
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
	if _, ok := prop.Value.(Color4); ok {
		return true
	}
	return false
}

// AsColor returns the value associated with the Property as a Color.
// Note that this does not sanity check to ensure the Property is a Color first.
func (prop *Property) AsColor() Color4 {
	return prop.Value.(Color4)
}

// IsVector3 returns true if the Property is a Vector3.
func (prop *Property) IsVector3() bool {
	if _, ok := prop.Value.(Vector3); ok {
		return true
	}
	return false
}

// AsVector3 returns the value associated with the Property as a 3D position Vector.
// The axes are corrected to account for the difference between Blender's axis order and Tetra3D's (i.e.
// Blender's +X, +Y, +Z becomes Tetra3D's +X, +Z, +Y).
// Note that this does not sanity check to ensure the Property is a Vector3 first.
func (prop *Property) AsVector3() Vector3 {
	return prop.Value.(Vector3)
}

// IsVector2 returns true if the Property is a Vector2.
func (prop *Property) IsVector2() bool {
	if _, ok := prop.Value.(Vector2); ok {
		return true
	}
	return false
}

// AsVector3 returns the value associated with the Property as a 2D Vector.
// Note that this does not sanity check to ensure the Property is a Vector2 first.
func (prop *Property) AsVector2() Vector2 {
	return prop.Value.(Vector2)
}
