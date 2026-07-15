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

func (data Data) AsVector() Vector3 {
	return data.contents.(Vector3)
}

func (data Data) AsQuaternion() Quaternion {
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
	return fd.Lerp(ld, t), true
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

			return fd.Slerp(ld, t), true

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
	name    string
	id      uint32
	library *Library
	// A Channel represents a set of tracks (one for position, scale, and rotation) for the various nodes contained within the Animation.
	channels   map[string]*AnimationChannel
	length     float32           // Length of the animation in seconds
	markers    []Marker          // Markers as specified in the Animation from the modeler
	properties *Properties       // Animation properties
	loopMode   AnimationLoopMode // Looping mode for the animation

	// relativeMotion indicates if an animation's motion happens relative to the object's starting
	// position.
	// When true, animations will play back relative to the node's transform prior to
	// starting the animation. When false, the animation will play back in absolute space.
	// For example, let's say an animation moved a cube from {0, 1, 2} to {10, 1, 2}.
	// If relativeMotion is on, then if the cube started at position {3, 3, 3}, it would end up at
	// {13, 1, 2}. If relativeMotion were off, it would teleport to {0, 1, 2} when the animation started.
	relativeMotion bool
}

var animationID uint32 = 1

// NewAnimation creates a new Animation of the name specified.
func NewAnimation(name string) *Animation {
	anim := &Animation{
		name:       name,
		id:         animationID,
		channels:   map[string]*AnimationChannel{},
		markers:    []Marker{},
		properties: NewProperties(),
	}
	animationID++
	return anim
}

// Adds an animation channel to the Animation.
func (animation *Animation) AddChannel(name string) *AnimationChannel {
	newChannel := NewAnimationChannel(name)
	animation.channels[name] = newChannel
	return newChannel
}

// Returns the Animation's name.
func (animation *Animation) Name() string {
	return animation.name
}

// Sets the name of the animation.
func (animation *Animation) SetName(name string) {
	animation.name = name
}

// Returns the length of the animation in seconds.
func (animation *Animation) Length() float32 {
	// TODO: Replace this with a function to calculate the length of the animation.
	return animation.length
}

// Returns if the animation is set to work with relative motion.
func (animation *Animation) RelativeMotion() bool {
	return animation.relativeMotion
}

// Sets relative motion for the Animation.
func (animation *Animation) SetRelativeMotion(relativeMotion bool) {
	animation.relativeMotion = relativeMotion
}

// Library returns the Library from which this Animation was loaded. If it was created in code, this function would return nil.
func (animation *Animation) Library() *Library {
	return animation.library
}

func (animation *Animation) String() string {
	if ReadableReferences {
		return "< Animation : " + animation.name + ">"
	} else {
		return fmt.Sprintf("%p", animation)
	}
}

// FindMarker returns a marker, found by name, and a boolean value indicating if a marker by the specified name was found or not.
func (animation *Animation) FindMarker(markerName string) (Marker, bool) {
	for _, m := range animation.markers {
		if m.Name == markerName {
			return m, true
		}
	}
	return Marker{}, false
}

func (animation *Animation) Properties() *Properties {
	return animation.properties
}

// Returns the loop mode for the Animation.
func (animation *Animation) LoopMode() AnimationLoopMode {
	return animation.loopMode
}

// Sets the loop mode for the given animation.
func (animation *Animation) SetLoopMode(loopMode AnimationLoopMode) {
	animation.loopMode = loopMode
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

// Represents a function to call when an animation player finishes playing an animation.
type AnimationPlayerCallbackFunction func(ap *AnimationPlayer, loopCount int)

// An object to sequence events to execute once an animation is finished playing.
type AnimationSequence struct {
	player    *AnimationPlayer
	error     error
	callbacks []AnimationPlayerCallbackFunction
	index     int
	update    bool
}

// Clears the callback's queued callback functions.
func (apc *AnimationSequence) Clear() *AnimationSequence {
	apc.callbacks = apc.callbacks[:0]
	apc.index = -1
	return apc
}

// Returns a clone of the AnimationPlayerCallback.
func (apc *AnimationSequence) Clone() *AnimationSequence {
	return &AnimationSequence{
		error:     apc.error,
		callbacks: append([]AnimationPlayerCallbackFunction{}, apc.callbacks...),
		index:     apc.index,
		player:    apc.player,
	}
}

// Schedules a callback function to execute when an animation player finishes playing back the previous animation,
// or to start with if the owning AnimationPlayer is not playing anything currently.
// Note that the callback executes on animation loop / completion.
func (apc *AnimationSequence) Then(callbackFunction AnimationPlayerCallbackFunction) *AnimationSequence {
	if apc.error == nil {
		if apc.index > len(apc.callbacks) {
			apc.Clear()
		}
		apc.update = true
		apc.callbacks = append(apc.callbacks, callbackFunction)
	}
	return apc
}

// Schedules playback of a sequence of animations to play when an animation player finishes playing back the previous animation,
// or to start with if the owning AnimationPlayer is not playing anything currently.
// Note that the callback executes on animation loop / completion.
func (apc *AnimationSequence) ThenPlay(anims ...*Animation) *AnimationSequence {
	for _, anim := range anims {
		apc.update = true
		apc.Then(func(ap *AnimationPlayer, loopCount int) {
			ap.Play(anim)
		})
	}
	return apc
}

// Schedules playback of a sequence of animations (retrieved from the AnimationPlayer's library by name)
// to play when an animation player finishes playing back the previous animation,
// or to start with if the owning AnimationPlayer is not playing anything currently.
func (apc *AnimationSequence) ThenPlayByName(animNames ...string) *AnimationSequence {
	for _, name := range animNames {
		apc.update = true

		apc.Then(func(ap *AnimationPlayer, loopCount int) {
			anim := name
			ap.PlayByName(anim)
		})

	}
	return apc
}

// Returns the error the callback holds if it cannot proceed with playback for whatever reason (e.g.
// can't find an animation by name).
func (apc *AnimationSequence) Error() error {
	return apc.error
}

// Clears the error in the callback.
func (apc *AnimationSequence) ClearError() {
	apc.error = nil
}

func (apc *AnimationSequence) run() {

	if apc.update {

		var callback AnimationPlayerCallbackFunction

		if apc.index >= 0 && apc.index < len(apc.callbacks) {
			callback = apc.callbacks[apc.index]
		} else {
			return
		}

		if callback != nil {
			apc.player.inPlaybackCallback = true
			callback(apc.player, apc.player.loopCount)
			apc.player.inPlaybackCallback = false
			apc.update = false
		}

	}

}

func (apc *AnimationSequence) next() {
	if apc.index+1 <= len(apc.callbacks) {
		apc.index++
		apc.update = true
	}
}

// AnimationPlayer is an object that allows you to play back an animation on a Node.
type AnimationPlayer struct {
	RootNode              INode
	ChannelsToNodes       map[*AnimationChannel]INode
	ChannelsUpdated       bool
	animation             *Animation
	playhead              float32 // Playhead of the animation. Setting this to 0 restarts the animation.
	prevPlayhead          float32
	prevFinishedAnimation string
	justLooped            bool
	PlaySpeed             float32 // Playback speed in percentage - defaults to 1 (100%)
	playing               bool    // Whether the player is playing back or not.
	finished              bool
	loopCount             int

	OnMarkerTouch          func(marker Marker, animation *Animation) // Callback indicating when the AnimationPlayer has entered a marker
	touchedMarkers         []Marker
	AnimatedProperties     map[INode]AnimationValues // The properties that have been animated
	currentProperties      map[INode]AnimationValues
	prevAnimatedProperties map[INode]AnimationValues // The previous properties that have been animated from the previously Play()'d animation
	BlendTime              float32                   // How much time in seconds to blend between two animations
	blendStart             time.Time                 // The time that the blend started

	relativePrevPosition  Vector3
	relativePrevScale     Vector3
	relativePrevRotation  Quaternion
	relativeNotFirstFrame bool

	sequence           *AnimationSequence
	inPlaybackCallback bool
}

// NewAnimationPlayer returns a new AnimationPlayer for the Node.
func NewAnimationPlayer(node INode) *AnimationPlayer {
	ap := &AnimationPlayer{
		RootNode:               node,
		PlaySpeed:              1,
		AnimatedProperties:     map[INode]AnimationValues{},
		currentProperties:      map[INode]AnimationValues{},
		prevAnimatedProperties: map[INode]AnimationValues{},
		ChannelsToNodes:        map[*AnimationChannel]INode{},
	}
	ap.sequence = &AnimationSequence{
		player: ap,
	}
	return ap
}

// Clone returns a clone of the specified AnimationPlayer.
func (ap *AnimationPlayer) Clone() *AnimationPlayer {
	newAP := NewAnimationPlayer(ap.RootNode)

	for channel, node := range ap.ChannelsToNodes {
		newAP.ChannelsToNodes[channel] = node
	}
	newAP.ChannelsUpdated = ap.ChannelsUpdated

	newAP.sequence = ap.sequence.Clone()
	newAP.sequence.player = newAP

	newAP.animation = ap.animation
	newAP.playhead = ap.playhead
	newAP.PlaySpeed = ap.PlaySpeed
	newAP.playing = ap.playing
	return newAP
}

// SetRoot sets the root node of the animation player to act on. Note that this should be the root node.
func (ap *AnimationPlayer) SetRoot(node INode) {
	ap.RootNode = node
	ap.ChannelsUpdated = false
}

// Plays the specified animations back in sequence, starting with the first one.
// If the first animation is already playing, the function does nothing and doesn't alter any sequenced animation set.
// Calling this clears any sequenced animations present in the AnimationPlayerCallback.
// The function returns the AnimationPlayer's AnimationSequence object to allow sequencing events after the animations finish playback.
// If no animations are provided to the function, the AnimationSequence object will hold an error indicating this fact.
func (ap *AnimationPlayer) Play(animations ...*Animation) *AnimationSequence {

	if len(animations) == 0 {
		ap.sequence.error = errors.New("can't call AnimationPlayer.Play() with no animation names provided")
		return ap.sequence
	}

	animation := animations[0]

	// No animation specified
	if animation == nil {
		return ap.sequence
	}

	// Already playing
	if ap.animation == animation && ap.playing {
		return ap.sequence
	}

	ap.animation = animation

	ap.playing = true
	ap.loopCount = 0

	if ap.PlaySpeed > 0 {
		ap.playhead = 0.0
	} else {
		ap.playhead = animation.length
	}

	ap.ChannelsUpdated = false

	if ap.BlendTime > 0 {
		clear(ap.prevAnimatedProperties)
		for n, v := range ap.currentProperties {
			ap.prevAnimatedProperties[n] = v
		}
		ap.blendStart = time.Now()
	}

	ap.relativeNotFirstFrame = false

	// Clear queued animations because it seems good to be explicit with either queueing animations or explicitly ordering them to play
	if !ap.inPlaybackCallback {
		ap.sequence.Clear()
		ap.sequence.ThenPlay(animations[1:]...)
	}

	return ap.sequence
}

// Plays the specified animations back in sequence, starting with the first one.
// If the first animation is already playing, the function does nothing and doesn't alter any sequenced animation set.
// Calling this clears any sequenced animations present in the AnimationSequence object.
// The function returns the AnimationPlayer's AnimationSequence object to allow sequencing events after the animations finish playback.
// If any animations aren't found or no animation names are provided, the AnimationSequence object will hold an error.
func (ap *AnimationPlayer) PlayByName(animationNames ...string) *AnimationSequence {
	if len(animationNames) == 0 {
		ap.sequence.error = errors.New("can't call AnimationPlayer.PlayByName() with no animation names provided")
		return ap.sequence
	}

	anims := []*Animation{}

	for _, an := range animationNames {
		if anim := ap.RootNode.Library().AnimationByName(an); anim != nil {
			anims = append(anims, anim)
		} else {
			ap.sequence.error = errors.New("Animation named {" + an + "} not found in node's owning Library; did you export it and stash its action?")
		}
	}

	ap.Play(anims...)

	return ap.sequence

}

// Pauses playback on the AnimationPlayer.
func (ap *AnimationPlayer) Pause() {
	ap.playing = false
}

// Resumes playback on the AnimationPlayer.
func (ap *AnimationPlayer) Resume() {
	ap.playing = true
}

// Returns the trigger object for sequencing animations or triggering them on animation completion.
// If clear is true, then any currently playing animation will be immediately halted.
func (ap *AnimationPlayer) Trigger() *AnimationSequence {
	return ap.sequence
}

// Returns the currently playing animation for the AnimationPlayer.
func (ap *AnimationPlayer) Animation() *Animation {
	return ap.animation
}

// assignChannels assigns the player's root node's children to channels in the player. This is called when the channels need to be
// updated after the root node changes.
func (ap *AnimationPlayer) assignChannels() {

	if !ap.ChannelsUpdated {

		if ap.animation != nil {

			clear(ap.AnimatedProperties)
			clear(ap.currentProperties)
			clear(ap.ChannelsToNodes)

			for _, channel := range ap.animation.channels {

				if ap.RootNode.Name() == channel.Name {
					ap.ChannelsToNodes[channel] = ap.RootNode
					ap.AnimatedProperties[ap.RootNode] = AnimationValues{
						channel: channel,
					}
					continue
				}

				found := false

				ap.RootNode.ForEachChild(true, func(child INode, index int) bool {

					if child.Name() == channel.Name {
						ap.ChannelsToNodes[channel] = child
						ap.AnimatedProperties[child] = AnimationValues{
							channel: channel,
						}
						found = true
						return false
					}

					return true

				})

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

	if ap.animation != nil {

		ap.assignChannels()

		for _, channel := range ap.animation.channels {

			node := ap.ChannelsToNodes[channel]

			if node == nil {
				log.Println("Error: Cannot find matching node for channel " + channel.Name + " for root " + ap.RootNode.Name())
			} else {

				n := ap.AnimatedProperties[node]

				if track, exists := channel.Tracks[TrackTypePosition]; exists {
					// node.SetLocalPositionVecVec(track.ValueAsVector(ap.Playhead))
					if vec, exists := track.ValueAsVector(ap.playhead); exists {
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
					if vec, exists := track.ValueAsVector(ap.playhead); exists {
						n.Scale = vec
						n.ScaleExists = true
						n.ScaleInterpolation = track.Interpolation
					}
				}

				if track, exists := channel.Tracks[TrackTypeRotation]; exists {
					if quat, exists := track.ValueAsQuaternion(ap.playhead); exists {
						// node.SetLocalRotation(NewMatrix4RotateFromQuaternion(quat))
						n.Rotation = quat
						n.RotationExists = true
						n.RotationInterpolation = track.Interpolation
					}
				}

				ap.AnimatedProperties[node] = n

			}

		}

		ap.playhead += dt * ap.PlaySpeed

		ph := ap.playhead

		if ap.PlaySpeed != 0 {

			for _, marker := range ap.animation.markers {
				if (ap.PlaySpeed > 0 && ph >= marker.Time && (ap.prevPlayhead < marker.Time || ap.justLooped)) || (ap.PlaySpeed < 0 && ph <= marker.Time && (ap.prevPlayhead > marker.Time || ap.justLooped)) {
					if ap.OnMarkerTouch != nil {
						ap.OnMarkerTouch(marker, ap.animation)
					}
					ap.touchedMarkers = append(ap.touchedMarkers, marker)
				}
			}

		}

		ap.prevPlayhead = ph

		ap.justLooped = false

		if ap.animation.loopMode == AnimationLoopModeLoop && (ph >= ap.animation.length || ph < 0) {

			// Set to be the start or end of the animation because otherwise the animation hitches
			if ph >= ap.animation.length {
				ap.playhead -= ap.animation.length
			}

			if ph < 0 {
				ap.playhead += ap.animation.length
			}

			ph = clamp(ph, 0, ap.animation.length)

			ap.justLooped = true

			ap.loopCount++
			ap.finished = true
			ap.prevFinishedAnimation = ap.animation.name

			ap.sequence.next()

		} else if ap.animation.loopMode == AnimationLoopModePingPong && (ph >= ap.animation.length || ph < 0) {

			if ph >= ap.animation.length {
				ap.playhead -= ap.playhead - ap.animation.length
			}

			if ph < 0 {
				ap.playhead = -ap.playhead
				ap.loopCount++
			}

			ap.finished = true
			ap.prevFinishedAnimation = ap.animation.name

			ap.PlaySpeed *= -1

			ap.sequence.next()

		} else if ap.animation.loopMode == AnimationLoopModeOneshot && ((ph >= ap.animation.length && ap.PlaySpeed > 0) || (ph <= 0 && ap.PlaySpeed < 0)) {

			prevAnim := ap.animation
			ap.prevFinishedAnimation = prevAnim.name

			ph = clamp(ph, 0, ap.animation.length)

			ap.finished = true
			ap.playing = false

			ap.sequence.next()

		}

	}

}

// Update updates the animation player by the delta specified in seconds (usually 1/FPS or 1/TARGET FPS), animating the transformation properties of the root node's tree.
func (ap *AnimationPlayer) Update(dt float32) {

	ap.finished = false
	ap.touchedMarkers = ap.touchedMarkers[:0]

	if !ap.playing && !ap.blendStart.IsZero() {
		ap.blendStart = time.Time{}
		clear(ap.prevAnimatedProperties)
	}

	ap.sequence.run()

	ap.forceUpdate(dt)

}

// SetPlayhead sets the playhead of the animation player to the specified time in seconds, and
// also performs an update of the animated nodes.
func (ap *AnimationPlayer) SetPlayhead(time float32) {
	ap.playhead = time
	ap.forceUpdate(0)
}

func (ap *AnimationPlayer) Playhead() float32 {
	return ap.playhead
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

			if start.PositionExists && props.PositionExists {
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

			if start.ScaleExists && props.ScaleExists {
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

			if start.RotationExists && props.RotationExists {
				targetRotation = start.Rotation.Slerp(props.Rotation, bp).Unit()
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
				clear(ap.prevAnimatedProperties)
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
			if ap.animation.relativeMotion {
				if ap.relativeNotFirstFrame {
					node.MoveVec(targetPosition.Sub(ap.relativePrevPosition))
				}
			} else {
				node.SetLocalPositionVec(targetPosition)
			}
			ap.relativePrevPosition = targetPosition
		}

		if scaleSet {
			if ap.animation.relativeMotion {
				if ap.relativeNotFirstFrame {
					node.GrowVec(targetScale.Sub(ap.relativePrevScale))
				}
			} else {
				node.SetLocalScaleVec(targetScale)
			}
			ap.relativePrevScale = targetScale
		}

		if rotSet {
			if ap.animation.relativeMotion {
				if ap.relativeNotFirstFrame {
					node.RotateMatrix4(targetRotation.Inverted().Mult(ap.relativePrevRotation).ToMatrix4())
				}
			} else {
				node.SetLocalRotation(targetRotation.ToMatrix4())
			}
			ap.relativePrevRotation = targetRotation
		}

	}

	ap.relativeNotFirstFrame = true

}

// Finished returns whether the AnimationPlayer is finished playing its current animation.
func (ap *AnimationPlayer) Finished() bool {
	return ap.finished
}

// FinishedPlayingByName returns whether the AnimationPlayer just got finished playing an animation of the specified name.
func (ap *AnimationPlayer) FinishedPlayingByName(animName string) bool {
	return ap.prevFinishedAnimation == animName && ap.Finished()
}

func (ap *AnimationPlayer) IsPlaying() bool {
	return ap.playing
}

// Returns if the AnimationPlayer is playing an animation of the given name.
func (ap *AnimationPlayer) IsPlayingByName(animName string) bool {
	return ap.playing && ap.animation != nil && ap.animation.name == animName
}

// Returns if the AnimationPlayer is playing an animation with a property of the given name.
func (ap *AnimationPlayer) IsPlayingByPropName(propName string) bool {
	return ap.playing && ap.animation != nil && ap.animation.Properties().Has(propName)
}

// Returns if the AnimationPlayer is playing an animation with a property of the given name and value.
func (ap *AnimationPlayer) IsPlayingByPropNameValue(propName string, value any) bool {
	return ap.playing && ap.animation != nil && ap.animation.Properties().Has(propName) && ap.animation.Properties().GetByName(propName).Value == value
}

// TouchedMarker returns if a marker with the specified name was touched this past frame - note that this relies on calling AnimationPlayer.Update().
func (ap *AnimationPlayer) TouchedMarker(markerName string) bool {

	if ap.animation == nil {
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

	if ap.animation == nil {
		return false
	}

	for _, marker := range ap.animation.markers {
		if marker.Name == markerName && marker.Time < ap.playhead {
			return true
		}
	}
	return false
}

// BeforeMarker returns if the AnimationPlayer's playhead is before a marker with the specified name.
func (ap *AnimationPlayer) BeforeMarker(markerName string) bool {

	if ap.animation == nil {
		return false
	}

	for _, marker := range ap.animation.markers {
		if marker.Name == markerName && marker.Time > ap.playhead {
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
