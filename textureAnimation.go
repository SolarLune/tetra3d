package tetra3d

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/quartercastle/vector"
)

// TextureAnimation is an animation struct. The TextureAnimation.Frames value is a []vector.Vector, with each Vector representing
// a frame of the animation (and the offset from the original, base position for all animated vertices).
type TextureAnimation struct {
	FPS    float64         // The playback frame per second (or FPS) of the animation
	Frames []vector.Vector // A slice of vectors, with each indicating the offset of the frame from the original position for the mesh.
}

// NewTextureAnimationPixels creates a new TextureAnimation using pixel positions instead of UV values. fps is the
// frames per second for the animation. image is the source texture used, and framePositions are the positions in pixels
// for each frame (i.e. 32, 32 instead of 0.25, 0.25 on a 128x128 spritesheet). NewTextureAnimationInPixels will panic
// if given less than 2 values for framePositions, or if it's an odd number of values (i.e. an X value for a frame,
// but no matching Y Value).
func NewTextureAnimationPixels(fps float64, image *ebiten.Image, framePositions ...float64) *TextureAnimation {

	if len(framePositions) < 2 || len(framePositions)%2 > 0 {
		panic("Error: NewTextureAnimation must take at least 2 frame values and has to be in pairs of X and Y values.")
	}

	textureWidth := float64(image.Bounds().Dx())
	textureHeight := float64(image.Bounds().Dy())

	frames := []vector.Vector{}

	for i := 0; i < len(framePositions); i += 2 {
		frames = append(frames, vector.Vector{
			framePositions[i] / textureWidth,
			framePositions[i+1] / textureHeight,
		})
	}

	return &TextureAnimation{
		FPS:    fps,
		Frames: frames,
	}

}

// TexturePlayer is a struct that allows you to animate a collection of vertices' UV values using a TextureAnimation.
type TexturePlayer struct {
	OriginalOffsets map[int]vector.Vector // OriginalOffsets is a map of vertex indices to their base UV offsets. All animating happens relative to these values.
	Animation       *TextureAnimation     // Animation is a pointer to the currently playing Animation.
	// Playhead increases as the TexturePlayer plays. The integer portion of Playhead is the frame that the TexturePlayer
	// resides in (so a Playhead of 1.2 indicates that it is in frame 1, the second frame).
	Playhead float64
	Speed    float64 // Speed indicates the playback speed and direction of the TexturePlayer, with a value of 1.0 being 100%.
	Playing  bool    // Playing indicates whether the TexturePlayer is currently playing or not.

	Mesh *Mesh
}

// NewTexturePlayer returns a new TexturePlayer instance.
func NewTexturePlayer(mesh *Mesh, vertexSelection *VertexSelection) *TexturePlayer {
	player := &TexturePlayer{
		Mesh:  mesh,
		Speed: 1,
	}
	player.Reset(vertexSelection)
	return player
}

// Reset resets a TexturePlayer to be ready to run on a new selection of vertices. Note that this also resets the base UV offsets
// to use the current values of the passed vertices in the slice.
func (player *TexturePlayer) Reset(vertexSelection *VertexSelection) {
	player.OriginalOffsets = map[int]vector.Vector{}
	for vert := range vertexSelection.Indices {
		player.OriginalOffsets[vert] = player.Mesh.VertexUVs[vert].Clone()
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
	for vertIndex, ogOffset := range player.OriginalOffsets {
		player.Mesh.VertexUVs[vertIndex][0] = ogOffset[0] + offsetX
		player.Mesh.VertexUVs[vertIndex][1] = ogOffset[1] + offsetY
	}
}
