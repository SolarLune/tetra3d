package tetra3d

import (
	"github.com/kvartborg/vector"
)

// TextureAnimation is an animation struct. There is no function to create a new TextureAnimation because instantiating a struct is actually cleaner
// and easier to read than doing so through a function for this simple of a struct. The TextureAnimation.Frames value is a []vector.Vector, with each
// Vector representing a frame of the animation (and the offset from the original, base position for all animated vertices).
type TextureAnimation struct {
	FPS    float64         // The playback frame per second (or FPS) of the animation
	Frames []vector.Vector // A slice of vectors, with each indicating the offset of the frame from the original position for the mesh.
}

// TexturePlayer is a struct that allows you to animate a collection of vertices' UV values using a TextureAnimation.
type TexturePlayer struct {
	OriginalOffsets map[*Vertex]vector.Vector // OriginalOffsets is a map of vertices to their base UV offsets. All animating happens relative to these values.
	Animation       *TextureAnimation         // Animation is a pointer to the currently playing Animation.
	// Playhead increases as the TexturePlayer plays. The rounded down values is the frame that the TexturePlayer
	// resides in (so a Playhead of 1.2 indicates that it is in frame 1, the second frame).
	Playhead float64
	Speed    float64 // Speed indicates the playback speed and direction of the TexturePlayer, with a value of 1.0 being 100%.
	Playing  bool    // Playing indicates whether the TexturePlayer is currently playing or not.
}

// NewTexturePlayer returns a new TexturePlayer instance.
func NewTexturePlayer(verts []*Vertex) *TexturePlayer {
	player := &TexturePlayer{
		Speed: 1,
	}
	player.Reset(verts)
	return player
}

// Reset resets a TexturePlayer to be ready to run on a new selection of vertices. Note that this also resets the base UV offsets
// to use the current values of the passed vertices in the slice.
func (player *TexturePlayer) Reset(verts []*Vertex) {
	player.OriginalOffsets = map[*Vertex]vector.Vector{}
	for _, vert := range verts {
		player.OriginalOffsets[vert] = vert.UV.Clone()
	}
}

// Play plays the passed TextureAnimation, resetting the playhead if the TexturePlayer is not playing an animation. If the player is not playing, it will begin playing.
func (player *TexturePlayer) Play(animation *TextureAnimation) {
	if !player.Playing || player.Animation != animation {
		player.Animation = animation
		player.Playhead = 0
	}
	player.Playing = true
}

// Update updates the TexturePlayer, using the passed delta time variable to animate the TexturePlayer's vertices.
func (player *TexturePlayer) Update(dt float64) {

	if player.Animation != nil && player.Playing && len(player.Animation.Frames) > 0 {

		player.Playhead += dt * player.Animation.FPS * player.Speed

		playhead := int(player.Playhead)
		for playhead >= len(player.Animation.Frames) {
			playhead -= len(player.Animation.Frames)
		}

		for playhead < 0 {
			playhead += len(player.Animation.Frames)
		}

		frameOffset := player.Animation.Frames[playhead]
		player.ApplyUVOffset(frameOffset[0], frameOffset[1])

	}

}

// ApplyUVOffset applies a specified UV offset to all vertices a player is assigned to. This offset is not additive, but rather is
// set once, regardless of how many times ApplyUVOffset is called.
func (player *TexturePlayer) ApplyUVOffset(offsetX, offsetY float64) {
	for vert, ogOffset := range player.OriginalOffsets {
		vert.UV[0] = ogOffset[0] + offsetX
		vert.UV[1] = ogOffset[1] + offsetY
	}
}
