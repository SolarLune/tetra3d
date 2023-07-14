package tetra3d

import (
	"bytes"
	"encoding/json"
	"image"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/qmuntal/gltf"
	"github.com/qmuntal/gltf/ext/lightspuntual"
	"github.com/qmuntal/gltf/modeler"

	_ "image/png"
)

type GLTFLoadOptions struct {
	// Width and height of loaded Cameras. Defaults to -1 each, which will then instead load the size from the camera's resolution as exported in the GLTF file;
	// if the camera resolution isn't set there, cameras will load with a 1920x1080 camera size.
	CameraWidth, CameraHeight int
	CameraDepth               bool // If cameras should render depth or not
	DefaultToAutoTransparency bool // If DefaultToAutoTransparency is true, then opaque materials become Auto transparent materials in Tetra3D.
	// DependentLibraryResolver is a function that takes a relative path (string) to the blend file representing the dependent Library that the loading
	// Library requires. This function should return a reference to the dependent Library; if it returns nil, the linked objects from the dependent Library
	// will not be instantiated in the loading Library.
	// An example would be loading a level (level.gltf) composed of assets from another file (a GLTF file exported from assets.blend, which is a directory up).
	// In this example, loading level.gltf would require the dependent library, found in assets.gltf. Loading level.gltf will refer to objects linked from the assets
	// blend file, known as "../assets.blend".
	// You could then simply load the assets library first and then code the DependentLibraryResolver function to take the assets library, or code the
	// function to use the path to load the library on demand. You could then store the loaded result as necessary if multiple levels use this assets Library.
	DependentLibraryResolver func(blendPath string) *Library

	//If top-level objects in collections should be renamed according to their instance objects.
	RenameCollectionObjects bool
}

// DefaultGLTFLoadOptions creates an instance of GLTFLoadOptions with some sensible defaults.
func DefaultGLTFLoadOptions() *GLTFLoadOptions {
	return &GLTFLoadOptions{
		CameraWidth:               -1,
		CameraHeight:              -1,
		CameraDepth:               true,
		DefaultToAutoTransparency: true,
		RenameCollectionObjects:   true,
	}
}

// LoadGLTFFile loads a .gltf or .glb file from the filepath given, using a provided GLTFLoadOptions struct to alter how the file is loaded.
// Passing nil for loadOptions will load the file using default load options. Unlike with DAE files, Animations (including armature-based
// animations) and Cameras (assuming they are exported in the GLTF file) will be parsed properly.
// LoadGLTFFile will return a Library, and an error if the process fails.
func LoadGLTFFile(path string, loadOptions *GLTFLoadOptions) (*Library, error) {

	fileData, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	return LoadGLTFData(fileData, loadOptions)

}

// LoadGLTFData loads a .gltf or .glb file from the byte data given, using a provided GLTFLoadOptions struct to alter how the file is loaded.
// Passing nil for loadOptions will load the file using default load options. Unlike with DAE files, Animations (including armature-based
// animations) and Cameras (assuming they are exported in the GLTF file) will be parsed properly.
// LoadGLTFFile will return a Library, and an error if the process fails.
func LoadGLTFData(data []byte, gltfLoadOptions *GLTFLoadOptions) (*Library, error) {

	decoder := gltf.NewDecoder(bytes.NewReader(data))

	doc := gltf.NewDocument()

	err := decoder.Decode(doc)

	if err != nil {
		return nil, err
	}

	if gltfLoadOptions == nil {
		gltfLoadOptions = DefaultGLTFLoadOptions()
	}

	library := NewLibrary()

	var images []*ebiten.Image

	exportedTextures := false

	type Collection struct {
		Objects []string
		Offset  []float64
		Path    string
	}

	collections := map[string]Collection{}

	t3dExport := false

	camWidth := gltfLoadOptions.CameraWidth
	camHeight := gltfLoadOptions.CameraHeight
	camDefaultSize := false

	sectorRendering := false
	sectorRenderDepth := 0

	if gltfLoadOptions.CameraWidth <= 0 && gltfLoadOptions.CameraHeight <= 0 {
		camWidth = 640
		camHeight = 360
		camDefaultSize = true
	}

	if len(doc.Scenes) > 0 {

		if doc.Scenes[0].Extras != nil {

			globalExporterSettings := doc.Scenes[0].Extras.(map[string]interface{})

			if camDefaultSize {

				if cr, exists := globalExporterSettings["t3dRenderResolutionW__"]; exists {
					if w, ok := cr.(float64); ok {
						camWidth = int(w)
					}
				}

				if cr, exists := globalExporterSettings["t3dRenderResolutionH__"]; exists {
					if h, ok := cr.(float64); ok {
						camHeight = int(h)
					}
				}

			}

			if et, exists := globalExporterSettings["t3dPackTextures__"]; exists {
				t3dExport = true
				exportedTextures = et.(bool)
			}

			if col, exists := globalExporterSettings["t3dCollections__"]; exists {
				t3dExport = true
				data := col.(map[string]interface{})

				jsonData, err := json.Marshal(data)
				if err != nil {
					panic(err)
				}

				err = json.Unmarshal(jsonData, &collections)

				if err != nil {
					panic(err)
				}

			}

			if on, exists := globalExporterSettings["t3dSectorRendering__"]; exists {
				sectorRendering = on.(bool)
			}

			if depth, exists := globalExporterSettings["t3dSectorRenderDepth__"]; exists {
				sectorRenderDepth = int(depth.(float64))
			}

		}

	}

	if exportedTextures {
		images = make([]*ebiten.Image, len(doc.Images))
		for i, gltfImage := range doc.Images {

			imageData, err := modeler.ReadBufferView(doc, doc.BufferViews[*gltfImage.BufferView])
			if err != nil {
				return nil, err
			}

			byteReader := bytes.NewReader(imageData)

			img, _, err := image.Decode(byteReader)
			if err != nil {
				return nil, err
			}

			images[i] = ebiten.NewImageFromImage(img)

		}

	}

	for _, gltfMat := range doc.Materials {

		newMat := NewMaterial(gltfMat.Name)
		newMat.library = library

		newMat.BackfaceCulling = !gltfMat.DoubleSided

		if texture := gltfMat.PBRMetallicRoughness.BaseColorTexture; texture != nil {
			if exportedTextures {
				newMat.Texture = images[*doc.Textures[texture.Index].Source]
			} else {
				newMat.TexturePath = doc.Images[*doc.Textures[texture.Index].Source].URI
			}
		}

		if gltfMat.Extras != nil {
			if dataMap, isMap := gltfMat.Extras.(map[string]interface{}); isMap {

				if c, exists := dataMap["t3dMaterialColor__"]; exists {
					color := c.([]interface{})
					newMat.Color.R = float32(color[0].(float64))
					newMat.Color.G = float32(color[1].(float64))
					newMat.Color.B = float32(color[2].(float64))
					newMat.Color.A = float32(color[3].(float64))
				}

				if s, exists := dataMap["t3dMaterialShadeless__"]; exists {
					newMat.Shadeless = s.(float64) > 0.5
				}

				if s, exists := dataMap["t3dMaterialFogless__"]; exists {
					newMat.Fogless = s.(float64) > 0.5
				}

				if s, exists := dataMap["t3dCompositeMode__"]; exists {
					switch int(s.(float64)) {
					case 0:
						newMat.CompositeMode = ebiten.CompositeModeSourceOver
					case 1:
						newMat.CompositeMode = ebiten.CompositeModeLighter
					// case 2:
					// 	newMat.CompositeMode = ebiten.CompositeModeMultiply // Multiply doesn't work right currently
					case 3:
						newMat.CompositeMode = ebiten.CompositeModeDestinationOut
						// newMat.CompositeMode = ebiten.CompositeModeClear
					}

				}

				if s, exists := dataMap["t3dBillboardMode__"]; exists {
					switch int(s.(float64)) {
					case 0:
						newMat.BillboardMode = BillboardModeNone
					case 1:
						newMat.BillboardMode = BillboardModeFixedVertical
					case 2:
						newMat.BillboardMode = BillboardModeHorizontal
					case 3:
						newMat.BillboardMode = BillboardModeAll
					}

				}

				if s, exists := dataMap["t3dCustomDepthOn__"]; exists {
					newMat.CustomDepthOffsetOn = s.(float64) > 0
				}
				if s, exists := dataMap["t3dCustomDepthValue__"]; exists {
					newMat.CustomDepthOffsetValue = s.(float64)
				}
				if s, exists := dataMap["t3dNormalFacesLights__"]; exists {
					newMat.NormalsAlwaysFaceLights = s.(float64) > 0
				}

				// At this point, parenting should be set up.

				if gameProps, exists := dataMap["t3dGameProperties__"]; exists {

					for _, p := range gameProps.([]interface{}) {

						name, value := handleGameProperties(p)

						newMat.properties.Get(name).Set(value)

					}
				}

				// Non-Tetra3D custom data
				for tagName, data := range dataMap {
					if !strings.HasPrefix(tagName, "t3d") || !strings.HasSuffix(tagName, "__") {
						newMat.properties.Get(tagName).Set(data)
					}
				}
			}
		}

		// If it's not exported through the Tetra addon, then just load the default GLTF material color value
		if !t3dExport {
			color := gltfMat.PBRMetallicRoughness.BaseColorFactor
			newMat.Color.R = float32(color[0])
			newMat.Color.G = float32(color[1])
			newMat.Color.B = float32(color[2])
			newMat.Color.A = float32(color[3])
		}

		newMat.Color.ConvertTosRGB()

		if gltfMat.AlphaMode == gltf.AlphaOpaque {
			if gltfLoadOptions.DefaultToAutoTransparency {
				newMat.TransparencyMode = TransparencyModeAuto
			} else {
				newMat.TransparencyMode = TransparencyModeOpaque
			}
		} else if gltfMat.AlphaMode == gltf.AlphaBlend {
			newMat.TransparencyMode = TransparencyModeTransparent
		} else { //if gltfMat.AlphaMode == gltf.AlphaMask
			newMat.TransparencyMode = TransparencyModeAlphaClip
		}

		library.Materials[gltfMat.Name] = newMat

	}

	for _, mesh := range doc.Meshes {

		// If t3dGrid__ is set on a mesh, then it can be skipped for loading
		if mesh.Extras != nil {

			if dataMap, isMap := mesh.Extras.(map[string]interface{}); isMap {

				if _, exists := dataMap["t3dGrid__"]; exists {
					continue
				}

			}

		}

		newMesh := NewMesh(mesh.Name)
		library.Meshes[mesh.Name] = newMesh
		newMesh.library = library

		colorChannelNames := []string{}

		if mesh.Extras != nil {

			if dataMap, isMap := mesh.Extras.(map[string]interface{}); isMap {

				if vcNames, exists := dataMap["t3dVertexColorNames__"]; exists {
					for index, name := range vcNames.([]interface{}) {
						newMesh.VertexColorChannelNames[name.(string)] = index
						colorChannelNames = append(colorChannelNames, name.(string))
					}
				}

				// Non-Tetra3D custom data
				for tagName, data := range dataMap {
					if !strings.HasPrefix(tagName, "t3d") || !strings.HasSuffix(tagName, "__") {
						newMesh.Properties.Get(tagName).Set(data)
					}
				}

			}

		}

		for _, v := range mesh.Primitives {

			posBuffer := [][3]float32{}
			vertPos, err := modeler.ReadPosition(doc, doc.Accessors[v.Attributes[gltf.POSITION]], posBuffer)

			if err != nil {
				return nil, err
			}

			vertexData := make([]VertexInfo, len(vertPos))

			for i, v := range vertPos {

				vertexData[i] = NewVertex(
					float64(v[0]),
					float64(v[1]),
					float64(v[2]),
					0, 0,
				)

			}

			if texCoordAccessor, texCoordExists := v.Attributes[gltf.TEXCOORD_0]; texCoordExists {

				uvBuffer := [][2]float32{}

				texCoords, err := modeler.ReadTextureCoord(doc, doc.Accessors[texCoordAccessor], uvBuffer)

				if err != nil {
					return nil, err
				}

				for i, v := range texCoords {
					vertexData[i].U = float64(v[0])
					vertexData[i].V = -(float64(v[1]) - 1)
				}

			}

			if normalAccessor, normalExists := v.Attributes[gltf.NORMAL]; normalExists {

				normalBuffer := [][3]float32{}

				normals, err := modeler.ReadNormal(doc, doc.Accessors[normalAccessor], normalBuffer)

				if err != nil {
					return nil, err
				}

				for i, v := range normals {
					vertexData[i].NormalX = float64(v[0])
					vertexData[i].NormalY = float64(v[1])
					vertexData[i].NormalZ = float64(v[2])
				}

			}

			// Blender's GLTF exporter changed - now the active vertex color channel
			// is turned to "COLOR_0", while the other channels are turned into attributes
			// accessible by their names like so: "_CHANNELNAMEINALLCAPS".

			if len(colorChannelNames) > 0 {

				dataMap, _ := mesh.Extras.(map[string]interface{})
				activeChannelIndex := int(dataMap["t3dActiveVertexColorIndex__"].(float64))

				for _, name := range colorChannelNames {

					name = "_" + strings.ToUpper(name)

					vertexColorAccessor, colorChannelExists := v.Attributes[name]

					if !colorChannelExists {
						vertexColorAccessor, colorChannelExists = v.Attributes["COLOR_0"]
					}

					if colorChannelExists {

						vcBuffer := [][4]uint16{}

						colors, err := modeler.ReadColor64(doc, doc.Accessors[vertexColorAccessor], vcBuffer)

						if err != nil {
							return nil, err
						}

						for i, colorData := range colors {

							// colors are exported from Blender as linear, but display as sRGB so we'll convert them here
							color := NewColor(
								float32(colorData[0])/math.MaxUint16,
								float32(colorData[1])/math.MaxUint16,
								float32(colorData[2])/math.MaxUint16,
								float32(colorData[3])/math.MaxUint16,
							)
							color.ConvertTosRGB()
							vertexData[i].Colors = append(vertexData[i].Colors, color)
							vertexData[i].ActiveColorChannel = activeChannelIndex

						}

					}

				}

			}

			if weightAccessor, weightExists := v.Attributes[gltf.WEIGHTS_0]; weightExists {

				weightBuffer := [][4]float32{}
				weights, err := modeler.ReadWeights(doc, doc.Accessors[weightAccessor], weightBuffer)

				if err != nil {
					return nil, err
				}

				boneBuffer := [][4]uint16{}
				bones, err := modeler.ReadJoints(doc, doc.Accessors[v.Attributes[gltf.JOINTS_0]], boneBuffer)

				if err != nil {
					return nil, err
				}

				// Store weights and bones; we don't want to waste space and speed storing bones if their weights are 0
				for w := range weights {
					vWeights := weights[w]
					for i := range vWeights {
						if vWeights[i] > 0 {
							vertexData[w].Weights = append(vertexData[w].Weights, vWeights[i])
							vertexData[w].Bones = append(vertexData[w].Bones, bones[w][i])
						}
					}
				}

			}

			newMesh.AddVertices(vertexData...)

			indexBuffer := []uint32{}

			indices, err := modeler.ReadIndices(doc, doc.Accessors[*v.Indices], indexBuffer)

			if err != nil {
				return nil, err
			}

			var mat *Material

			if v.Material != nil {
				gltfMat := doc.Materials[*v.Material]
				mat = library.Materials[gltfMat.Name]
			}

			newIndices := make([]int, len(indices))

			for i, j := range indices {
				newIndices[i] = int(j)
			}

			newMesh.AddMeshPart(mat, newIndices...)

			newMesh.UpdateBounds()

		}

	}

	for _, gltfAnim := range doc.Animations {
		anim := NewAnimation(gltfAnim.Name)
		anim.library = library
		library.Animations[gltfAnim.Name] = anim

		animLength := 0.0

		for _, channel := range gltfAnim.Channels {

			sampler := gltfAnim.Samplers[*channel.Sampler]

			channelName := "root"
			if channel.Target.Node != nil {
				channelName = doc.Nodes[*channel.Target.Node].Name
			}

			animChannel := anim.Channels[channelName]
			if animChannel == nil {
				animChannel = anim.AddChannel(channelName)
			}

			if channel.Target.Path == gltf.TRSTranslation {

				id, err := modeler.ReadAccessor(doc, doc.Accessors[sampler.Input], nil)

				if err != nil {
					return nil, err
				}

				inputData := id.([]float32)

				od, err := modeler.ReadAccessor(doc, doc.Accessors[sampler.Output], nil)

				if err != nil {
					return nil, err
				}

				outputData := od.([][3]float32)

				track := animChannel.AddTrack(TrackTypePosition)
				track.Interpolation = int(sampler.Interpolation)
				for i := 0; i < len(inputData); i++ {
					t := inputData[i]
					p := outputData[i]
					track.AddKeyframe(float64(t), Vector{float64(p[0]), float64(p[1]), float64(p[2]), 0})
					if float64(t) > animLength {
						animLength = float64(t)
					}
				}

			} else if channel.Target.Path == gltf.TRSScale {

				id, err := modeler.ReadAccessor(doc, doc.Accessors[sampler.Input], nil)

				if err != nil {
					return nil, err
				}

				inputData := id.([]float32)

				od, err := modeler.ReadAccessor(doc, doc.Accessors[sampler.Output], nil)

				if err != nil {
					return nil, err
				}

				outputData := od.([][3]float32)

				track := animChannel.AddTrack(TrackTypeScale)
				track.Interpolation = int(sampler.Interpolation)
				for i := 0; i < len(inputData); i++ {
					t := inputData[i]
					p := outputData[i]
					track.AddKeyframe(float64(t), Vector{float64(p[0]), float64(p[1]), float64(p[2]), 0})
					if float64(t) > animLength {
						animLength = float64(t)
					}
				}

			} else if channel.Target.Path == gltf.TRSRotation {

				id, err := modeler.ReadAccessor(doc, doc.Accessors[sampler.Input], nil)

				if err != nil {
					return nil, err
				}

				inputData := id.([]float32)

				od, err := modeler.ReadAccessor(doc, doc.Accessors[sampler.Output], nil)

				if err != nil {
					return nil, err
				}

				outputData := od.([][4]float32)

				track := animChannel.AddTrack(TrackTypeRotation)
				track.Interpolation = int(sampler.Interpolation)

				for i := 0; i < len(inputData); i++ {
					t := inputData[i]
					p := outputData[i]
					track.AddKeyframe(float64(t), NewQuaternion(float64(p[0]), float64(p[1]), float64(p[2]), float64(p[3])))
					if float64(t) > animLength {
						animLength = float64(t)
					}
				}

			}

		}

		if gltfAnim.Extras != nil {
			m := gltfAnim.Extras.(map[string]interface{})
			if markerData, exists := m["t3dMarkers__"]; exists {
				for _, mData := range markerData.([]interface{}) {

					marker := mData.(map[string]interface{})

					anim.Markers = append(anim.Markers, Marker{
						Name: marker["name"].(string),
						Time: marker["time"].(float64),
					})
				}
			}
		}

		anim.Length = animLength

	}

	// skins := []*Skin{}

	// for _, skin := range doc.Skins {

	// 	skins = append(skins, skin.)

	// }

	objects := []INode{}

	objToNode := map[INode]*gltf.Node{}

	nodeHasProp := func(node *gltf.Node, propName string) bool {

		if node.Extras == nil {
			return false
		}

		if _, exists := node.Extras.(map[string]interface{})[propName]; exists {
			return true
		}
		return false

	}

	nodeGetProp := func(node *gltf.Node, propName string) interface{} {

		if node.Extras == nil {
			return nil
		}

		if value, exists := node.Extras.(map[string]interface{})[propName]; exists {
			return value
		}
		return nil

	}

	// Node / Object creation
	for _, node := range doc.Nodes {

		var obj INode

		var mesh *Mesh

		if node.Mesh != nil {
			mesh = library.Meshes[doc.Meshes[*node.Mesh].Name]
		}

		if mesh != nil {
			obj = NewModel(mesh, node.Name)

			if node.Extras != nil && nodeHasProp(node, "t3dAutoBatch__") {
				s := node.Extras.(map[string]interface{})["t3dAutoBatch__"].(float64)
				obj.(*Model).AutoBatchMode = int(s)
			}

		} else if node.Camera != nil {

			gltfCam := doc.Cameras[*node.Camera]

			newCam := NewCamera(camWidth, camHeight)
			newCam.name = node.Name
			newCam.RenderDepth = gltfLoadOptions.CameraDepth
			newCam.updateProjectionMatrix = true

			if gltfCam.Perspective != nil {
				newCam.near = float64(gltfCam.Perspective.Znear)
				newCam.far = float64(*gltfCam.Perspective.Zfar)
				newCam.fieldOfView = ToDegrees(float64(gltfCam.Perspective.Yfov))
				newCam.perspective = true
			} else if gltfCam.Orthographic != nil {
				newCam.near = float64(gltfCam.Orthographic.Znear)
				newCam.far = float64(gltfCam.Orthographic.Zfar)
				newCam.orthoScale = float64(gltfCam.Orthographic.Xmag * 2)
				newCam.perspective = false
			}

			newCam.SectorRendering = sectorRendering
			newCam.SectorRenderDepth = sectorRenderDepth

			obj = newCam

		} else if lighting := node.Extensions["KHR_lights_punctual"]; lighting != nil {
			lights := doc.Extensions["KHR_lights_punctual"].(lightspuntual.Lights)
			lightData := lights[lighting.(lightspuntual.LightIndex)]

			if lightData.Type == lightspuntual.TypeDirectional {
				directionalLight := NewDirectionalLight(node.Name, lightData.Color[0], lightData.Color[1], lightData.Color[2], *lightData.Intensity) // Sun is in "energy"
				obj = directionalLight
			} else if lightData.Type == lightspuntual.TypePoint {
				pointLight := NewPointLight(node.Name, lightData.Color[0], lightData.Color[1], lightData.Color[2], *lightData.Intensity/80) // Point lights have wattage energy
				if !math.IsInf(float64(*lightData.Range), 0) {
					pointLight.Range = float64(*lightData.Range)
				}
				obj = pointLight
			} else {
				// Any unsupported light type just gets turned into an ambient light
				pointLight := NewAmbientLight(node.Name, lightData.Color[0], lightData.Color[1], lightData.Color[2], *lightData.Intensity/80)
				obj = pointLight
			}

		} else if node.Extras != nil && nodeHasProp(node, "t3dPathPoints__") {

			points := []Vector{}
			extraMap := node.Extras.(map[string]interface{})

			for _, p := range extraMap["t3dPathPoints__"].([]interface{}) {
				pointData := p.([]interface{})
				points = append(points, Vector{pointData[0].(float64), pointData[2].(float64), -pointData[1].(float64), 0})
			}

			path := NewPath(node.Name, points...)

			if nodeHasProp(node, "t3dPathCyclic__") {
				path.Closed = extraMap["t3dPathCyclic__"].(bool)
			}

			obj = path

		} else if node.Extras != nil && nodeHasProp(node, "t3dGridConnections__") {

			obj = NewGrid(node.Name)

			extraMap := node.Extras.(map[string]interface{})

			type gridPointPosition struct {
				X, Y, Z float64
			}

			creationOrder := []*GridPoint{}

			parsePosition := func(gpString string) gridPointPosition {
				trimmed := strings.Trim(gpString, "()")
				components := strings.Split(trimmed, ", ")
				x, _ := strconv.ParseFloat(components[0], 64)
				y, _ := strconv.ParseFloat(components[1], 64)
				z, _ := strconv.ParseFloat(components[2], 64)

				return gridPointPosition{
					X: x,
					Y: y,
					Z: z,
				}
			}

			for _, k := range extraMap["t3dGridEntries__"].([]interface{}) {

				key := parsePosition(k.(string))
				gp := NewGridPoint("Grid Point")
				obj.AddChildren(gp)
				gp.SetWorldPosition(key.X, key.Z, -key.Y)
				creationOrder = append(creationOrder, gp)

			}

			for k, array := range extraMap["t3dGridConnections__"].(map[string]interface{}) {

				key, _ := strconv.Atoi(k)
				for _, v := range array.([]interface{}) {
					connectedID, _ := strconv.Atoi(v.(string))
					creationOrder[key].Connect(creationOrder[connectedID])
				}

			}

		} else {
			obj = NewNode(node.Name)
		}

		objToNode[obj] = node

		obj.setLibrary(library)

		for _, child := range node.Children {
			obj.AddChildren(objects[int(child)])
		}

		if node.Extras != nil {
			if dataMap, isMap := node.Extras.(map[string]interface{}); isMap {

				getOrDefaultBool := func(path string, defaultValue bool) bool {
					if value, exists := dataMap[path]; exists {
						return value.(float64) > 0.5
					}
					return defaultValue
				}

				getOrDefaultFloat := func(path string, defaultValue float64) float64 {
					if value, exists := dataMap[path]; exists {
						return value.(float64)
					}
					return defaultValue
				}

				getOrDefaultFloatSlice := func(path string, defaultValues []float64) []float64 {
					if value, exists := dataMap[path]; exists {
						floats := []float64{}
						for _, v := range value.([]interface{}) {
							floats = append(floats, v.(float64))
						}
						return floats
					}
					return defaultValues
				}

				obj.SetVisible(getOrDefaultBool("t3dVisible__", true), false)

				if bt, exists := dataMap["t3dBoundsType__"]; exists {

					boundsType := int(bt.(float64))

					switch boundsType {
					// case 0: // NONE
					case 1: // AABB

						var aabb *BoundingAABB

						if aabbCustomEnabled := getOrDefaultBool("t3dAABBCustomEnabled__", false); aabbCustomEnabled {

							boundsSize := getOrDefaultFloatSlice("t3dAABBCustomSize__", []float64{2, 2, 2})
							aabb = NewBoundingAABB("BoundingAABB", boundsSize[0], boundsSize[1], boundsSize[2])

						} else if obj.Type().Is(NodeTypeModel) && obj.(*Model).Mesh != nil {
							mesh := obj.(*Model).Mesh
							dim := mesh.Dimensions
							aabb = NewBoundingAABB("BoundingAABB", dim.Width(), dim.Height(), dim.Depth())
						}

						if aabb != nil {

							if obj.Type().Is(NodeTypeModel) && obj.(*Model).Mesh != nil {
								aabb.SetLocalPositionVec(obj.(*Model).Mesh.Dimensions.Center())
							}

							obj.AddChildren(aabb)

						} else {
							log.Println("Warning: object " + obj.Name() + " has bounds type BoundingAABB with no size and is not a Model")
						}

					case 2: // Capsule

						var capsule *BoundingCapsule

						if capsuleCustomEnabled := getOrDefaultBool("t3dCapsuleCustomEnabled__", false); capsuleCustomEnabled {
							height := getOrDefaultFloat("t3dCapsuleCustomHeight__", 2)
							radius := getOrDefaultFloat("t3dCapsuleCustomRadius__", 0.5)
							capsule = NewBoundingCapsule("BoundingCapsule", height, radius)
						} else if obj.Type().Is(NodeTypeModel) && obj.(*Model).Mesh != nil {
							mesh := obj.(*Model).Mesh
							dim := mesh.Dimensions
							capsule = NewBoundingCapsule("BoundingCapsule", dim.Height(), math.Max(dim.Width(), dim.Depth())/2)
						}

						if capsule != nil {

							if obj.Type().Is(NodeTypeModel) && obj.(*Model).Mesh != nil {
								capsule.SetLocalPositionVec(obj.(*Model).Mesh.Dimensions.Center())
							}

							obj.AddChildren(capsule)

						} else {
							log.Println("Warning: object " + obj.Name() + " has bounds type BoundingCapsule with no size and is not a Model")
						}

					case 3: // Sphere

						var sphere *BoundingSphere

						if sphereCustomEnabled := getOrDefaultBool("t3dSphereCustomEnabled__", false); sphereCustomEnabled {
							radius := getOrDefaultFloat("t3dSphereCustomRadius__", 1)
							sphere = NewBoundingSphere("BoundingSphere", radius)
						} else if obj.Type().Is(NodeTypeModel) && obj.(*Model).Mesh != nil {

							model := obj.(*Model)
							dim := model.Mesh.Dimensions
							scale := model.WorldScale()

							dim.Min = dim.Min.Mult(scale)
							dim.Max = dim.Max.Mult(scale)

							sphere = NewBoundingSphere("BoundingSphere", dim.MaxDimension()/2)
						}

						if sphere != nil {

							if obj.Type().Is(NodeTypeModel) && obj.(*Model).Mesh != nil {
								sphere.SetLocalPositionVec(obj.(*Model).Mesh.Dimensions.Center())
							}

							obj.AddChildren(sphere)

						} else {
							log.Println("Warning: object " + obj.Name() + " has bounds type BoundingSphere with no size and is not a Model")
						}

					case 4: // Triangles

						if obj.Type().Is(NodeTypeModel) && obj.(*Model).Mesh != nil {

							gridSize := 20.0

							if getOrDefaultBool("t3dTrianglesCustomBroadphaseEnabled__", false) {
								gridSize = getOrDefaultFloat("t3dTrianglesCustomBroadphaseGridSize__", 20)
							}

							triangles := NewBoundingTriangles("BoundingTriangles", obj.(*Model).Mesh, gridSize)

							obj.AddChildren(triangles)
						}

					}
				}

				// Non-Tetra3D custom data
				for tagName, data := range dataMap {
					if !strings.HasPrefix(tagName, "t3d") || !strings.HasSuffix(tagName, "__") {
						obj.Properties().Get(tagName).Set(data)
					}
				}
			}
		}

		mtData := node.Matrix

		matrix := NewMatrix4()
		matrix.SetRow(0, Vector{float64(mtData[0]), float64(mtData[1]), float64(mtData[2]), float64(mtData[3])})
		matrix.SetRow(1, Vector{float64(mtData[4]), float64(mtData[5]), float64(mtData[6]), float64(mtData[7])})
		matrix.SetRow(2, Vector{float64(mtData[8]), float64(mtData[9]), float64(mtData[10]), float64(mtData[11])})
		matrix.SetRow(3, Vector{float64(mtData[12]), float64(mtData[13]), float64(mtData[14]), float64(mtData[15])})

		if !matrix.IsIdentity() {

			p, s, r := matrix.Decompose()

			obj.SetLocalPositionVec(p)
			obj.SetLocalScaleVec(s)
			obj.SetLocalRotation(r)

		} else {

			obj.SetLocalPositionVec(Vector{float64(node.Translation[0]), float64(node.Translation[1]), float64(node.Translation[2]), 0})
			obj.SetLocalScaleVec(Vector{float64(node.Scale[0]), float64(node.Scale[1]), float64(node.Scale[2]), 0})
			obj.SetLocalRotation(NewQuaternion(float64(node.Rotation[0]), float64(node.Rotation[1]), float64(node.Rotation[2]), float64(node.Rotation[3])).ToMatrix4())

		}

		if value := nodeGetProp(node, "t3dSector__"); value != nil && value.(float64) > 0 {
			obj.(*Model).sector = NewSector(obj.(*Model))
		}

		objects = append(objects, obj)

	}

	allBones := []*Node{}

	// We do this again here so we can be sure that all of the nodes can be created first
	for i, node := range doc.Nodes {

		// Set up skin for skinning animations
		if node.Skin != nil {

			model := objects[i].(*Model)

			skin := doc.Skins[*node.Skin]

			// Unsure of if this is necessary.
			// if skin.Skeleton != nil {
			// 	skeletonRoot := objects[*skin.Skeleton]
			// 	model.SetWorldPositionVec(skeletonRoot.WorldPosition())
			// 	model.SetWorldScale(skeletonRoot.WorldScale())
			// 	model.SetWorldRotation(skeletonRoot.WorldRotation())
			// }

			model.skinned = true

			// We should keep a local slice of bones because we can't simply loop through all bones with the matrix index, as
			// the matrix index resets at 0 for each armature, naturally.
			localBones := []*Node{}

			for _, b := range skin.Joints {
				bone := objects[b].(*Node)
				// This is incorrect, but it gives us a link to any bone in the armature to establish
				// the true root after parenting is set below
				model.SkinRoot = bone
				allBones = append(allBones, bone)
				localBones = append(localBones, bone)
			}

			matrices, err := modeler.ReadAccessor(doc, doc.Accessors[*skin.InverseBindMatrices], nil)
			if err != nil {
				return nil, err
			}

			for matIndex, matrix := range matrices.([][4][4]float32) {

				newMat := NewMatrix4()
				for rowIndex, row := range matrix {
					newMat.SetColumn(rowIndex, Vector{float64(row[0]), float64(row[1]), float64(row[2]), float64(row[3])})
				}

				localBones[matIndex].inverseBindMatrix = newMat
				localBones[matIndex].isBone = true

			}

			for vertIndex, boneIndices := range model.Mesh.VertexBones {
				model.bones = append(model.bones, []*Node{})

				for _, boneID := range boneIndices {
					model.bones[vertIndex] = append(model.bones[vertIndex], allBones[boneID])
				}

				// model.Mesh.VertexBones[i] = append(model.Mesh.VertexBones[i], )
			}

			// for _, part := range model.Mesh.MeshParts {

			// 	for _, vertex := range part.Vertices {

			// 		model.bones = append(model.bones, []*Node{})

			// 		for _, boneID := range verticesToBones[vertex] {
			// 			model.bones[vertex.ID] = append(model.bones[vertex.ID], allBones[boneID])
			// 		}

			// 	}

			// }

		}

		// Set up parenting
		for _, childIndex := range node.Children {
			objects[i].AddChildren(objects[int(childIndex)])
		}

	}

	findNode := func(objName string) INode {
		for _, obj := range objects {
			if obj.Name() == objName {
				return obj
			}
		}
		return nil
	}

	// At this point, parenting should be set up.
	for obj, node := range objToNode {

		if node.Extras != nil {
			if dataMap, isMap := node.Extras.(map[string]interface{}); isMap {

				if gameProps, exists := dataMap["t3dGameProperties__"]; exists {

					for _, p := range gameProps.([]interface{}) {

						name, value := handleGameProperties(p)

						obj.Properties().Get(name).Set(value)

					}
				}

			}

		}
	}

	// Set up SkinRoot for skinned Models; this should be the root node of a hierarchy of bone Nodes.
	for _, n := range objects {

		if model, isModel := n.(*Model); isModel && model.skinned {

			parent := model.SkinRoot

			for parent.IsBone() && parent.Parent() != nil {
				parent = parent.Parent()
			}

			model.SkinRoot = parent

		}

	}

	for obj, node := range objToNode {

		if node.Extras != nil {
			if dataMap, isMap := node.Extras.(map[string]interface{}); isMap {

				if c, exists := dataMap["t3dInstanceCollection__"]; exists {

					n := obj.(*Node)
					n.collectionObjects = []INode{}

					collection := collections[c.(string)]

					offset := Vector{-collection.Offset[0], -collection.Offset[2], collection.Offset[1], 0}

					for _, cloneName := range collection.Objects {

						var clone INode

						path := collection.Path

						if path == "" {
							clone = findNode(cloneName).Clone()
						} else {
							path = convertBlenderPath(path)
							if gltfLoadOptions.DependentLibraryResolver == nil {
								log.Printf("Warning: No dependent library resolver defined to resolve dependent library %s for object %s.\n", path, cloneName)
							} else {

								if library := gltfLoadOptions.DependentLibraryResolver(path); library != nil {
									if foundNode := library.FindNode(cloneName); foundNode != nil {
										clone = foundNode.Clone()
									} else {
										panic("Error in instantiating linked element: " + cloneName + " as there is no such object in the returned library.")
									}
								} else {
									log.Printf("Warning: No library returned in resolving dependent library %s for object %s.\n", path, cloneName)
								}

							}

						}

						if clone != nil {

							clone.MoveVec(offset)
							obj.AddChildren(clone)

							n.collectionObjects = append(n.collectionObjects, clone)

							// Share properties to top-level clones
							for k, v := range obj.Properties().props {
								clone.Properties().Get(k).Set(v.Value)
							}

						} else {
							log.Println("Error in instantiating linked element:", cloneName, "from:", path, "; did you pass the Library as a dependent Library in the GLTFLoadOptions struct?")
						}

					}

				}

			}

		}
	}

	// Set up worlds
	if len(doc.Scenes) > 0 && doc.Scenes[0].Extras != nil {

		dataMap := doc.Scenes[0].Extras.(map[string]interface{})

		if wd, exists := dataMap["t3dWorlds__"]; exists {

			worldData := wd.(map[string]interface{})

			for worldName, p := range worldData {

				world := NewWorld(worldName)

				props := p.(map[string]interface{})

				if wc, exists := props["ambient color"]; exists {
					wcc := wc.([]interface{})
					worldColor := NewColor(float32(wcc[0].(float64)), float32(wcc[1].(float64)), float32(wcc[2].(float64)), 1)
					worldColor.ConvertTosRGB()
					world.AmbientLight.Color.Set(worldColor.R, worldColor.G, worldColor.B, 1)
				}

				if wc, exists := props["ambient energy"]; exists {
					world.AmbientLight.Energy = float32(wc.(float64))
				}

				if cc, exists := props["clear color"]; exists {
					wcc := cc.([]interface{})
					clearColor := NewColor(float32(wcc[0].(float64)), float32(wcc[1].(float64)), float32(wcc[2].(float64)), float32(wcc[3].(float64)))
					clearColor.ConvertTosRGB()
					world.ClearColor = clearColor
				}

				if v, exists := props["fog mode"]; exists {
					world.FogOn = true
					fm := v.(string)
					switch fm {
					case "OFF":
						world.FogOn = false
					case "ADDITIVE":
						world.FogMode = FogAdd
					case "SUBTRACT":
						world.FogMode = FogSub
					case "OVERWRITE":
						world.FogMode = FogOverwrite
					case "TRANSPARENT":
						world.FogMode = FogTransparent
					}
				}

				if v, exists := props["fog curve"]; exists {
					fc := v.(string)
					switch fc {
					case "LINEAR":
						world.FogCurve = FogCurveLinear
					case "OUTCIRC":
						world.FogCurve = FogCurveOutCirc
					case "INCIRC":
						world.FogCurve = FogCurveInCirc
					}
				}

				if v, exists := props["dithered transparency"]; exists {
					world.DitheredFogSize = float32(v.(float64))
				}

				if v, exists := props["fog color"]; exists {
					wcc := v.([]interface{})
					fogColor := NewColor(float32(wcc[0].(float64)), float32(wcc[1].(float64)), float32(wcc[2].(float64)), float32(wcc[3].(float64)))
					fogColor.ConvertTosRGB()
					world.FogColor = fogColor
				}

				if v, exists := props["fog range start"]; exists {
					fogStart := v.(float64)
					world.FogRange[0] = float32(fogStart)
				}

				if v, exists := props["fog range end"]; exists {
					fogEnd := v.(float64)
					world.FogRange[1] = float32(fogEnd)
				}

				library.Worlds[world.Name] = world

			}

		}

	}

	// Set up scene roots

	for _, s := range doc.Scenes {

		scene := library.AddScene(s.Name)

		// Parent all parentless objects to the scene root to be visible.
		for _, n := range s.Nodes {
			scene.Root.AddChildren(objects[n])
		}

		if s.Extras != nil {
			extras := s.Extras.(map[string]interface{})
			if wn, exists := extras["t3dCurrentWorld__"]; exists {
				scene.World = library.Worlds[wn.(string)]
			}

			if gameProps, exists := extras["t3dGameProperties__"]; exists {

				for _, p := range gameProps.([]interface{}) {

					name, value := handleGameProperties(p)

					scene.Properties().Get(name).Set(value)

				}
			}

			// Non-Tetra3D custom data
			for tagName, data := range extras {
				if !strings.HasPrefix(tagName, "t3d") || !strings.HasSuffix(tagName, "__") {
					scene.Properties().Get(tagName).Set(data)
				}
			}

		}

		/*
			Below, we handle collection instances specially, as objects in the collection should be instantiated
			instead in the same relative locations, rather than parenting them to their collection instance objects.
			Otherwise, there would be nodes in the middle - the collection instance. For example, a collection composed
			of a chair mesh parented to a collision box would become a hierarchy like this:

			* Root
				|- * Chair (Collection Instance, named "Chair.001")
					|- * Chair Collision Box (named "ChairCol")
						|- * Chair Mesh

			Instead, we can substitute the chair collision box in the collection instance's location:

			* Root
				|- * Chair Collision Box (now optionally named "Chair.001")
					|- * Chair Mesh

		*/

		for _, n := range scene.Root.SearchTree().INodes() {

			n.setOriginalTransform()

			if node, ok := n.(*Node); ok && len(node.collectionObjects) > 0 {

				for _, child := range node.collectionObjects {
					transform := child.Transform()
					node.parent.AddChildren(child)
					if gltfLoadOptions.RenameCollectionObjects {
						child.SetName(node.name)
					}
					child.SetWorldTransform(transform)
				}

				// TODO: For now, this code doesn't do anything because objects parented to instance collections
				// in Blender aren't exported (for whatever reason).
				for _, child := range node.children {
					node.collectionObjects[0].AddChildren(child)
				}

				node.Unparent()

			}

			models := scene.Root.SearchTree().Models()
			if model, ok := n.(*Model); ok && model.sector != nil {
				model.sector.UpdateNeighbors(models...)
			}

		}

	}

	// Cameras exported through GLTF become nodes + a camera child with the correct orientation for some reason???
	// So here we basically cut the empty nodes out of the equation, leaving just the cameras with the correct orientation.

	// EDIT: This is no longer done, as the camera direction in Blender and the camera direction in GLTF aren't the same, whoops.
	// See: https://github.com/KhronosGroup/glTF-Blender-Exporter/issues/113
	// Cutting out the inserted correction Node breaks relative transforms (i.e. camera parented to another object for positioning).

	// for _, n := range objects {

	// 	if camera, isCamera := n.(*Camera); isCamera {
	// 		oldParent := camera.Parent()
	// 		root := oldParent.Parent()

	// 		camera.name = oldParent.Name()

	// 		for _, child := range oldParent.Children() {
	// 			if child == camera {
	// 				continue
	// 			}
	// 			camera.AddChildren(child)
	// 		}

	// 		root.RemoveChildren(camera.parent)
	// 		root.AddChildren(camera)
	// 	}

	// }

	library.ExportedScene = library.Scenes[*doc.Scene]

	return library, nil

}

func handleGameProperties(p interface{}) (string, interface{}) {

	getOrDefaultInt := func(propMap map[string]interface{}, key string, defaultValue int) int {
		if value, keyExists := propMap[key]; keyExists {
			return int(value.(float64))
		}
		return defaultValue
	}

	getOrDefaultString := func(propMap map[string]interface{}, key string, defaultValue string) string {
		if value, keyExists := propMap[key]; keyExists {
			return value.(string)
		}
		return defaultValue
	}

	getOrDefaultFloat := func(propMap map[string]interface{}, key string, defaultValue float64) float64 {
		if value, keyExists := propMap[key]; keyExists {
			return value.(float64)
		}
		return defaultValue
	}

	getOrDefaultBool := func(propMap map[string]interface{}, key string, defaultValue bool) bool {
		if value, keyExists := propMap[key]; keyExists {
			return value.(float64) > 0
		}
		return defaultValue
	}

	getIfExistingMap := func(propMap map[string]interface{}, key string) map[string]interface{} {
		if value, keyExists := propMap[key]; keyExists && value != nil {
			return value.(map[string]interface{})
		}
		return nil
	}

	getOrDefaultFloatArray := func(propMap map[string]interface{}, key string, defaultValue []float64) []float64 {
		if value, keyExists := propMap[key]; keyExists {
			values := make([]float64, 0, len(value.([]interface{})))
			for _, v := range value.([]interface{}) {
				values = append(values, v.(float64))
			}
			return values
		}
		return defaultValue
	}

	property := p.(map[string]interface{})

	propType := getOrDefaultInt(property, "valueType", 0)

	// Property types:

	// bool, int, float, string, reference (string)

	name := getOrDefaultString(property, "name", "New Property")
	var value interface{}

	if propType == 0 {
		value = getOrDefaultBool(property, "valueBool", false)
	} else if propType == 1 {
		value = getOrDefaultInt(property, "valueInt", 0)
	} else if propType == 2 {
		value = getOrDefaultFloat(property, "valueFloat", 0)
	} else if propType == 3 {
		value = getOrDefaultString(property, "valueString", "")
	} else if propType == 4 {
		scene := ""
		// Can be nil if it was set to something and then set to nothing
		if ref := getIfExistingMap(property, "valueReferenceScene"); ref != nil {
			scene = getOrDefaultString(ref, "name", "")
		}
		if ref := getIfExistingMap(property, "valueReference"); ref != nil {
			value = scene + ":" + getOrDefaultString(ref, "name", "")
		}
	} else if propType == 5 {
		colorValues := getOrDefaultFloatArray(property, "valueColor", []float64{1, 1, 1, 1})
		color := NewColor(float32(colorValues[0]), float32(colorValues[1]), float32(colorValues[2]), float32(colorValues[3]))
		color.ConvertTosRGB()
		value = color
	} else if propType == 6 {
		vecValues := getOrDefaultFloatArray(property, "valueVector3D", []float64{0, 0, 0})
		value = Vector{vecValues[0], vecValues[2], -vecValues[1], 0}
	} else if propType == 7 {
		value = getOrDefaultString(property, "valueFilepath", "")
		if value != "" {
			value = convertBlenderPath(value.(string))
		}
	}

	return name, value

}

func convertBlenderPath(path string) string {
	path = strings.ReplaceAll(path, "//", "") // Blender relative paths have double-slashes; we don't need them to
	return path
}
