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

		if friction := part.ParticleSystem.Settings().Friction; friction > 0 {
			part.Velocity = part.Velocity.SubMagnitude(friction)
		}

		part.Model.MoveVec(part.Velocity)

	}

	if part.ParticleSystem.Settings().MovementFunction != nil {
		part.ParticleSystem.Settings().MovementFunction(part)
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

	if !part.ParticleSystem.Settings().AllowNegativeScale && (scale.X < 0 || scale.Y < 0 || scale.Z < 0) {
		part.Life = part.Lifetime // Die because the particle got too small
	}

	if part.Life >= part.Lifetime {
		part.Model.visible = false
		part.ParticleSystem.Remove(part)
	}

	if curve := part.ParticleSystem.Settings().ColorCurve; len(curve.Points) > 0 {
		part.Model.Color = curve.Color(part.Life / part.Lifetime)
	}

}

type ParticleSystemSettings struct {
	SpawnOn     bool         // If the particle system should spawn particles at all
	SpawnRate   FloatRange   // SpawnRate is how often a particle is spawned in seconds
	SpawnCount  IntRange     // SpawnCount is how many particles are spawned at a time when a particle is spawned
	Lifetime    FloatRange   // Lifetime is how long a particle lives in seconds
	SpawnOffset Vector3Range // The range indicating how far of an offset to move

	Velocity    Vector3Range // The range indicating how fast a particle constantly moves per frame
	VelocityAdd Vector3Range // The range indicating how fast a particle accelerates per frame

	Scale    Vector3Range // The range indicating how large the particle should spawn in as
	ScaleAdd Vector3Range // The range indicating how large the particle should grow per frame

	RotationAdd        Vector3Range // The range indicating how fast a particle should spin per frame
	LocalPosition      bool         // Whether the particles' positions should be local to the system or not (if true, particles move with the system); defaults to false.
	Friction           float32      // Friction to apply to velocity
	AllowNegativeScale bool         // If negative scale should be allowed for particles. By default, this is false.

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

func (p ParticleSystemSettings) WithSpawnOn(spawnOn bool) ParticleSystemSettings {
	p.SpawnOn = spawnOn
	return p
}

func (p ParticleSystemSettings) WithSpawnRate(spawnRate FloatRange) ParticleSystemSettings {
	p.SpawnRate = spawnRate
	return p
}

func (p ParticleSystemSettings) WithSpawnCount(spawnCount IntRange) ParticleSystemSettings {
	p.SpawnCount = spawnCount
	return p
}

func (p ParticleSystemSettings) WithLifetime(lifetime FloatRange) ParticleSystemSettings {
	p.Lifetime = lifetime
	return p
}

func (p ParticleSystemSettings) WithSpawnOffset(spawnOffset Vector3Range) ParticleSystemSettings {
	p.SpawnOffset = spawnOffset
	return p
}

func (p ParticleSystemSettings) WithVelocity(velocity Vector3Range) ParticleSystemSettings {
	p.Velocity = velocity
	return p
}

func (p ParticleSystemSettings) WithVelocityAdd(velocity Vector3Range) ParticleSystemSettings {
	p.VelocityAdd = velocity
	return p
}

func (p ParticleSystemSettings) WithScale(scale Vector3Range) ParticleSystemSettings {
	p.Scale = scale
	return p
}

func (p ParticleSystemSettings) WithScaleAdd(scale Vector3Range) ParticleSystemSettings {
	p.ScaleAdd = scale
	return p
}

func (p ParticleSystemSettings) WithRotationAdd(rotation Vector3Range) ParticleSystemSettings {
	p.RotationAdd = rotation
	return p
}

func (p ParticleSystemSettings) WithLocalPosition(localPosition bool) ParticleSystemSettings {
	p.LocalPosition = localPosition
	return p
}

func (p ParticleSystemSettings) WithFriction(friction float32) ParticleSystemSettings {
	p.Friction = friction
	return p
}

func (p ParticleSystemSettings) WithAllowNegativeScale(allow bool) ParticleSystemSettings {
	p.AllowNegativeScale = allow
	return p
}

func (p ParticleSystemSettings) WithVertexSpawn(model *Model, spawnMode int) ParticleSystemSettings {
	p.VertexSpawnModel = model
	p.VertexSpawnMode = spawnMode
	return p
}

func (p ParticleSystemSettings) WithSpawnOffsetFunction(spawnOffsetFunction func(p *Particle)) ParticleSystemSettings {
	p.SpawnOffsetFunction = spawnOffsetFunction
	return p
}

func (p ParticleSystemSettings) WithMovementFunction(movementFunction func(p *Particle)) ParticleSystemSettings {
	p.MovementFunction = movementFunction
	return p
}

func (p ParticleSystemSettings) WithColorCurve(colorCurve ColorCurve) ParticleSystemSettings {
	p.ColorCurve = colorCurve
	return p
}

// NewParticleSystemSettings creates a new particle system settings.
func NewParticleSystemSettings() ParticleSystemSettings {

	scale := NewVectorRange()
	scale.SetRanges(1, 1)

	lifetime := NewFloatRange()
	lifetime.Set(1, 1)

	spawnRate := NewFloatRange()
	spawnRate.Set(1, 1)

	spawnCount := NewIntRange()
	spawnCount.Set(1, 1)

	return ParticleSystemSettings{
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
func (pss *ParticleSystemSettings) Clone() ParticleSystemSettings {

	newPS := ParticleSystemSettings{
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

	BaseParticleModels []*Model

	spawnTimer       float32
	settings         ParticleSystemSettings
	model            *Model
	vertexSpawnIndex int
}

// NewParticleSystem creates a new ParticleSystem, operating on the baseModel Model and
// randomly creating particles from the provided collection of particle Models.
func NewParticleSystem(baseParticles ...*Model) *ParticleSystem {

	baseModel := NewModel("particle system", NewMesh("particle system mesh"))

	for _, part := range baseParticles {
		mat := part.mesh.MeshParts[0].Material
		if baseModel.mesh.MeshPartByMaterialName(mat.name) == nil {
			baseModel.mesh.AddMeshPart(part.mesh.MeshParts[0].Material)
		}
	}

	// baseModel.FrustumCulling = false // if we leave frustum culling on, the particles will turn invisible if the batch goes offscreen

	// for _, p := range particles {
	// 	p.AutoBatchMode = AutoBatchDynamic
	// }

	partSys := &ParticleSystem{
		model: baseModel,

		BaseParticleModels: baseParticles,

		LivingParticles: []*Particle{},
		DeadParticles:   []*Particle{},
		toRemove:        []*Particle{},

		settings: NewParticleSystemSettings(),

		On: true,
	}

	// We calculate the frustum sphere based off of the particles spawned
	partSys.model.updateFrustumSphere = false

	// partSys.root.SetVisible(false, false)

	return partSys

}

func (ps *ParticleSystem) Settings() ParticleSystemSettings {
	return ps.settings
}

func (ps *ParticleSystem) SetSettings(settings ParticleSystemSettings) {
	ps.settings = settings
}

// Clone creates a duplicate of the given ParticleSystem.
func (ps *ParticleSystem) Clone() *ParticleSystem {

	newPS := NewParticleSystem(ps.BaseParticleModels...)
	newPS.model = ps.model.Clone().(*Model)
	newPS.settings = ps.settings
	return newPS

}

// Update should be called once per tick.
func (ps *ParticleSystem) Update(dt float32) {

	if len(ps.BaseParticleModels) == 0 {
		return
	}

	furthestDist := float32(0.0)
	largestParticle := float32(0.0)

	// This could be multithreaded if the particles don't interact at all
	for _, part := range ps.LivingParticles {
		part.Update(dt)
		furthestDist = math32.Max(furthestDist, ps.model.DistanceSquaredTo(part.Model))
		largestParticle = math32.Max(largestParticle, part.Model.mesh.Dimensions.MaxSpan()*part.Model.scale.Magnitude())
	}

	ps.model.frustumCullingSphere.position = ps.model.WorldPosition()
	ps.model.frustumCullingSphere.Radius = (math32.Sqrt(furthestDist)) + largestParticle
	ps.model.frustumCullingSphere.scale = ps.model.WorldScale()
	// ps.root.FrustumCulling = false
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

	if ps.settings.SpawnOn {

		if ps.spawnTimer <= 0 {
			spawnCount := int(ps.settings.SpawnCount.value())
			for i := 0; i < spawnCount; i++ {
				ps.Spawn()
			}
			ps.spawnTimer = ps.settings.SpawnRate.value()
		}

		ps.spawnTimer -= dt
	}

	// if len(ps.root.DynamicBatchModels) > 0 {
	// 	ps.root.SetVisible(true, true)
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
		part = NewParticle(ps, ps.BaseParticleModels)
		for _, model := range part.ModelBank {
			ps.model.DynamicBatchAdd(model.mesh.MeshParts[0], model)
		}
		// for _, newModel := range part.ModelBank {
		// 	ps.root.DynamicBatchAdd(ps.root.Mesh.FindMeshPart(part.Model.Mesh.MeshParts[0].Material.Name), newModel)
		// }
	}

	ps.LivingParticles = append(ps.LivingParticles, part)

	part.Lifetime = ps.settings.Lifetime.value()

	part.Reinit()

	if ps.settings.LocalPosition {
		ps.model.AddChildren(part.Model)
	} else {
		root := ps.model.Root()
		if root != nil {
			root.AddChildren(part.Model)
		}
	}

	part.Model.SetWorldScaleVec(ps.settings.Scale.value())

	part.Velocity = ps.settings.Velocity.value()
	part.VelocityAdd = ps.settings.VelocityAdd.value()
	part.ScaleAdd = ps.settings.ScaleAdd.value()
	part.RotationAdd = ps.settings.RotationAdd.value()

	var pos Vector3

	if ps.settings.VertexSpawnMode != ParticleVertexSpawnModeOff && ps.settings.VertexSpawnModel != nil {

		model := ps.settings.VertexSpawnModel

		vertCount := len(model.mesh.VertexPositions)

		// TODO: Reimplement this for skinned models; this was working, but only probably reliably if not more
		// than one skinned model with the same mesh was active at any given time
		pos = model.Transform().MultVec(model.mesh.VertexPositions[ps.vertexSpawnIndex])

		switch ps.settings.VertexSpawnMode {
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
		pos = ps.model.WorldPosition()
	}

	part.Model.SetWorldPositionVec(pos.Add(ps.settings.SpawnOffset.value()))

	if ps.settings.SpawnOffsetFunction != nil {
		ps.settings.SpawnOffsetFunction(part)
	}

}

// Returns the renderable model for the particle system - rendering this renders all particles it has spawned.
func (ps *ParticleSystem) Model() *Model {
	return ps.model
}

// Remove removes a particle from the ParticleSystem, recycling the Particle for the next time a particle is spawned.
func (ps *ParticleSystem) Remove(part *Particle) {
	ps.toRemove = append(ps.toRemove, part)
}

type FloatRange struct {
	Min, Max   float32
	CustomFunc func(floatRange *FloatRange) float32
}

func NewFloatRange() FloatRange {
	return FloatRange{}
}

func (ran *FloatRange) Set(min, max float32) {
	ran.Min = min
	ran.Max = max
}

func (ran *FloatRange) SetCustomFunc(customFunc func(floatRange *FloatRange) float32) {
	ran.CustomFunc = customFunc
}

func (ran FloatRange) value() float32 {
	if ran.CustomFunc != nil {
		return ran.CustomFunc(&ran)
	}
	return ran.Value()
}

func (ran FloatRange) Value() float32 {
	random := rand.Float32()
	return ran.Min + ((ran.Max - ran.Min) * random)
}

type IntRange struct {
	Min, Max   int
	CustomFunc func(intRange *IntRange) int
}

func NewIntRange() IntRange {
	return IntRange{}
}

func (ran *IntRange) Set(min, max int) {
	ran.Min = min
	ran.Max = max
}

func (ran *IntRange) SetCustomFunc(customFunc func(nRange *IntRange) int) {
	ran.CustomFunc = customFunc
}

func (ran IntRange) value() int {
	if ran.CustomFunc != nil {
		return ran.CustomFunc(&ran)
	}
	return ran.Value()
}

func (ran IntRange) Value() int {
	if ran.Min >= ran.Max {
		return ran.Min
	}
	return ran.Min + rand.Intn(ran.Max-ran.Min)
}

// Vector3Range represents a range of possible values, and allows Tetra3D to get a random value from within
// that number range.
type Vector3Range struct {
	Uniform    bool    // If the random value returned by the NumberRange should be consistent across all axes or not
	Min        Vector3 // Min is the set of minimum numbers allowed in the NumberRange
	Max        Vector3 // Max is the set of maximum numbers allowed in the NumberRange
	CustomFunc func(vectorRange *Vector3Range) Vector3
}

// NewVectorRange returns a new instance of a 3D NumberRange struct.
func NewVectorRange() Vector3Range {
	return Vector3Range{}
}

// SetAll sets the minimum and maximum values of all components of the number range at the same time to the value
// passed.
func (ran *Vector3Range) SetAll(value float32) {
	ran.Min.X = value
	ran.Max.X = value
	ran.Min.Y = value
	ran.Max.Y = value
	ran.Min.Z = value
	ran.Max.Z = value
}

// SetAxes sets the minimum and maximum values of all components of the number range at the same time. Both
// minimum and maximum boundaries of the NumberRange will be the same.
func (ran *Vector3Range) SetAxes(x, y, z float32) {
	ran.Min.X = -x
	ran.Max.X = x

	ran.Min.Y = -y
	ran.Max.Y = y

	ran.Min.Z = -z
	ran.Max.Z = z
}

// SetRanges sets the minimum and maximum values of all components (axes) of the number range.
func (ran *Vector3Range) SetRanges(min, max float32) {

	ran.Min.X = min
	ran.Min.Y = min
	ran.Min.Z = min

	ran.Max.X = max
	ran.Max.Y = max
	ran.Max.Z = max

}

// SetRangeX sets the minimum and maximum values of the X component of the number range.
func (ran *Vector3Range) SetRangeX(min, max float32) {
	ran.Min.X = min
	ran.Max.X = max
}

// SetRangeY sets the minimum and maximum values of the Y component of the number range.
func (ran *Vector3Range) SetRangeY(min, max float32) {
	ran.Min.Y = min
	ran.Max.Y = max
}

// SetRangeZ sets the minimum and maximum values of the Z component of the number range.
func (ran *Vector3Range) SetRangeZ(min, max float32) {
	ran.Min.Z = min
	ran.Max.Z = max
}

func (ran *Vector3Range) value() Vector3 {
	if ran.CustomFunc != nil {
		return ran.CustomFunc(ran)
	}
	return ran.Value()
}

// Value returns a random value from within the bounds of the NumberRange.
func (ran Vector3Range) Value() Vector3 {

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
