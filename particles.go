package tetra3d

import (
	"math/rand"
)

const (
	ParticleVertexSpawnModeOff        = iota // Particles spawn at the center of the system's root Model.
	ParticleVertexSpawnModeAscending         // Particles spawn at the vertices of the system's root Model. They spawn in ascending order before looping.
	ParticleVertexSpawnModeDescending        // Particles spawn at the vertices of the system's root Model. They spawn in descending order before looping.
	ParticleVertexSpawnModeRandom            // Particles spawn at the vertices of the system's root Model. They spawn in random order.
)

// Particle represents a particle, rendered in a ParticleSystem.
type Particle struct {
	ParticleSystem *ParticleSystem
	ModelBank      []*Model // Bank of models the particle could possibly use
	Model          *Model   // The currently active Model
	Velocity       Vector   // The velocity of the Particle in world-space
	Acceleration   Vector   // The acceleration of the Particle in world-space; these values are added to the particle's velocity each frame.
	Growth         Vector   // The growth of the Particle in world-space
	Rotation       Vector   // The additive rotation of the Particle in local-space
	Life           float64
	Lifetime       float64
	Data           map[string]interface{}
}

// NewParticle creates a new Particle for the given particle system, with the provided slice of particle factories to make particles from.
func NewParticle(partSystem *ParticleSystem, partModels []*Model) *Particle {

	bank := []*Model{}
	for _, p := range partModels {
		clone := p.Clone().(*Model)
		clone.visible = false
		// clone.FrustumCulling = false
		bank = append(bank, clone)
	}

	particle := &Particle{
		ParticleSystem: partSystem,
		ModelBank:      bank,
	}

	particle.Reinit()

	return particle
}

func (part *Particle) Reinit() {
	part.Model = part.ModelBank[rand.Intn(len(part.ModelBank))]
	part.Model.ClearLocalTransform()
}

// Update updates the particle's color and movement.
func (part *Particle) Update(dt float64) {

	part.Model.visible = part.ParticleSystem.Root.visible

	part.Life += dt

	if part.Acceleration.Magnitude() > 0 {
		part.Velocity = part.Velocity.Add(part.Acceleration)
	}

	vel := part.Velocity

	if friction := part.ParticleSystem.Settings.Friction; friction > 0 {

		if vel.X > friction {
			vel.X -= friction
		} else if vel.X < -friction {
			vel.X += friction
		} else {
			vel.X = 0
		}

		if vel.Y > friction {
			vel.Y -= friction
		} else if vel.Y < -friction {
			vel.Y += friction
		} else {
			vel.Y = 0
		}

		if vel.Z > friction {
			vel.Z -= friction
		} else if vel.Z < -friction {
			vel.Z += friction
		} else {
			vel.Z = 0
		}

	}

	if vel.Magnitude() != 0 {
		part.Model.MoveVec(vel)
	}

	if part.ParticleSystem.Settings.MovementFunction != nil {
		part.ParticleSystem.Settings.MovementFunction(part)
	}

	if part.Growth.Magnitude() != 0 {
		part.Model.GrowVec(part.Growth)
	}

	if part.Rotation.Magnitude() != 0 {
		part.Model.RotateVec(WorldRight, part.Rotation.X)
		part.Model.RotateVec(WorldUp, part.Rotation.Y)
		part.Model.RotateVec(WorldBack, part.Rotation.Z)
	}

	scale := part.Model.LocalScale()

	if !part.ParticleSystem.Settings.AllowNegativeScale && (scale.X < 0 || scale.Y < 0 || scale.Z < 0) {
		part.Life = part.Lifetime // Die because the particle got too small
	}

	if part.Life >= part.Lifetime {
		part.Model.visible = false
		part.ParticleSystem.Remove(part)
	}

	if curve := part.ParticleSystem.Settings.ColorCurve; curve != nil && len(curve.Points) > 0 {
		part.Model.Color = curve.Color(part.Life / part.Lifetime)
	}

}

type ParticleSystemSettings struct {
	SpawnRate      float64      // SpawnRate is how often a particle is spawned
	SpawnCount     int          // SpawnCount is how many particles are spawned at a time when a particle is spawned
	Lifetime       *NumberRange // Lifetime is how long a particle lives in seconds
	ParentToSystem bool         // If particles should be parented to the owning particle system

	Velocity           *VectorRange
	Acceleration       *VectorRange
	SpawnOffset        *VectorRange
	Scale              *VectorRange
	Growth             *VectorRange
	Rotation           *VectorRange
	Friction           float64 // Friction to apply to velocity
	AllowNegativeScale bool    // If negative scale should be allowed for particles. By default, this is false.

	VertexSpawnMode int // VertexSpawnMode influences where a particle spawns. By default, this is ParticleVertexSpawnModeOff.

	// SpawnOffsetFunction is a function the user can use to customize spawning position of the particles within the system. This function
	// is called additively to the SpawnOffset setting.
	SpawnOffsetFunction func(particle *Particle)

	// MovementFunction is a function the user can use to customize movement of the particles within the system. This function
	// is called additively to the other movement settings.
	MovementFunction func(particle *Particle)

	ColorCurve *ColorCurve // ColorCurve is a curve indicating how the spawned particles should change color as they live.
}

// NewParticleSystemSettings creates a new particle system settings.
func NewParticleSystemSettings() *ParticleSystemSettings {

	scale := NewVectorRange()
	scale.SetRanges(1, 1)

	lifetime := NewNumberRange()
	lifetime.Set(1, 1)

	return &ParticleSystemSettings{
		SpawnRate:  1,
		SpawnCount: 1,
		Lifetime:   lifetime,

		Velocity:     NewVectorRange(),
		SpawnOffset:  NewVectorRange(),
		Scale:        scale,
		Growth:       NewVectorRange(),
		Acceleration: NewVectorRange(),
		Rotation:     NewVectorRange(),

		ColorCurve: NewColorCurve(),
	}
}

// Clone duplicates the ParticleSystemSettings.
func (pss *ParticleSystemSettings) Clone() *ParticleSystemSettings {

	newPS := &ParticleSystemSettings{
		SpawnRate:  pss.SpawnRate,
		SpawnCount: pss.SpawnCount,
		Lifetime:   pss.Lifetime.Clone(),

		Velocity:     pss.Velocity.Clone(),
		Acceleration: pss.Acceleration.Clone(),
		Scale:        pss.Scale.Clone(),
		Growth:       pss.Growth.Clone(),
		SpawnOffset:  pss.SpawnOffset.Clone(),
		Rotation:     pss.Rotation.Clone(),
		Friction:     pss.Friction,

		ColorCurve:      pss.ColorCurve.Clone(),
		VertexSpawnMode: pss.VertexSpawnMode,

		MovementFunction:    pss.MovementFunction,
		SpawnOffsetFunction: pss.SpawnOffsetFunction,
	}

	return newPS

}

// ParticleSystem represents a collection of particles.
type ParticleSystem struct {
	LivingParticles []*Particle
	ToRemove        []*Particle
	DeadParticles   []*Particle

	ParticleFactories []*Model
	Root              *Model

	SpawnTimer       float64
	Settings         *ParticleSystemSettings
	vertexSpawnIndex int
}

// NewParticleSystem creates a new ParticleSystem, operating on the systemNode and randomly creating particles from the provided collection of particle Models.
func NewParticleSystem(systemNode *Model, particles ...*Model) *ParticleSystem {

	for _, part := range particles {
		mat := part.Mesh.MeshParts[0].Material
		if systemNode.Mesh.FindMeshPart(mat.Name) == nil {
			systemNode.Mesh.AddMeshPart(part.Mesh.MeshParts[0].Material)
		}
	}

	// systemNode.FrustumCulling = false // if we leave frustum culling on, the particles will turn invisible if the batch goes offscreen

	return &ParticleSystem{
		ParticleFactories: particles,
		Root:              systemNode,

		LivingParticles: []*Particle{},
		DeadParticles:   []*Particle{},
		ToRemove:        []*Particle{},

		Settings: NewParticleSystemSettings(),
	}

}

// Clone creates a duplicate of the given ParticleSystem.
func (ps *ParticleSystem) Clone() *ParticleSystem {

	newPS := NewParticleSystem(ps.Root, ps.ParticleFactories...)
	newPS.Settings = ps.Settings.Clone()
	return newPS

}

// Update should be called once per tick.
func (ps *ParticleSystem) Update(dt float64) {

	if ps.SpawnTimer <= 0 {
		for i := 0; i < ps.Settings.SpawnCount; i++ {
			ps.Spawn()
		}
		ps.SpawnTimer = ps.Settings.SpawnRate
	}

	ps.SpawnTimer -= dt

	for _, part := range ps.LivingParticles {
		part.Update(dt)
	}

	for _, toRemove := range ps.ToRemove {
		for i, part := range ps.LivingParticles {
			if part == toRemove {
				ps.LivingParticles[i] = nil
				ps.LivingParticles = append(ps.LivingParticles[:i], ps.LivingParticles[i+1:]...)
				ps.DeadParticles = append(ps.DeadParticles, part)
				break
			}
		}
	}

	ps.ToRemove = []*Particle{}

}

// Spawn spawns exactly one particle when called.
func (ps *ParticleSystem) Spawn() {

	var part *Particle
	if len(ps.DeadParticles) > 0 {
		part = ps.DeadParticles[0]
		ps.DeadParticles[0] = nil
		ps.DeadParticles = ps.DeadParticles[1:]
	} else {
		part = NewParticle(ps, ps.ParticleFactories)
		for _, newModel := range part.ModelBank {
			ps.Root.DynamicBatchAdd(ps.Root.Mesh.FindMeshPart(part.Model.Mesh.MeshParts[0].Material.Name), newModel)
		}
	}

	ps.LivingParticles = append(ps.LivingParticles, part)

	part.Life = 0
	part.Lifetime = ps.Settings.Lifetime.Value()

	part.Reinit()

	if ps.Settings.ParentToSystem {
		ps.Root.AddChildren(part.Model)
	}

	part.Model.SetWorldScaleVec(ps.Settings.Scale.Value())

	part.Velocity = ps.Settings.Velocity.Value()
	part.Acceleration = ps.Settings.Acceleration.Value()
	part.Growth = ps.Settings.Growth.Value()
	part.Rotation = ps.Settings.Rotation.Value()

	var pos Vector

	if ps.Settings.VertexSpawnMode != ParticleVertexSpawnModeOff {

		vertCount := len(ps.Root.Mesh.VertexPositions)

		if ps.Root.skinned {
			pos = ps.Root.Mesh.vertexSkinnedPositions[ps.vertexSpawnIndex]
		} else {
			pos = ps.Root.Transform().MultVec(ps.Root.Mesh.VertexPositions[ps.vertexSpawnIndex])
		}

		switch ps.Settings.VertexSpawnMode {
		case ParticleVertexSpawnModeAscending:
			ps.vertexSpawnIndex++
		case ParticleVertexSpawnModeDescending:
			ps.vertexSpawnIndex--
		case ParticleVertexSpawnModeRandom:
			ps.vertexSpawnIndex = rand.Intn(vertCount)

		}

		if ps.vertexSpawnIndex < 0 {
			ps.vertexSpawnIndex = vertCount - 1
		} else if ps.vertexSpawnIndex >= vertCount {
			ps.vertexSpawnIndex = 0
		}

	} else {
		pos = ps.Root.WorldPosition()
	}

	part.Model.SetWorldPositionVec(pos.Add(ps.Settings.SpawnOffset.Value()))

	if ps.Settings.SpawnOffsetFunction != nil {
		ps.Settings.SpawnOffsetFunction(part)
	}

}

// Remove removes a particle from the ParticleSystem, recycling the Particle for the next time a particle is spawned.
func (ps *ParticleSystem) Remove(part *Particle) {
	ps.ToRemove = append(ps.ToRemove, part)
}

type NumberRange struct {
	Min, Max float64
}

func NewNumberRange() *NumberRange {
	return &NumberRange{}
}

func (ran *NumberRange) Clone() *NumberRange {
	return &NumberRange{
		Min: ran.Min,
		Max: ran.Max,
	}
}

func (ran *NumberRange) Set(min, max float64) {
	ran.Min = min
	ran.Max = max
}

func (ran *NumberRange) Value() float64 {
	random := rand.Float64()
	return ran.Min + ((ran.Max - ran.Min) * random)
}

// VectorRange represents a range of possible values, and allows Tetra3D to get a random value from within
// that number range.
type VectorRange struct {
	Uniform bool   // If the random value returned by the NumberRange should be consistent across all axes or not
	Min     Vector // Min is the set of minimum numbers allowed in the NumberRange
	Max     Vector // Max is the set of maximum numbers allowed in the NumberRange
}

// NewVectorRange returns a new instance of a 3D NumberRange struct.
func NewVectorRange() *VectorRange {
	return &VectorRange{
		Min: Vector{0, 0, 0, 0},
		Max: Vector{0, 0, 0, 0},
	}
}

// Clone returns a clone of the NumberRange.
func (ran *VectorRange) Clone() *VectorRange {
	new := NewVectorRange()
	new.Uniform = ran.Uniform
	new.Min = ran.Min
	new.Max = ran.Max
	return new
}

// SetAll sets the minimum and maximum values of all components of the number range at the same time to the value
// passed.
func (ran *VectorRange) SetAll(value float64) {
	ran.Min.X = value
	ran.Max.X = value
	ran.Min.Y = value
	ran.Max.Y = value
	ran.Min.Z = value
	ran.Max.Z = value
}

// SetAxes sets the minimum and maximum values of all components of the number range at the same time. Both
// minimum and maximum boundaries of the NumberRange will be the same.
func (ran *VectorRange) SetAxes(x, y, z float64) {
	ran.Min.X = x
	ran.Max.X = x

	ran.Min.Y = y
	ran.Max.Y = y

	ran.Min.Z = z
	ran.Max.Z = z
}

// SetRanges sets the minimum and maximum values of all components (axes) of the number range.
func (ran *VectorRange) SetRanges(min, max float64) {

	ran.Min.X = min
	ran.Min.Y = min
	ran.Min.Z = min

	ran.Max.X = max
	ran.Max.Y = max
	ran.Max.Z = max

}

// SetRangeX sets the minimum and maximum values of the X component of the number range.
func (ran *VectorRange) SetRangeX(min, max float64) {
	ran.Min.X = min
	ran.Max.X = max
}

// SetRangeY sets the minimum and maximum values of the Y component of the number range.
func (ran *VectorRange) SetRangeY(min, max float64) {
	ran.Min.Y = min
	ran.Max.Y = max
}

// SetRangeZ sets the minimum and maximum values of the Z component of the number range.
func (ran *VectorRange) SetRangeZ(min, max float64) {
	ran.Min.Z = min
	ran.Max.Z = max
}

// Value returns a random value from within the bounds of the NumberRange.
func (ran *VectorRange) Value() Vector {

	var vec Vector

	if ran.Uniform {
		random := rand.Float64()
		vec = Vector{
			ran.Min.X + ((ran.Max.X - ran.Min.X) * random),
			ran.Min.Y + ((ran.Max.Y - ran.Min.Y) * random),
			ran.Min.Z + ((ran.Max.Z - ran.Min.Z) * random),
			0,
		}
	} else {
		vec = Vector{
			ran.Min.X + ((ran.Max.X - ran.Min.X) * rand.Float64()),
			ran.Min.Y + ((ran.Max.Y - ran.Min.Y) * rand.Float64()),
			ran.Min.Z + ((ran.Max.Z - ran.Min.Z) * rand.Float64()),
			0,
		}
	}

	return vec
}
