package tetra3d

import (
	"math/rand"

	"github.com/solarlune/tetra3d/math32"
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

	Velocity Vector3 // The constant velocity of the Particle in world-space

	VelocityAdd Vector3 // The acceleration of the Particle in world-space; these values are added to the particle's velocity each frame.
	ScaleAdd    Vector3 // The growth of the Particle in world-space
	RotationAdd Vector3 // The additive rotation of the Particle in local-space

	Life     float32        // How long the particle has left to live
	Lifetime float32        // How long the particle lives, maximum
	Data     map[string]any // A custom Data map for storing and retrieving data
}

// NewParticle creates a new Particle for the given particle system, with the provided slice of particle factories to make particles from.
func NewParticle(partSystem *ParticleSystem, partModels []*Model) *Particle {

	bank := []*Model{}
	for _, p := range partModels {
		clone := p.Clone().(*Model)
		clone.visible = false
		// clone.FrustumCulling = false
		// clone.AutoBatchMode = AutoBatchDynamic
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
	part.Life = 0
}

// Update updates the particle's color and movement.
func (part *Particle) Update(dt float32) {

	part.Model.visible = true

	part.Life += dt

	if !part.VelocityAdd.IsZero() {
		part.Velocity = part.Velocity.Add(part.VelocityAdd)
	}

	if !part.Velocity.IsZero() {

		if friction := part.ParticleSystem.Settings.Friction; friction > 0 {
			part.Velocity = part.Velocity.SubMagnitude(friction)
		}

		part.Model.MoveVec(part.Velocity)

	}

	if part.ParticleSystem.Settings.MovementFunction != nil {
		part.ParticleSystem.Settings.MovementFunction(part)
	}

	if !part.ScaleAdd.IsZero() {
		part.Model.GrowVec(part.ScaleAdd)
	}

	if !part.RotationAdd.IsZero() {
		part.Model.RotateVec(WorldRight, part.RotationAdd.X)
		part.Model.RotateVec(WorldUp, part.RotationAdd.Y)
		part.Model.RotateVec(WorldBackward, part.RotationAdd.Z)
	}

	scale := part.Model.LocalScale()

	if !part.ParticleSystem.Settings.AllowNegativeScale && (scale.X < 0 || scale.Y < 0 || scale.Z < 0) {
		part.Life = part.Lifetime // Die because the particle got too small
	}

	if part.Life >= part.Lifetime {
		part.Model.visible = false
		part.ParticleSystem.Remove(part)
	}

	if curve := part.ParticleSystem.Settings.ColorCurve; len(curve.Points) > 0 {
		part.Model.Color = curve.Color(part.Life / part.Lifetime)
	}

}

type ParticleSystemSettings struct {
	SpawnOn     bool        // If the particle system should spawn particles at all
	SpawnRate   FloatRange  // SpawnRate is how often a particle is spawned in seconds
	SpawnCount  IntRange    // SpawnCount is how many particles are spawned at a time when a particle is spawned
	Lifetime    FloatRange  // Lifetime is how long a particle lives in seconds
	SpawnOffset VectorRange // The range indicating how far of an offset to move

	Velocity    VectorRange // The range indicating how fast a particle constantly moves per frame
	VelocityAdd VectorRange // The range indicating how fast a particle accelerates per frame

	Scale    VectorRange // The range indicating how large the particle should spawn in as
	ScaleAdd VectorRange // The range indicating how large the particle should grow per frame

	RotationAdd        VectorRange // The range indicating how fast a particle should spin per frame
	LocalPosition      bool        // Whether the particles' positions should be local to the system or not; defaults to false.
	Friction           float32     // Friction to apply to velocity
	AllowNegativeScale bool        // If negative scale should be allowed for particles. By default, this is false.

	VertexSpawnMode  int // VertexSpawnMode influences where a particle spawns. By default, this is ParticleVertexSpawnModeOff.
	VertexSpawnModel *Model

	// SpawnOffsetFunction is a function the user can use to customize spawning position of the particles within the system. This function
	// is called additively to the SpawnOffset setting.
	SpawnOffsetFunction func(particle *Particle)

	// MovementFunction is a function the user can use to customize movement of the particles within the system. This function
	// is called additively to the other movement settings.
	MovementFunction func(particle *Particle)

	// Todo: Add curves for all features?
	ColorCurve ColorCurve // ColorCurve is a curve indicating how the spawned particles should change color as they live.
}

// NewParticleSystemSettings creates a new particle system settings.
func NewParticleSystemSettings() *ParticleSystemSettings {

	scale := NewVectorRange()
	scale.SetRanges(1, 1)

	lifetime := NewFloatRange()
	lifetime.Set(1, 1)

	spawnRate := NewFloatRange()
	spawnRate.Set(1, 1)

	spawnCount := NewIntRange()
	spawnCount.Set(1, 1)

	return &ParticleSystemSettings{
		SpawnOn:    true,
		SpawnRate:  spawnRate,
		SpawnCount: spawnCount,
		Lifetime:   lifetime,

		Velocity:    NewVectorRange(),
		SpawnOffset: NewVectorRange(),
		Scale:       scale,
		ScaleAdd:    NewVectorRange(),
		VelocityAdd: NewVectorRange(),
		RotationAdd: NewVectorRange(),

		ColorCurve: NewColorCurve(),
	}
}

// Clone duplicates the ParticleSystemSettings.
func (pss *ParticleSystemSettings) Clone() *ParticleSystemSettings {

	newPS := &ParticleSystemSettings{
		SpawnOn: pss.SpawnOn,

		SpawnRate:  pss.SpawnRate,
		SpawnCount: pss.SpawnCount,
		Lifetime:   pss.Lifetime,

		Velocity:    pss.Velocity,
		VelocityAdd: pss.VelocityAdd,
		Scale:       pss.Scale,
		ScaleAdd:    pss.ScaleAdd,
		SpawnOffset: pss.SpawnOffset,
		RotationAdd: pss.RotationAdd,
		Friction:    pss.Friction,

		ColorCurve:      pss.ColorCurve,
		VertexSpawnMode: pss.VertexSpawnMode,

		MovementFunction:    pss.MovementFunction,
		SpawnOffsetFunction: pss.SpawnOffsetFunction,

		LocalPosition:      pss.LocalPosition,
		AllowNegativeScale: pss.AllowNegativeScale,
		VertexSpawnModel:   pss.VertexSpawnModel,
	}

	return newPS

}

// ParticleSystem represents a collection of particles.
type ParticleSystem struct {
	LivingParticles []*Particle
	toRemove        []*Particle
	DeadParticles   []*Particle
	On              bool

	ParticleFactories []*Model
	Root              *Model

	spawnTimer       float32
	Settings         *ParticleSystemSettings
	vertexSpawnIndex int
}

// NewParticleSystem creates a new ParticleSystem, operating on the baseModel Model and
// randomly creating particles from the provided collection of particle Models.
func NewParticleSystem(baseModel *Model, particles ...*Model) *ParticleSystem {

	for _, part := range particles {
		mat := part.Mesh.MeshParts[0].Material
		if baseModel.Mesh.MeshPartByMaterialName(mat.name) == nil {
			baseModel.Mesh.AddMeshPart(part.Mesh.MeshParts[0].Material)
		}
	}

	// baseModel.FrustumCulling = false // if we leave frustum culling on, the particles will turn invisible if the batch goes offscreen

	// for _, p := range particles {
	// 	p.AutoBatchMode = AutoBatchDynamic
	// }

	partSys := &ParticleSystem{
		ParticleFactories: particles,
		Root:              baseModel,

		LivingParticles: []*Particle{},
		DeadParticles:   []*Particle{},
		toRemove:        []*Particle{},

		Settings: NewParticleSystemSettings(),

		On: true,
	}

	// We calculate the frustum sphere based off of the particles spawned
	partSys.Root.updateFrustumSphere = false

	// partSys.Root.SetVisible(false, false)

	return partSys

}

// Clone creates a duplicate of the given ParticleSystem.
func (ps *ParticleSystem) Clone() *ParticleSystem {

	newPS := NewParticleSystem(ps.Root, ps.ParticleFactories...)
	newPS.Settings = ps.Settings
	return newPS

}

// Update should be called once per tick.
func (ps *ParticleSystem) Update(dt float32) {

	furthestDist := float32(0.0)
	largestParticle := float32(0.0)

	for _, part := range ps.LivingParticles {
		part.Update(dt)
		furthestDist = math32.Max(furthestDist, ps.Root.DistanceSquaredTo(part.Model))
		largestParticle = math32.Max(largestParticle, part.Model.Mesh.Dimensions.MaxSpan()*part.Model.scale.Magnitude())
	}

	ps.Root.frustumCullingSphere.position = ps.Root.WorldPosition()
	ps.Root.frustumCullingSphere.Radius = (math32.Sqrt(furthestDist)) + largestParticle
	ps.Root.frustumCullingSphere.scale = ps.Root.WorldScale()
	// ps.Root.FrustumCulling = false
	// Rotation doesn't matter

	for _, toRemove := range ps.toRemove {
		for i, part := range ps.LivingParticles {
			if part == toRemove {
				ps.LivingParticles[i] = nil
				ps.LivingParticles = append(ps.LivingParticles[:i], ps.LivingParticles[i+1:]...)
				ps.DeadParticles = append(ps.DeadParticles, part)
				part.Model.Unparent()
				break
			}
		}
	}

	ps.toRemove = ps.toRemove[:0]

	if !ps.On {
		return
	}

	if ps.Settings.SpawnOn {

		if ps.spawnTimer <= 0 {
			spawnCount := int(ps.Settings.SpawnCount.Value())
			for i := 0; i < spawnCount; i++ {
				ps.Spawn()
			}
			ps.spawnTimer = ps.Settings.SpawnRate.Value()
		}

		ps.spawnTimer -= dt
	}

	// if len(ps.Root.DynamicBatchModels) > 0 {
	// 	ps.Root.SetVisible(true, true)
	// }

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
		for _, model := range part.ModelBank {
			ps.Root.DynamicBatchAdd(model.Mesh.MeshParts[0], model)
		}
		// for _, newModel := range part.ModelBank {
		// 	ps.Root.DynamicBatchAdd(ps.Root.Mesh.FindMeshPart(part.Model.Mesh.MeshParts[0].Material.Name), newModel)
		// }
	}

	ps.LivingParticles = append(ps.LivingParticles, part)

	part.Lifetime = ps.Settings.Lifetime.Value()

	part.Reinit()

	if ps.Settings.LocalPosition {
		ps.Root.AddChildren(part.Model)
	} else {
		ps.Root.Root().AddChildren(part.Model)
	}

	part.Model.SetWorldScaleVec(ps.Settings.Scale.Value())

	part.Velocity = ps.Settings.Velocity.Value()
	part.VelocityAdd = ps.Settings.VelocityAdd.Value()
	part.ScaleAdd = ps.Settings.ScaleAdd.Value()
	part.RotationAdd = ps.Settings.RotationAdd.Value()

	var pos Vector3

	if ps.Settings.VertexSpawnMode != ParticleVertexSpawnModeOff && ps.Settings.VertexSpawnModel != nil {

		model := ps.Settings.VertexSpawnModel

		vertCount := len(model.Mesh.VertexPositions)

		if model.skinned {
			pos = model.Mesh.vertexSkinnedPositions[ps.vertexSpawnIndex]
		} else {
			pos = model.Transform().MultVec(model.Mesh.VertexPositions[ps.vertexSpawnIndex])
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
	ps.toRemove = append(ps.toRemove, part)
}

type FloatRange struct {
	Min, Max float32
}

func NewFloatRange() FloatRange {
	return FloatRange{}
}

func (ran *FloatRange) Set(min, max float32) {
	ran.Min = min
	ran.Max = max
}

func (ran FloatRange) Value() float32 {
	random := rand.Float32()
	return ran.Min + ((ran.Max - ran.Min) * random)
}

type IntRange struct {
	Min, Max int
}

func NewIntRange() IntRange {
	return IntRange{}
}

func (ran *IntRange) Set(min, max int) {
	ran.Min = min
	ran.Max = max
}

func (ran IntRange) Value() int {
	if ran.Min >= ran.Max {
		return ran.Min
	}
	return ran.Min + rand.Intn(ran.Max-ran.Min)
}

// VectorRange represents a range of possible values, and allows Tetra3D to get a random value from within
// that number range.
type VectorRange struct {
	Uniform bool    // If the random value returned by the NumberRange should be consistent across all axes or not
	Min     Vector3 // Min is the set of minimum numbers allowed in the NumberRange
	Max     Vector3 // Max is the set of maximum numbers allowed in the NumberRange
}

// NewVectorRange returns a new instance of a 3D NumberRange struct.
func NewVectorRange() VectorRange {
	return VectorRange{}
}

// SetAll sets the minimum and maximum values of all components of the number range at the same time to the value
// passed.
func (ran *VectorRange) SetAll(value float32) {
	ran.Min.X = value
	ran.Max.X = value
	ran.Min.Y = value
	ran.Max.Y = value
	ran.Min.Z = value
	ran.Max.Z = value
}

// SetAxes sets the minimum and maximum values of all components of the number range at the same time. Both
// minimum and maximum boundaries of the NumberRange will be the same.
func (ran *VectorRange) SetAxes(x, y, z float32) {
	ran.Min.X = x
	ran.Max.X = x

	ran.Min.Y = y
	ran.Max.Y = y

	ran.Min.Z = z
	ran.Max.Z = z
}

// SetRanges sets the minimum and maximum values of all components (axes) of the number range.
func (ran *VectorRange) SetRanges(min, max float32) {

	ran.Min.X = min
	ran.Min.Y = min
	ran.Min.Z = min

	ran.Max.X = max
	ran.Max.Y = max
	ran.Max.Z = max

}

// SetRangeX sets the minimum and maximum values of the X component of the number range.
func (ran *VectorRange) SetRangeX(min, max float32) {
	ran.Min.X = min
	ran.Max.X = max
}

// SetRangeY sets the minimum and maximum values of the Y component of the number range.
func (ran *VectorRange) SetRangeY(min, max float32) {
	ran.Min.Y = min
	ran.Max.Y = max
}

// SetRangeZ sets the minimum and maximum values of the Z component of the number range.
func (ran *VectorRange) SetRangeZ(min, max float32) {
	ran.Min.Z = min
	ran.Max.Z = max
}

// Value returns a random value from within the bounds of the NumberRange.
func (ran VectorRange) Value() Vector3 {

	var vec Vector3

	if ran.Uniform {
		random := rand.Float32()
		vec = Vector3{
			ran.Min.X + ((ran.Max.X - ran.Min.X) * random),
			ran.Min.Y + ((ran.Max.Y - ran.Min.Y) * random),
			ran.Min.Z + ((ran.Max.Z - ran.Min.Z) * random),
		}
	} else {
		vec = Vector3{
			ran.Min.X + ((ran.Max.X - ran.Min.X) * rand.Float32()),
			ran.Min.Y + ((ran.Max.Y - ran.Min.Y) * rand.Float32()),
			ran.Min.Z + ((ran.Max.Z - ran.Min.Z) * rand.Float32()),
		}
	}

	return vec
}
