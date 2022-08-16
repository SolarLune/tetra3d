package tetra3d

// LightGroup represents a grouping of lights. This is used on Models to control which lights are used to light them.
type LightGroup struct {
	Lights []ILight // The collection of lights present in the LightGroup.
	Active bool     // If the LightGroup is active or not; when a Model's LightGroup is inactive, Models will fallback to the lights present under the Scene's root.
}

// NewLightGroup creates a new LightGroup.
func NewLightGroup(lights ...ILight) *LightGroup {
	lg := &LightGroup{
		Active: true,
	}
	lg.Add(lights...)
	return lg
}

// Add adds the passed ILights to the LightGroup.
func (lg *LightGroup) Add(lights ...ILight) {
	lg.Lights = append(lg.Lights, lights...)
}

// Clone clones the LightGroup.
func (lg *LightGroup) Clone() *LightGroup {
	newLG := NewLightGroup(lg.Lights...)
	newLG.Active = lg.Active
	return newLG
}
