package tetra3d

import (
	"log"

	"github.com/kvartborg/vector"
)

const (
	TrackTypePosition = "Pos"
	TrackTypeScale    = "Sca"
	TrackTypeRotation = "Rot"
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
	Type      string
	Keyframes []*Keyframe
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

			if track.Type == TrackTypePosition || track.Type == TrackTypeScale {
				return fd.Add(ld.Sub(fd).Scale(t))
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
	FinishModeLoop = iota
	FinishModePingPong
	FinishModeStop
)

type AnimationPlayer struct {
	RootNode        Node
	ChannelsToNodes map[*AnimationChannel]Node
	ChannelsUpdated bool
	Animation       *Animation
	Playhead        float64
	PlaySpeed       float64
	Playing         bool
	FinishMode      int
	OnFinish        func()
}

func NewAnimationPlayer(node Node) *AnimationPlayer {
	return &AnimationPlayer{
		RootNode:   node,
		PlaySpeed:  1,
		FinishMode: FinishModeStop,
	}
}

func (ap *AnimationPlayer) Clone() *AnimationPlayer {
	newAP := NewAnimationPlayer(ap.RootNode)

	newAP.ChannelsToNodes = map[*AnimationChannel]Node{}
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

func (ap *AnimationPlayer) SetRoot(node Node) {
	ap.RootNode = node
	ap.ChannelsUpdated = false
}

func (ap *AnimationPlayer) Play(animation *Animation) {

	if ap.Animation != animation || !ap.Playing {
		ap.Animation = animation
		ap.Playhead = 0.0
		ap.Playing = true
	}

}

func (ap *AnimationPlayer) AssignChannels() {

	if ap.Animation != nil {

		ap.ChannelsToNodes = map[*AnimationChannel]Node{}

		childrenRecursive := ap.RootNode.ChildrenRecursive(false)

		for _, channel := range ap.Animation.Channels {

			if ap.RootNode.Name() == channel.Name {
				ap.ChannelsToNodes[channel] = ap.RootNode
				continue
			}

			for _, n := range childrenRecursive {

				if n.Name() == channel.Name {
					ap.ChannelsToNodes[channel] = n
					continue
				}

			}

		}

	}

	ap.ChannelsUpdated = true

}

func (ap *AnimationPlayer) Update(dt float64) {

	if ap.Playing {

		if ap.Animation != nil {

			if !ap.ChannelsUpdated {
				ap.AssignChannels()
			}

			for _, channel := range ap.Animation.Channels {

				node := ap.ChannelsToNodes[channel]

				if node == nil {
					log.Println("Error: Cannot find matching node for channel " + channel.Name + " for root " + ap.RootNode.Name())
				}

				if track, exists := channel.Tracks[TrackTypePosition]; exists {
					node.SetLocalPosition(track.ValueAsVector(ap.Playhead))
				}

				if track, exists := channel.Tracks[TrackTypeScale]; exists {
					node.SetLocalScale(track.ValueAsVector(ap.Playhead))
				}

				if track, exists := channel.Tracks[TrackTypeRotation]; exists {
					quat := track.ValueAsQuaternion(ap.Playhead)
					node.SetLocalRotation(NewMatrix4RotateFromQuaternion(quat))
				}

			}

			ap.Playhead += dt * ap.PlaySpeed

			if ap.FinishMode == FinishModeLoop && (ap.Playhead >= ap.Animation.Length || ap.Playhead < 0) {

				if ap.Playhead > ap.Animation.Length {
					ap.Playhead = ap.Animation.Length
				} else if ap.Playhead < 0 {
					ap.Playhead = 0
				}

				if ap.OnFinish != nil {
					ap.OnFinish()
				}

			} else if ap.FinishMode == FinishModePingPong && (ap.Playhead > ap.Animation.Length || ap.Playhead < 0) {

				if ap.Playhead > ap.Animation.Length {
					ap.Playhead = ap.Animation.Length
				} else {
					ap.Playhead = 0
					if ap.OnFinish != nil {
						ap.OnFinish()
					}
				}

				ap.PlaySpeed *= -1

			} else if ap.FinishMode == FinishModeStop && (ap.Playhead > ap.Animation.Length || ap.Playhead < 0) {

				if ap.OnFinish != nil {
					ap.OnFinish()
				}
				ap.Playing = false

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
