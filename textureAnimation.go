package tetra3d

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// TextureAnimation is an animation struct. The TextureAnimation.Frames value is a []Vector, with each Vector representing
// a frame of the animation (and the offset from the original, base position for all animated vertices).
type TextureAnimation struct {
	FPS    float32   // The playback frame per second (or FPS) of the animation
	Frames []Vector2 // A slice of vectors, with each indicating the offset of the frame from the original position for the mesh.
}

// NewTextureAnimationPixels creates a new TextureAnimation using pixel positions instead of UV values. fps is the
// frames per second for the animation. image is the source texture used, and framePositions are the positions in pixels
// for each frame (i.e. 32, 32 instead of 0.25, 0.25 on a 128x128 spritesheet). NewTextureAnimationInPixels will panic
// if given less than 2 values for framePositions, or if it's an odd number of values (i.e. an X value for a frame,
// but no matching Y Value).
func NewTextureAnimationPixels(fps float32, image *ebiten.Image, framePositions ...float32) *TextureAnimation {

	if len(framePositions) < 2 || len(framePositions)%2 > 0 {
		panic("Error: NewTextureAnimation must take at least 2 frame values and has to be in pairs of X and Y values.")
	}

	textureWidth := float32(image.Bounds().Dx())
	textureHeight := float32(image.Bounds().Dy())

	frames := []Vector2{}

	for i := 0; i < len(framePositions); i += 2 {
		frames = append(frames, Vector2{
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
	originalOffsets map[int]Vector2   // OriginalOffsets is a map of vertex indices to their base UV offsets. All animating happens relative to these values.
	animation       *TextureAnimation // Animation is a pointer to the currently playing Animation.
	// playhead increases as the TexturePlayer plays. The integer portion of playhead is the frame that the TexturePlayer
	// resides in (so a playhead of 1.2 indicates that it is in frame 1, the second frame).
	playhead        float32
	Speed           float32 // Speed indicates the playback speed and direction of the TexturePlayer, with a value of 1.0 being 100%.
	playing         bool    // Playing indicates whether the TexturePlayer is currently playing or not.
	vertexSelection VertexSelection
}

// NewTexturePlayer returns a new TexturePlayer instance.
func NewTexturePlayer(vertexSelection VertexSelection) *TexturePlayer {
	player := &TexturePlayer{
		Speed:           1,
		vertexSelection: vertexSelection,
	}
	player.Reset(vertexSelection)
	return player
}

// Reset resets a TexturePlayer to be ready to run on a new selection of vertices. Note that this also resets the base UV offsets
// to use the current values of the passed vertices in the slice.
func (player *TexturePlayer) Reset(vertexSelection VertexSelection) {
	player.originalOffsets = map[int]Vector2{}
	vertexSelection.ForEachIndex(func(mesh *Mesh, index int) {
		player.originalOffsets[index] = mesh.VertexUVs[index]
	})
}

// Play plays the passed TextureAnimation, resetting the playhead if the TexturePlayer is not playing an animation. If the player is not playing, it will begin playing.
func (player *TexturePlayer) Play(animation *TextureAnimation) {
	if !player.playing || player.animation != animation {
		player.animation = animation
		player.playhead = 0
	}
	player.playing = true
}

// Returns the currently playing animation for the TexturePlayer.
func (player *TexturePlayer) Animation() *TextureAnimation {
	return player.animation
}

// Sets the currently playing animation for the TexturePlayer.
func (player *TexturePlayer) SetAnimation(anim *TextureAnimation) {
	player.animation = anim
}

// Update updates the TexturePlayer, using the passed delta time variable to animate the TexturePlayer's vertices.
func (player *TexturePlayer) Update(dt float32) {

	if player.animation != nil && player.playing && len(player.animation.Frames) > 0 {

		player.playhead += dt * player.animation.FPS * player.Speed

		playhead := int(player.playhead)
		for playhead >= len(player.animation.Frames) {
			playhead -= len(player.animation.Frames)
		}

		for playhead < 0 {
			playhead += len(player.animation.Frames)
		}

		frameOffset := player.animation.Frames[playhead]
		player.ApplyUVOffset(frameOffset.X, frameOffset.Y)

	}

}

// ApplyUVOffset applies a specified UV offset to all vertices a player is assigned to. This offset is not additive, but rather is
// set once, regardless of how many times ApplyUVOffset is called.
func (player *TexturePlayer) ApplyUVOffset(offsetX, offsetY float32) {
	for vertIndex, ogOffset := range player.originalOffsets {
		player.vertexSelection.mesh.VertexUVs[vertIndex].X = ogOffset.X + offsetX
		player.vertexSelection.mesh.VertexUVs[vertIndex].Y = ogOffset.Y + offsetY
	}
}

func (player *TexturePlayer) IsPlaying() bool {
	return player.playing
}

func (player *TexturePlayer) SetPlaying(isPlaying bool) {
	player.playing = isPlaying
}
