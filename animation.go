package tetra3d

import (
	"errors"
	"fmt"
	"log"
	"math"
	"time"
)

const (
	TrackTypePosition = "Pos"
	TrackTypeScale    = "Sca"
	TrackTypeRotation = "Rot"
)

const (
	InterpolationLinear int = iota
	InterpolationConstant
	InterpolationCubic // Unimplemented
)

type Data struct {
	contents any
}

func (data *Data) AsVector() Vector3 {
	return data.contents.(Vector3)
}

func (data *Data) AsQuaternion() Quaternion {
	return data.contents.(Quaternion)
}

// Keyframe represents a single keyframe in an animation for an AnimationTrack.
type Keyframe struct {
	Time float32
	Data Data
}

func newKeyframe(time float32, data Data) *Keyframe {
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
func (track *AnimationTrack) AddKeyframe(time float32, data any) {
	track.Keyframes = append(track.Keyframes, newKeyframe(time, Data{data}))
}

// ValueAsVector returns a Vector associated with the current time in seconds, as well as a boolean indicating if
// the Vector exists (i.e. if the AnimationTrack has location or scale data in the animation). The Vector will be
// interpolated according to time between keyframes.
func (track *AnimationTrack) ValueAsVector(time float32) (Vector3, bool) {

	if len(track.Keyframes) == 0 {
		return Vector3{}, false
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

	// Constant (stepped) interpolation
	if track.Interpolation == InterpolationConstant {
		return fd, true
	}

	// Linear interpolation

	t := (time - first.Time) / (last.Time - first.Time)

	// We still need to implement InterpolationCubic
	// if track.Type == TrackTypePosition || track.Type == TrackTypeScale {
	return fd.Add(ld.Sub(fd).Scale(t)), true
	// }

}

// ValueAsQuaternion returns the Quaternion associated with this AnimationTrack using the given time in seconds,
// and a boolean indicating if the Quaternion exists (i.e. if this is track has rotation animation data).
func (track *AnimationTrack) ValueAsQuaternion(time float32) (Quaternion, bool) {

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

			if track.Interpolation == InterpolationConstant {
				return fd, true
			}

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

// AnimationChannel represents a set of tracks (one for position, scale, and rotation) for the various nodes contained within the Animation.
type AnimationChannel struct {
	Name                string
	Tracks              map[string]*AnimationTrack
	startingPositionSet bool
	startingPosition    Vector3
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
	Time float32 // Time of the marker in seconds in the Animation.
	Name string  // Name of the marker.
}

// Animation represents an animation of some description; it can have multiple channels, indicating movement, scale, or rotational change of one or more Nodes in the Animation.
type Animation struct {
	Name    string
	id      uint32
	library *Library
	// A Channel represents a set of tracks (one for position, scale, and rotation) for the various nodes contained within the Animation.
	Channels   map[string]*AnimationChannel
	Length     float32    // Length of the animation in seconds
	Markers    []Marker   // Markers as specified in the Animation from the modeler
	properties Properties // Animation properties

	// RelativeMotion indicates if an animation's motion happens relative to the object's starting
	// position.
	// When true, animations will play back relative to the node's transform prior to
	// starting the animation. When false, the animation will play back in absolute space.
	// For example, let's say an animation moved a cube from {0, 1, 2} to {10, 1, 2}.
	// If RelativeMotion is on, then if the cube started at position {3, 3, 3}, it would end up at
	// {13, 4, 5}. If RelativeMotion were off, it would teleport to {0, 1, 2} when the animation started.
	RelativeMotion bool
}

var animationID uint32 = 1

// NewAnimation creates a new Animation of the name specified.
func NewAnimation(name string) *Animation {
	anim := &Animation{
		Name:       name,
		id:         animationID,
		Channels:   map[string]*AnimationChannel{},
		Markers:    []Marker{},
		properties: NewProperties(),
	}
	animationID++
	return anim
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

// FindMarker returns a marker, found by name, and a boolean value indicating if a marker by the specified name was found or not.
func (animation *Animation) FindMarker(markerName string) (Marker, bool) {
	for _, m := range animation.Markers {
		if m.Name == markerName {
			return m, true
		}
	}
	return Marker{}, false
}

func (animation *Animation) Properties() Properties {
	return animation.properties
}

// AnimationValues indicate the current position, scale, and rotation for a Node.
type AnimationValues struct {
	Position              Vector3
	PositionExists        bool
	PositionInterpolation int
	Scale                 Vector3
	ScaleExists           bool
	ScaleInterpolation    int
	Rotation              Quaternion
	RotationExists        bool
	RotationInterpolation int
	channel               *AnimationChannel
}

// AnimationPlayer is an object that allows you to play back an animation on a Node.
type AnimationPlayer struct {
	RootNode               INode
	ChannelsToNodes        map[*AnimationChannel]INode
	ChannelsUpdated        bool
	Animation              *Animation
	Playhead               float32 // Playhead of the animation. Setting this to 0 restarts the animation.
	prevPlayhead           float32
	prevFinishedAnimation  string
	justLooped             bool
	PlaySpeed              float32                    // Playback speed in percentage - defaults to 1 (100%)
	Playing                bool                       // Whether the player is playing back or not.
	FinishMode             FinishMode                 // What to do when the player finishes playback. Defaults to looping.
	OnFinish               func(animation *Animation) // Callback indicating the Animation has completed
	finished               bool
	OnMarkerTouch          func(marker Marker, animation *Animation) // Callback indicating when the AnimationPlayer has entered a marker
	touchedMarkers         []Marker
	AnimatedProperties     map[INode]AnimationValues // The properties that have been animated
	currentProperties      map[INode]AnimationValues
	prevAnimatedProperties map[INode]AnimationValues // The previous properties that have been animated from the previously Play()'d animation
	BlendTime              float32                   // How much time in seconds to blend between two animations
	blendStart             time.Time                 // The time that the blend started
	// If the AnimationPlayer should play the last frame or not. For example, if you have an animation that starts on frame 1 and goes to frame 10,
	// then if PlayLastFrame is on, it will play all frames, INCLUDING frame 10, and only then repeat (if it's set to repeat).
	// Otherwise, it will only play frames 1 - 9, which can be good if your last frame is a repeat of the first to make a cyclical animation.
	// The default for PlayLastFrame is false.
	PlayLastFrame bool

	startingPosition Vector3
	startingScale    Vector3
	startingRotation Matrix4
}

// NewAnimationPlayer returns a new AnimationPlayer for the Node.
func NewAnimationPlayer(node INode) *AnimationPlayer {
	return &AnimationPlayer{
		RootNode:               node,
		PlaySpeed:              1,
		FinishMode:             FinishModeLoop,
		AnimatedProperties:     map[INode]AnimationValues{},
		currentProperties:      map[INode]AnimationValues{},
		prevAnimatedProperties: map[INode]AnimationValues{},
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

// Play plays the specified animation back, resetting the playhead if the specified animation is not currently
// playing, or if the animation is paused. If the animation is already playing, Play() does nothing.
func (ap *AnimationPlayer) Play(animation *Animation) {

	if animation == nil {
		return
	}

	if ap.Animation == animation && ap.Playing {
		return
	}

	if ap.Animation != animation {
		ap.startingPosition = ap.RootNode.LocalPosition()
		ap.startingRotation = ap.RootNode.LocalRotation()
		ap.startingScale = ap.RootNode.LocalScale()
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
		ap.prevAnimatedProperties = map[INode]AnimationValues{}
		for n, v := range ap.currentProperties {
			ap.prevAnimatedProperties[n] = v
		}
		ap.blendStart = time.Now()
	}

}

// PlayByName plays back an animation by name, accessing it through the AnimationPlayer's root node's library. If the animation isn't found,
// it will return an error.
func (ap *AnimationPlayer) PlayByName(animationName string) error {
	if anim := ap.RootNode.Library().AnimationByName(animationName); anim == nil {
		return errors.New("Animation named {" + animationName + "} not found in node's owning Library")
	} else {
		ap.Play(anim)
	}
	return nil
}

// Stop stops the AnimationPlayer's playback. Note that this is fundamentally the same as calling ap.Playing = false (for now).
func (ap *AnimationPlayer) Stop() {
	ap.Playing = false
}

// assignChannels assigns the player's root node's children to channels in the player. This is called when the channels need to be
// updated after the root node changes.
func (ap *AnimationPlayer) assignChannels() {

	if !ap.ChannelsUpdated {

		if ap.Animation != nil {

			ap.AnimatedProperties = map[INode]AnimationValues{}
			ap.currentProperties = map[INode]AnimationValues{}

			ap.ChannelsToNodes = map[*AnimationChannel]INode{}

			childrenRecursive := ap.RootNode.SearchTree().INodes()

			for _, channel := range ap.Animation.Channels {

				if ap.RootNode.Name() == channel.Name {
					ap.ChannelsToNodes[channel] = ap.RootNode
					ap.AnimatedProperties[ap.RootNode] = AnimationValues{
						channel: channel,
					}
					continue
				}

				found := false

				for _, n := range childrenRecursive {

					if n.Name() == channel.Name {
						ap.ChannelsToNodes[channel] = n
						ap.AnimatedProperties[n] = AnimationValues{
							channel: channel,
						}
						found = true
						break
					}

				}

				// If no channel matches, we'll just go with the root

				if !found {
					ap.ChannelsToNodes[channel] = ap.RootNode
					ap.AnimatedProperties[ap.RootNode] = AnimationValues{
						channel: channel,
					}
				}

			}

		}

		ap.ChannelsUpdated = true

	}

}

func (ap *AnimationPlayer) updateValues(dt float32) {

	if ap.Animation != nil {

		ap.assignChannels()

		for _, channel := range ap.Animation.Channels {

			node := ap.ChannelsToNodes[channel]

			if node == nil {
				log.Println("Error: Cannot find matching node for channel " + channel.Name + " for root " + ap.RootNode.Name())
			} else {

				n := ap.AnimatedProperties[node]

				if track, exists := channel.Tracks[TrackTypePosition]; exists {
					// node.SetLocalPositionVecVec(track.ValueAsVector(ap.Playhead))
					if vec, exists := track.ValueAsVector(ap.Playhead); exists {
						n.Position = vec
						n.PositionExists = true
						n.PositionInterpolation = track.Interpolation

						// set the starting position for the animation; for relative motion, all
						// movement is relative to this position
						if !channel.startingPositionSet {
							channel.startingPositionSet = true
							channel.startingPosition, _ = track.ValueAsVector(-math.MaxFloat32)
						}

					}
				}

				if track, exists := channel.Tracks[TrackTypeScale]; exists {
					// node.SetLocalScaleVec(track.ValueAsVector(ap.Playhead))
					if vec, exists := track.ValueAsVector(ap.Playhead); exists {
						n.Scale = vec
						n.ScaleExists = true
						n.ScaleInterpolation = track.Interpolation
					}
				}

				if track, exists := channel.Tracks[TrackTypeRotation]; exists {
					if quat, exists := track.ValueAsQuaternion(ap.Playhead); exists {
						// node.SetLocalRotation(NewMatrix4RotateFromQuaternion(quat))
						n.Rotation = quat
						n.RotationExists = true
						n.RotationInterpolation = track.Interpolation
					}
				}

				ap.AnimatedProperties[node] = n

			}

		}

		ap.Playhead += dt * ap.PlaySpeed

		ph := ap.Playhead

		if !ap.PlayLastFrame {
			ph += dt * ap.PlaySpeed
		}

		if ap.PlaySpeed != 0 {

			for _, marker := range ap.Animation.Markers {
				if (ap.PlaySpeed > 0 && ph >= marker.Time && (ap.prevPlayhead < marker.Time || ap.justLooped)) || (ap.PlaySpeed < 0 && ph <= marker.Time && (ap.prevPlayhead > marker.Time || ap.justLooped)) {
					if ap.OnMarkerTouch != nil {
						ap.OnMarkerTouch(marker, ap.Animation)
					}
					ap.touchedMarkers = append(ap.touchedMarkers, marker)
				}
			}

		}

		ap.prevPlayhead = ph

		ap.justLooped = false

		if ap.FinishMode == FinishModeLoop && (ph >= ap.Animation.Length || ph < 0) {

			if ph >= ap.Animation.Length {
				ap.Playhead -= ap.Animation.Length
			}

			if ph < 0 {
				ap.Playhead += ap.Animation.Length
			}

			ap.justLooped = true

			if ap.OnFinish != nil {
				ap.OnFinish(ap.Animation)
			}
			ap.finished = true
			ap.prevFinishedAnimation = ap.Animation.Name

		} else if ap.FinishMode == FinishModePingPong && (ph >= ap.Animation.Length || ph < 0) {

			if ph >= ap.Animation.Length {
				ap.Playhead -= ph - ap.Animation.Length
			}

			finishedLoop := false
			if ph < 0 {
				ap.Playhead *= -1
				finishedLoop = true
			}

			if finishedLoop {
				ap.justLooped = true
				if ap.OnFinish != nil {
					ap.OnFinish(ap.Animation)
				}
			}
			ap.finished = true
			ap.prevFinishedAnimation = ap.Animation.Name

			ap.PlaySpeed *= -1

		} else if ap.FinishMode == FinishModeStop && ((ph >= ap.Animation.Length && ap.PlaySpeed > 0) || (ph <= 0 && ap.PlaySpeed < 0)) {

			if ph >= ap.Animation.Length {
				ap.Playhead = ap.Animation.Length
			} else if ph <= 0 {
				ap.Playhead = 0
			}

			if ap.OnFinish != nil {
				ap.OnFinish(ap.Animation)
			}
			ap.finished = true
			ap.prevFinishedAnimation = ap.Animation.Name

			ap.Playing = false

		}

	}

}

// Update updates the animation player by the delta specified in seconds (usually 1/FPS or 1/TARGET FPS), animating the transformation properties of the root node's tree.
func (ap *AnimationPlayer) Update(dt float32) {

	ap.finished = false
	ap.touchedMarkers = ap.touchedMarkers[:0]

	if !ap.Playing && !ap.blendStart.IsZero() {
		ap.blendStart = time.Time{}
		ap.prevAnimatedProperties = map[INode]AnimationValues{}
	}

	if ap.Animation == nil || !ap.Playing {
		return
	}

	ap.forceUpdate(dt)

}

// SetPlayhead sets the playhead of the animation player to the specified time in seconds, and
// also performs an update of the animated nodes.
func (ap *AnimationPlayer) SetPlayhead(time float32) {
	ap.Playhead = time
	ap.forceUpdate(0)
}

func (ap *AnimationPlayer) forceUpdate(dt float32) {

	ap.updateValues(dt)

	for node, props := range ap.AnimatedProperties {

		_, prevExists := ap.prevAnimatedProperties[node]

		var targetPosition Vector3
		var posSet bool
		var targetScale Vector3
		var scaleSet bool
		var targetRotation Quaternion
		var rotSet bool

		if !ap.blendStart.IsZero() && prevExists {

			bp := float32(time.Since(ap.blendStart).Milliseconds()) / (ap.BlendTime * 1000)
			if bp > 1 {
				bp = 1
			}

			start := ap.prevAnimatedProperties[node]

			if start.PositionExists && props.PositionExists && props.PositionInterpolation != InterpolationConstant {
				diff := props.Position.Sub(start.Position)
				targetPosition = start.Position.Add(diff.Scale(bp))
				posSet = true
			} else if props.PositionExists {
				targetPosition = props.Position
				posSet = true
			} else if start.PositionExists {
				targetPosition = start.Position
				posSet = true
			}

			if start.ScaleExists && props.ScaleExists && props.ScaleInterpolation != InterpolationConstant {
				diff := props.Scale.Sub(start.Scale)
				targetScale = start.Scale.Add(diff.Scale(bp))
				scaleSet = true
			} else if props.ScaleExists {
				targetScale = props.Scale
				scaleSet = true
			} else if start.ScaleExists {
				targetScale = start.Scale
				scaleSet = true
			}

			if start.RotationExists && props.RotationExists && props.RotationInterpolation != InterpolationConstant {
				targetRotation = start.Rotation.Lerp(props.Rotation, bp).Normalized()
				rotSet = true
			} else if props.RotationExists {
				targetRotation = props.Rotation
				rotSet = true
			} else if start.RotationExists {
				targetRotation = start.Rotation
				rotSet = true
			}

			if bp == 1 {
				ap.blendStart = time.Time{}
				ap.prevAnimatedProperties = map[INode]AnimationValues{}
			}

		} else {

			if props.PositionExists {
				targetPosition = props.Position
				posSet = true
			}
			if props.ScaleExists {
				targetScale = props.Scale
				scaleSet = true
			}
			if props.RotationExists {
				targetRotation = props.Rotation
				rotSet = true
			}

		}

		// If we're blending, we need to keep track of the current status of the animation, as we could blend not just
		// between two animations, but between a blending result and the next animation (i.e. you start animation A
		// after playing animation B - this creates a blend. However, halfway through the blend, you start animation D,
		// meaning we need to blend between this current state and animation D).
		if ap.BlendTime > 0 {

			if _, exists := ap.currentProperties[node]; !exists {
				ap.currentProperties[node] = AnimationValues{}
			}

			n := ap.currentProperties[node]

			n.channel = props.channel
			n.PositionExists = posSet
			if posSet {
				n.Position = targetPosition
			}
			n.RotationExists = rotSet
			if rotSet {
				n.Rotation = targetRotation
			}
			n.ScaleExists = scaleSet
			if scaleSet {
				n.Scale = targetScale
			}

			ap.currentProperties[node] = n

		}

		if posSet {
			if ap.Animation.RelativeMotion {
				node.SetLocalPositionVec(ap.startingPosition.Add(targetPosition.Sub(props.channel.startingPosition)))
			} else {
				node.SetLocalPositionVec(targetPosition)
			}
		}

		if scaleSet {
			if ap.Animation.RelativeMotion {
				node.SetLocalScaleVec(ap.startingScale.Mult(targetScale))
			} else {
				node.SetLocalScaleVec(targetScale)
			}
		}

		if rotSet {
			if ap.Animation.RelativeMotion {
				node.SetLocalRotation(ap.startingRotation.Mult(targetRotation.ToMatrix4()))
			} else {
				node.SetLocalRotation(targetRotation.ToMatrix4())
			}
		}

	}

}

// Finished returns whether the AnimationPlayer is finished playing its current animation.
func (ap *AnimationPlayer) Finished() bool {
	return ap.finished
}

// FinishedPlayingByName returns whether the AnimationPlayer just got finished playing an animation of the specified name.
func (ap *AnimationPlayer) FinishedPlayingByName(animName string) bool {
	return ap.prevFinishedAnimation == animName && ap.Finished()
}

// IsPlayingByName returns if the AnimationPlayer is playing an animation of the given name.
func (ap *AnimationPlayer) IsPlayingByName(animName string) bool {
	return ap.Animation != nil && ap.Animation.Name == animName
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
