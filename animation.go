package tetra3d

import (
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	TrackTypePosition = "Pos"
	TrackTypeScale    = "Sca"
	TrackTypeRotation = "Rot"

	InterpolationLinear = iota
	InterpolationConstant
	InterpolationCubic // Unimplemented
)

type Data struct {
	contents interface{}
}

func (data *Data) AsVector() Vector {
	return data.contents.(Vector)
}

func (data *Data) AsQuaternion() Quaternion {
	return data.contents.(Quaternion)
}

// Keyframe represents a single keyframe in an animation for an AnimationTrack.
type Keyframe struct {
	Time float64
	Data Data
}

func newKeyframe(time float64, data Data) *Keyframe {
	return &Keyframe{
		Time: time,
		Data: data,
	}
}

// AnimationTrack represents a collection of keyframes that drive an animation type (position, scale or rotation) for a node in an animation.
type AnimationTrack struct {
	Type          string
	Keyframes     []*Keyframe
	Interpolation int
}

// AddKeyframe adds a keyframe of the necessary data type to the AnimationTrack.
func (track *AnimationTrack) AddKeyframe(time float64, data interface{}) {
	track.Keyframes = append(track.Keyframes, newKeyframe(time, Data{data}))
}

// ValueAsVector returns a Vector associated with the current time in seconds, as well as a boolean indicating if
// the Vector exists (i.e. if the AnimationTrack has location or scale data in the animation). The Vector will be
// interpolated according to time between keyframes.
func (track *AnimationTrack) ValueAsVector(time float64) (Vector, bool) {

	if len(track.Keyframes) == 0 {
		return Vector{}, false
	}

	if first := track.Keyframes[0]; time <= first.Time {
		return first.Data.AsVector(), true
	} else if last := track.Keyframes[len(track.Keyframes)-1]; time >= last.Time {
		return last.Data.AsVector(), true
	}

	var first *Keyframe
	var last *Keyframe

	for _, k := range track.Keyframes {

		if k.Time < time {
			first = k
		} else {
			last = k
			break
		}

	}

	if time == first.Time {
		return first.Data.AsVector(), true
	} else if time == last.Time {
		return last.Data.AsVector(), true
	}

	fd := first.Data.AsVector()
	ld := last.Data.AsVector()

	// Linear interpolation
	t := (time - first.Time) / (last.Time - first.Time)

	if track.Interpolation == InterpolationConstant {
		return fd, true
	}

	// We still need to implement InterpolationCubic
	// if track.Type == TrackTypePosition || track.Type == TrackTypeScale {
	return fd.Add(ld.Sub(fd).Scale(t)), true
	// }

}

// ValueAsQuaternion returns the Quaternion associated with this AnimationTrack using the given time in seconds,
// and a boolean indicating if the Quaternion exists (i.e. if this is track has rotation animation data).
func (track *AnimationTrack) ValueAsQuaternion(time float64) (Quaternion, bool) {

	if len(track.Keyframes) == 0 {
		return Quaternion{}, false
	}

	if first := track.Keyframes[0]; time <= first.Time {
		return first.Data.AsQuaternion(), true
	} else if last := track.Keyframes[len(track.Keyframes)-1]; time >= last.Time {
		return last.Data.AsQuaternion(), true
	} else {

		var first *Keyframe
		var last *Keyframe

		for _, k := range track.Keyframes {

			if k.Time < time {
				first = k
			} else {
				last = k
				break
			}

		}

		if time == first.Time {
			return first.Data.AsQuaternion(), true
		} else if time == last.Time {
			return last.Data.AsQuaternion(), true
		} else {

			fd := first.Data.AsQuaternion()
			ld := last.Data.AsQuaternion()

			// Linear interpolation
			t := (time - first.Time) / (last.Time - first.Time)

			return fd.Lerp(ld, t), true

		}

	}

}

func newAnimationTrack(trackType string) *AnimationTrack {
	return &AnimationTrack{
		Type:      trackType,
		Keyframes: []*Keyframe{},
	}
}

type AnimationChannel struct {
	Name   string
	Tracks map[string]*AnimationTrack
}

func NewAnimationChannel(name string) *AnimationChannel {
	return &AnimationChannel{
		Name:   name,
		Tracks: map[string]*AnimationTrack{},
	}
}

func (channel *AnimationChannel) AddTrack(trackType string) *AnimationTrack {
	newTrack := newAnimationTrack(trackType)
	channel.Tracks[trackType] = newTrack
	return newTrack
}

// Marker represents a tag as placed in an Animation in a 3D modeler.
type Marker struct {
	Time float64 // Time of the marker in seconds in the Animation.
	Name string  // Name of the marker.
}

// Animation represents an animation of some description; it can have multiple channels, indicating movement, scale, or rotational change of one or more Nodes in the Animation.
type Animation struct {
	library  *Library
	Name     string
	Channels map[string]*AnimationChannel
	Length   float64  // Length of the animation in seconds
	Markers  []Marker // Markers as specified in the Animation from the modeler
}

// NewAnimation creates a new Animation of the name specified.
func NewAnimation(name string) *Animation {
	return &Animation{
		Name:     name,
		Channels: map[string]*AnimationChannel{},
		Markers:  []Marker{},
	}
}

func (animation *Animation) AddChannel(name string) *AnimationChannel {
	newChannel := NewAnimationChannel(name)
	animation.Channels[name] = newChannel
	return newChannel
}

// Library returns the Library from which this Animation was loaded. If it was created in code, this function would return nil.
func (animation *Animation) Library() *Library {
	return animation.library
}

func (animation *Animation) String() string {
	if ReadableReferences {
		return "<" + animation.Name + ">"
	} else {
		return fmt.Sprintf("%p", animation)
	}
}

// AnimationValues indicate the current position, scale, and rotation for a Node.
type AnimationValues struct {
	Position       Vector
	PositionExists bool
	Scale          Vector
	ScaleExists    bool
	Rotation       Quaternion
	RotationExists bool
}

// AnimationPlayer is an object that allows you to play back an animation on a Node.
type AnimationPlayer struct {
	RootNode               INode
	ChannelsToNodes        map[*AnimationChannel]INode
	ChannelsUpdated        bool
	Animation              *Animation
	Playhead               float64 // Playhead of the animation. Setting this to 0 restarts the animation.
	prevPlayhead           float64
	PlaySpeed              float64    // Playback speed in percentage - defaults to 1 (100%)
	Playing                bool       // Whether the player is playing back or not.
	FinishMode             FinishMode // What to do when the player finishes playback. Defaults to looping.
	OnFinish               func()     // Callback indicating the Animation has completed
	finished               bool
	OnMarkerTouch          func(marker Marker, animation *Animation) // Callback indicating when the AnimationPlayer has entered a marker
	touchedMarkers         []Marker
	AnimatedProperties     map[INode]*AnimationValues // The properties that have been animated
	prevAnimatedProperties map[INode]*AnimationValues // The previous properties that have been animated from the previously Play()'d animation
	BlendTime              float64                    // How much time in seconds to blend between two animations
	blendStart             time.Time                  // The time that the blend started
	// If the AnimationPlayer should play the last frame or not. For example, if you have an animation that starts on frame 1 and goes to frame 10,
	// then if PlayLastFrame is on, it will play all frames, INCLUDING frame 10, and only then repeat (if it's set to repeat).
	// Otherwise, it will only play frames 1 - 9, which can be good if your last frame is a repeat of the first to make a cyclical animation.
	// The default for PlayLastFrame is false.
	PlayLastFrame bool
}

// NewAnimationPlayer returns a new AnimationPlayer for the Node.
func NewAnimationPlayer(node INode) *AnimationPlayer {
	return &AnimationPlayer{
		RootNode:               node,
		PlaySpeed:              1,
		FinishMode:             FinishModeLoop,
		AnimatedProperties:     map[INode]*AnimationValues{},
		prevAnimatedProperties: map[INode]*AnimationValues{},
		PlayLastFrame:          false,
	}
}

// Clone returns a clone of the specified AnimationPlayer.
func (ap *AnimationPlayer) Clone() *AnimationPlayer {
	newAP := NewAnimationPlayer(ap.RootNode)

	newAP.ChannelsToNodes = map[*AnimationChannel]INode{}
	for channel, node := range ap.ChannelsToNodes {
		newAP.ChannelsToNodes[channel] = node
	}
	newAP.ChannelsUpdated = ap.ChannelsUpdated

	newAP.Animation = ap.Animation
	newAP.Playhead = ap.Playhead
	newAP.PlaySpeed = ap.PlaySpeed
	newAP.FinishMode = ap.FinishMode
	newAP.OnFinish = ap.OnFinish
	newAP.Playing = ap.Playing
	newAP.PlayLastFrame = ap.PlayLastFrame
	return newAP
}

// SetRoot sets the root node of the animation player to act on. Note that this should be the root node.
func (ap *AnimationPlayer) SetRoot(node INode) {
	ap.RootNode = node
	ap.ChannelsUpdated = false
}

// PlayAnim plays the specified animation back, resetting the playhead if the specified animation is not currently
// playing, or if the animation is paused. If the animation is already playing, Play() does nothing.
func (ap *AnimationPlayer) PlayAnim(animation *Animation) {

	if ap.Animation == animation && ap.Playing {
		return
	}

	ap.Animation = animation
	ap.Playing = true

	if ap.PlaySpeed > 0 {
		ap.Playhead = 0.0
	} else {
		ap.Playhead = animation.Length
	}

	ap.ChannelsUpdated = false

	if ap.BlendTime > 0 {
		ap.prevAnimatedProperties = map[INode]*AnimationValues{}
		for n, v := range ap.AnimatedProperties {
			ap.prevAnimatedProperties[n] = v
		}
		ap.blendStart = time.Now()
	}

}

// Play plays back an animation by name, accessing it through the AnimationPlayer's root node's library. If the animation isn't found,
// it will return an error.
func (ap *AnimationPlayer) Play(animationName string) error {
	anim, ok := ap.RootNode.Library().Animations[animationName]
	if ok {
		ap.PlayAnim(anim)
	}
	return errors.New("Animation named {" + animationName + "} not found in node's owning Library")
}

// assignChannels assigns the player's root node's children to channels in the player. This is called when the channels need to be
// updated after the root node changes.
func (ap *AnimationPlayer) assignChannels() {

	if ap.Animation != nil {

		ap.AnimatedProperties = map[INode]*AnimationValues{}

		ap.ChannelsToNodes = map[*AnimationChannel]INode{}

		childrenRecursive := ap.RootNode.SearchTree().INodes()

		for _, channel := range ap.Animation.Channels {

			if ap.RootNode.Name() == channel.Name {
				ap.ChannelsToNodes[channel] = ap.RootNode
				ap.AnimatedProperties[ap.RootNode] = &AnimationValues{}
				continue
			}

			found := false

			for _, n := range childrenRecursive {

				if n.Name() == channel.Name {
					ap.ChannelsToNodes[channel] = n
					ap.AnimatedProperties[n] = &AnimationValues{}
					found = true
					break
				}

			}

			// If no channel matches, we'll just go with the root

			if !found {
				ap.ChannelsToNodes[channel] = ap.RootNode
				ap.AnimatedProperties[ap.RootNode] = &AnimationValues{}
			}

		}

	}

	ap.ChannelsUpdated = true

}

func (ap *AnimationPlayer) updateValues(dt float64) {

	if ap.Playing {

		if ap.Animation != nil {

			if !ap.ChannelsUpdated {
				ap.assignChannels()
			}

			for _, channel := range ap.Animation.Channels {

				node := ap.ChannelsToNodes[channel]

				if node == nil {
					log.Println("Error: Cannot find matching node for channel " + channel.Name + " for root " + ap.RootNode.Name())
				} else {

					if track, exists := channel.Tracks[TrackTypePosition]; exists {
						// node.SetLocalPositionVecVec(track.ValueAsVector(ap.Playhead))
						if vec, exists := track.ValueAsVector(ap.Playhead); exists {
							ap.AnimatedProperties[node].Position = vec
							ap.AnimatedProperties[node].PositionExists = true
						}
					}

					if track, exists := channel.Tracks[TrackTypeScale]; exists {
						// node.SetLocalScaleVec(track.ValueAsVector(ap.Playhead))
						if vec, exists := track.ValueAsVector(ap.Playhead); exists {
							ap.AnimatedProperties[node].Scale = vec
							ap.AnimatedProperties[node].ScaleExists = true
						}
					}

					if track, exists := channel.Tracks[TrackTypeRotation]; exists {
						if quat, exists := track.ValueAsQuaternion(ap.Playhead); exists {
							// node.SetLocalRotation(NewMatrix4RotateFromQuaternion(quat))
							ap.AnimatedProperties[node].Rotation = quat
							ap.AnimatedProperties[node].RotationExists = true
						}
					}

				}

			}

			ap.Playhead += dt * ap.PlaySpeed

			ph := ap.Playhead

			if !ap.PlayLastFrame {
				ph += dt * ap.PlaySpeed
			}

			if ap.PlaySpeed != 0 {

				for _, marker := range ap.Animation.Markers {
					// fmt.Println(ph, prevPlayhead, marker.Time, ph >= marker.Time, prevPlayhead <= marker.Time)
					// fmt.Println(gteWithMargin(ph, marker.Time))
					// fmt.Println(lteWithMargin(prevPlayhead, marker.Time))
					// if gteWithMargin(ph, marker.Time) && lteWithMargin(prevPlayhead, marker.Time) {
					if ph >= marker.Time && ap.prevPlayhead <= marker.Time {
						// if ap.Playhead >= marker.Time && prevPlayhead <= marker.Time {
						if ap.OnMarkerTouch != nil {
							ap.OnMarkerTouch(marker, ap.Animation)
						}
						ap.touchedMarkers = append(ap.touchedMarkers, marker)
					}
				}

			}

			ap.prevPlayhead = ph

			if ap.FinishMode == FinishModeLoop && (ph >= ap.Animation.Length || ph < 0) {

				if ph > ap.Animation.Length {
					ap.Playhead -= ap.Animation.Length
				}

				if ph < 0 {
					ap.Playhead += ap.Animation.Length
				}

				if ap.OnFinish != nil {
					ap.OnFinish()
				}
				ap.finished = true

			} else if ap.FinishMode == FinishModePingPong && (ph > ap.Animation.Length || ph < 0) {

				if ph > ap.Animation.Length {
					ap.Playhead -= ph - ap.Animation.Length
				}

				finishedLoop := false
				if ph < 0 {
					ap.Playhead *= -1
					finishedLoop = true
				}

				if finishedLoop && ap.OnFinish != nil {
					ap.OnFinish()
				}
				ap.finished = true

				ap.PlaySpeed *= -1

			} else if ap.FinishMode == FinishModeStop && ((ph > ap.Animation.Length && ap.PlaySpeed > 0) || (ph < 0 && ap.PlaySpeed < 0)) {

				if ph > ap.Animation.Length {
					ap.Playhead = ap.Animation.Length
				} else if ph < 0 {
					ap.Playhead = 0
				}

				if ap.OnFinish != nil {
					ap.OnFinish()
				}
				ap.finished = true

				ap.Playing = false

			}

		}

	}

}

// Update updates the animation player by the delta specified in seconds (usually 1/FPS or 1/TARGET FPS), animating the transformation properties of the root node's tree.
func (ap *AnimationPlayer) Update(dt float64) {

	ap.finished = false
	ap.touchedMarkers = []Marker{}

	ap.updateValues(dt)

	if !ap.Playing && !ap.blendStart.IsZero() {
		ap.blendStart = time.Time{}
		ap.prevAnimatedProperties = map[INode]*AnimationValues{}
	}

	if ap.Animation == nil || !ap.Playing {
		return
	}

	for node, props := range ap.AnimatedProperties {

		_, prevExists := ap.prevAnimatedProperties[node]

		if !ap.blendStart.IsZero() && prevExists {

			bp := float64(time.Since(ap.blendStart).Milliseconds()) / (ap.BlendTime * 1000)
			if bp > 1 {
				bp = 1
			}

			start := ap.prevAnimatedProperties[node]

			if start.PositionExists && props.PositionExists {
				diff := props.Position.Sub(start.Position)
				node.SetLocalPositionVec(start.Position.Add(diff.Scale(bp)))
			} else if props.PositionExists {
				node.SetLocalPositionVec(props.Position)
			} else if start.PositionExists {
				node.SetLocalPositionVec(start.Position)
			}

			if start.ScaleExists && props.ScaleExists {
				diff := props.Scale.Sub(start.Scale)
				node.SetLocalScaleVec(start.Scale.Add(diff.Scale(bp)))
			} else if props.ScaleExists {
				node.SetLocalScaleVec(props.Scale)
			} else if start.ScaleExists {
				node.SetLocalScaleVec(start.Scale)
			}

			if start.RotationExists && props.RotationExists {
				rot := start.Rotation.Lerp(props.Rotation, bp).Normalized()
				node.SetLocalRotation(rot.ToMatrix4())
			} else if props.RotationExists {
				node.SetLocalRotation(props.Rotation.ToMatrix4())
			} else if start.RotationExists {
				node.SetLocalRotation(start.Rotation.ToMatrix4())
			}

			if bp == 1 {
				ap.blendStart = time.Time{}
				ap.prevAnimatedProperties = map[INode]*AnimationValues{}
			}

		} else {

			if props.PositionExists {
				node.SetLocalPositionVec(props.Position)
			}
			if props.ScaleExists {
				node.SetLocalScaleVec(props.Scale)
			}
			if props.RotationExists {
				node.SetLocalRotation(props.Rotation.ToMatrix4())
			}

		}

	}

}

// Finished returns whether the AnimationPlayer is finished playing its current animation.
func (ap *AnimationPlayer) Finished() bool {
	return ap.finished
}

// TouchedMarker returns if a marker with the specified name was touched this past frame - note that this relies on calling AnimationPlayer.Update().
func (ap *AnimationPlayer) TouchedMarker(markerName string) bool {

	if ap.Animation == nil {
		return false
	}

	for _, marker := range ap.touchedMarkers {
		if marker.Name == markerName {
			return true
		}
	}
	return false
}

// AfterMarker returns if the AnimationPlayer's playhead is after a marker with the specified name.
func (ap *AnimationPlayer) AfterMarker(markerName string) bool {

	if ap.Animation == nil {
		return false
	}

	for _, marker := range ap.Animation.Markers {
		if marker.Name == markerName && marker.Time < ap.Playhead {
			return true
		}
	}
	return false
}

// BeforeMarker returns if the AnimationPlayer's playhead is before a marker with the specified name.
func (ap *AnimationPlayer) BeforeMarker(markerName string) bool {

	if ap.Animation == nil {
		return false
	}

	for _, marker := range ap.Animation.Markers {
		if marker.Name == markerName && marker.Time > ap.Playhead {
			return true
		}
	}
	return false
}

// func (anim *Animation) TracksToString() string {
// 	str := ""
// 	for trackType, t := range anim.Tracks {
// 		str += trackType + " : "
// 		for _, key := range t.Keyframes {
// 			key.Time
// 		}
// 	}
// 	return str
// }
