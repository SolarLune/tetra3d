package tetra3d

import (
	"log"
	"time"

	"github.com/kvartborg/vector"
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

func (data *Data) AsVector() vector.Vector {
	return data.contents.(vector.Vector)
}

func (data *Data) AsQuaternion() *Quaternion {
	return data.contents.(*Quaternion)
}

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

type AnimationTrack struct {
	Type          string
	Keyframes     []*Keyframe
	Interpolation int
}

func (track *AnimationTrack) AddKeyframe(time float64, data interface{}) {
	track.Keyframes = append(track.Keyframes, newKeyframe(time, Data{data}))
}

func (track *AnimationTrack) ValueAsVector(time float64) vector.Vector {

	if len(track.Keyframes) == 0 {
		return nil
	}

	if first := track.Keyframes[0]; time <= first.Time {
		return first.Data.AsVector()
	} else if last := track.Keyframes[len(track.Keyframes)-1]; time >= last.Time {
		return last.Data.AsVector()
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
			return first.Data.AsVector()
		} else if time == last.Time {
			return last.Data.AsVector()
		} else {

			fd := first.Data.AsVector()
			ld := last.Data.AsVector()

			t := (time - first.Time) / (last.Time - first.Time)

			if track.Interpolation == InterpolationConstant {
				return fd
			} else {
				// We still need to implement InterpolationCubic
				if track.Type == TrackTypePosition || track.Type == TrackTypeScale {
					return fd.Add(ld.Sub(fd).Scale(t))
				}
			}

		}

	}

	return nil

}

func (track *AnimationTrack) ValueAsQuaternion(time float64) *Quaternion {

	if len(track.Keyframes) == 0 {
		return nil
	}

	if first := track.Keyframes[0]; time <= first.Time {
		return first.Data.AsQuaternion()
	} else if last := track.Keyframes[len(track.Keyframes)-1]; time >= last.Time {
		return last.Data.AsQuaternion()
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
			return first.Data.AsQuaternion()
		} else if time == last.Time {
			return last.Data.AsQuaternion()
		} else {

			fd := first.Data.AsQuaternion()
			ld := last.Data.AsQuaternion()

			t := (time - first.Time) / (last.Time - first.Time)

			return fd.Slerp(ld, t)

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

type Animation struct {
	Name     string
	Channels map[string]*AnimationChannel
	Length   float64 // Length of the animation in seconds
}

func NewAnimation(name string) *Animation {
	return &Animation{
		Name:     name,
		Channels: map[string]*AnimationChannel{},
	}
}

func (animation *Animation) AddChannel(name string) *AnimationChannel {
	newChannel := NewAnimationChannel(name)
	animation.Channels[name] = newChannel
	return newChannel
}

const (
	FinishModeLoop     = iota // Loop on animation completion
	FinishModePingPong        // Reverse on animation completion; if this is the case, the OnFinish() callback is called after two loops (one reversal)
	FinishModeStop            // Stop on animation completion
)

// AnimationValues indicate the current position, scale, and rotation for a Node.
type AnimationValues struct {
	Position vector.Vector
	Scale    vector.Vector
	Rotation *Quaternion
}

// AnimationPlayer is an object that allows you to play back an animation on a Node.
type AnimationPlayer struct {
	RootNode               INode
	ChannelsToNodes        map[*AnimationChannel]INode
	ChannelsUpdated        bool
	Animation              *Animation
	Playhead               float64                    // Playhead of the animation. Setting this to 0 restarts the animation.
	PlaySpeed              float64                    // Playback speed in percentage - defaults to 1 (100%)
	Playing                bool                       // Whether the player is playing back or not.
	FinishMode             int                        // What to do when the player finishes playback. Defaults to looping.
	OnFinish               func()                     // Callback indicating the Animation has completed
	AnimatedProperties     map[INode]*AnimationValues // The properties that have been animated
	PrevAnimatedProperties map[INode]*AnimationValues // The previous properties that have been animated from the previously Play()'d animation
	BlendTime              float64                    // How much time in seconds to blend between two animations
	BlendStart             time.Time                  // The time that the blend started
}

// NewAnimationPlayer returns a new AnimationPlayer for the Node.
func NewAnimationPlayer(node INode) *AnimationPlayer {
	return &AnimationPlayer{
		RootNode:               node,
		PlaySpeed:              1,
		FinishMode:             FinishModeLoop,
		AnimatedProperties:     map[INode]*AnimationValues{},
		PrevAnimatedProperties: map[INode]*AnimationValues{},
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
	return newAP
}

// SetRoot sets the root node of the animation player to act on. Note that this should be the root node.
func (ap *AnimationPlayer) SetRoot(node INode) {
	ap.RootNode = node
	ap.ChannelsUpdated = false
}

// Play plays the specified animation back. Calling Play() resets the playhead.
func (ap *AnimationPlayer) Play(animation *Animation) {

	ap.Playhead = 0.0
	ap.ChannelsUpdated = false

	if ap.Animation != animation || !ap.Playing {
		ap.Animation = animation
		ap.Playing = true
	}

	if ap.BlendTime > 0 {
		ap.PrevAnimatedProperties = map[INode]*AnimationValues{}
		for n, v := range ap.AnimatedProperties {
			ap.PrevAnimatedProperties[n] = v
		}
		ap.BlendStart = time.Now()
	}

}

// assignChannels assigns the player's root node's children to channels in the player. This is called when the channels need to be
// updated after the root node changes.
func (ap *AnimationPlayer) assignChannels() {

	if ap.Animation != nil {

		ap.AnimatedProperties = map[INode]*AnimationValues{}

		ap.ChannelsToNodes = map[*AnimationChannel]INode{}

		childrenRecursive := ap.RootNode.ChildrenRecursive(false)

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
						// node.SetLocalPosition(track.ValueAsVector(ap.Playhead))
						ap.AnimatedProperties[node].Position = track.ValueAsVector(ap.Playhead)
					}

					if track, exists := channel.Tracks[TrackTypeScale]; exists {
						// node.SetLocalScale(track.ValueAsVector(ap.Playhead))
						ap.AnimatedProperties[node].Scale = track.ValueAsVector(ap.Playhead)
					}

					if track, exists := channel.Tracks[TrackTypeRotation]; exists {
						quat := track.ValueAsQuaternion(ap.Playhead)
						// node.SetLocalRotation(NewMatrix4RotateFromQuaternion(quat))
						ap.AnimatedProperties[node].Rotation = quat
					}

				}

			}

			ap.Playhead += dt * ap.PlaySpeed

			if ap.FinishMode == FinishModeLoop && (ap.Playhead >= ap.Animation.Length || ap.Playhead < 0) {

				for ap.Playhead > ap.Animation.Length {
					ap.Playhead -= ap.Animation.Length
				}

				for ap.Playhead < 0 {
					ap.Playhead += ap.Animation.Length
				}

				if ap.OnFinish != nil {
					ap.OnFinish()
				}

			} else if ap.FinishMode == FinishModePingPong && (ap.Playhead > ap.Animation.Length || ap.Playhead < 0) {

				for ap.Playhead > ap.Animation.Length {
					ap.Playhead -= ap.Playhead - ap.Animation.Length
				}

				finishedLoop := false
				for ap.Playhead < 0 {
					ap.Playhead *= -1
					finishedLoop = true
				}

				if finishedLoop && ap.OnFinish != nil {
					ap.OnFinish()
				}

				ap.PlaySpeed *= -1

			} else if ap.FinishMode == FinishModeStop && ((ap.Playhead > ap.Animation.Length && ap.PlaySpeed > 0) || (ap.Playhead < 0 && ap.PlaySpeed < 0)) {

				if ap.Playhead > ap.Animation.Length {
					ap.Playhead = ap.Animation.Length
				} else if ap.Playhead < 0 {
					ap.Playhead = 0
				}

				if ap.OnFinish != nil {
					ap.OnFinish()
				}
				ap.Playing = false

			}

		}

	}

}

// Update updates the animation player by the delta specified in seconds (usually 1/FPS or 1/TARGET FPS), animating the transformation properties of the root node's tree.
func (ap *AnimationPlayer) Update(dt float64) {

	ap.updateValues(dt)

	if !ap.Playing && !ap.BlendStart.IsZero() {
		ap.BlendStart = time.Time{}
		ap.PrevAnimatedProperties = map[INode]*AnimationValues{}
	}

	for node, props := range ap.AnimatedProperties {

		_, prevExists := ap.PrevAnimatedProperties[node]

		if !ap.BlendStart.IsZero() && prevExists {

			bp := float64(time.Since(ap.BlendStart).Milliseconds()) / (ap.BlendTime * 1000)
			if bp > 1 {
				bp = 1
			}

			start := ap.PrevAnimatedProperties[node]

			if start.Position != nil && props.Position != nil {
				diff := props.Position.Sub(start.Position)
				node.SetLocalPosition(start.Position.Add(diff.Scale(bp)))
			} else if props.Position != nil {
				node.SetLocalPosition(props.Position)
			} else if start.Position != nil {
				node.SetLocalPosition(start.Position)
			}

			if start.Scale != nil && props.Scale != nil {
				diff := props.Scale.Sub(start.Scale)
				node.SetLocalScale(start.Scale.Add(diff.Scale(bp)))
			} else if props.Scale != nil {
				node.SetLocalScale(props.Scale)
			} else if start.Scale != nil {
				node.SetLocalScale(start.Scale)
			}

			if start.Rotation != nil && props.Rotation != nil {
				rot := start.Rotation.Slerp(props.Rotation, bp)
				node.SetLocalRotation(NewMatrix4RotateFromQuaternion(rot))
			} else if props.Rotation != nil {
				node.SetLocalRotation(NewMatrix4RotateFromQuaternion(props.Rotation))
			} else if start.Rotation != nil {
				node.SetLocalRotation(NewMatrix4RotateFromQuaternion(start.Rotation))
			}

			if bp == 1 {
				ap.BlendStart = time.Time{}
				ap.PrevAnimatedProperties = map[INode]*AnimationValues{}
			}

		} else {

			if props.Position != nil {
				node.SetLocalPosition(props.Position)
			}
			if props.Scale != nil {
				node.SetLocalScale(props.Scale)
			}
			if props.Rotation != nil {
				node.SetLocalRotation(NewMatrix4RotateFromQuaternion(props.Rotation))
			}

		}

	}

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
