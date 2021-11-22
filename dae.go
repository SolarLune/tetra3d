package tetra3d

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/kvartborg/vector"
)

type daeSource struct {
	ID          string `xml:"id,attr"`
	StringArray string `xml:"float_array"`
}

func (source daeSource) Parse() []float64 {
	split := strings.Split(strings.ReplaceAll(source.StringArray, "\n", " "), " ")
	data := []float64{}
	for _, v := range split {
		f, _ := strconv.ParseFloat(v, 64)
		data = append(data, f)
	}
	return data
}

type daeTrianglesInput struct {
	Source string `xml:"source,attr"`
	Offset int    `xml:"offset,attr"`
}

type daeTriangles struct {
	String       string              `xml:"p"`
	MaterialName string              `xml:"material,attr"`
	Inputs       []daeTrianglesInput `xml:"input"`
}

type daeGeometry struct {
	Name      string       `xml:"name,attr"`
	URL       string       `xml:"id,attr"`
	Sources   []daeSource  `xml:"mesh>source"`
	Triangles daeTriangles `xml:"mesh>triangles"`
}

type daeLibraryGeometries struct {
	Geometries []daeGeometry `xml:"library_geometries>geometry"`
}

type daeLibraryMaterial struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

type daeLibraryMaterials struct {
	MaterialNames []daeLibraryMaterial `xml:"library_materials>material"`
}

type daeInstanceGeometry struct {
	URL string `xml:"url,attr"`
}

type daeNode struct {
	Name      string              `xml:"name,attr"`
	Mesh      daeInstanceGeometry `xml:"instance_geometry"`
	Transform string              `xml:"matrix"`
	Children  []daeNode           `xml:"node"`
}

func (daeNode daeNode) ParseTransform() Matrix4 {
	split := strings.Split(strings.ReplaceAll(daeNode.Transform, "\n", " "), " ")
	data := []float64{}
	for _, v := range split {
		f, _ := strconv.ParseFloat(v, 64)
		data = append(data, f)
	}

	// Matrices are column-major in Blender, so we need to adjust for that

	mat := NewMatrix4()

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			mat[j][i] = data[j+(i*4)]
		}
	}

	return mat
}

type daeVisualScene struct {
	Name  string    `xml:"name,attr"`
	Nodes []daeNode `xml:"node"`
}

type daeVisualLibraryScene struct {
	LibraryScene daeVisualScene `xml:"library_visual_scenes>visual_scene"`
}

func parseDAESourceName(sourceName string) string {

	idPieces := strings.Split(sourceName, "-")

	for endingIndex := len(idPieces) - 1; endingIndex >= 0; endingIndex-- {

		ending := strings.ToLower(idPieces[endingIndex])

		if strings.Contains(ending, "color") {
			return "color"
		} else if strings.Contains(ending, "position") || strings.Contains(ending, "vert") {
			return "vertex"
		} else if strings.Contains(ending, "normal") {
			return "normal"
		} else if strings.Contains(ending, "map") {
			return "uv"
		}

	}

	return ""

}

// DaeLoadOptions represents options one can use to tweak how .dae files are loaded into Tetra3D.
type DaeLoadOptions struct {
	CorrectYUp bool // Whether to correct Z being up for Blender importing.
}

// DefaultDaeLoadOptions returns a default instance of DaeLoadOptions.
func DefaultDaeLoadOptions() *DaeLoadOptions {
	return &DaeLoadOptions{
		CorrectYUp: true,
	}
}

// LoadDAEFile takes a filepath to a .dae model file, and returns a *Scene populated with the .dae file's objects and meshes.
// If the call couldn't complete for any reason, like due to a malformed DAE file, it will return an error.
// Note that this calls os.ReadFile(), and so requires Go v1.16 or above.
func LoadDAEFile(path string, options *DaeLoadOptions) (*Scene, error) {

	if fileData, err := os.ReadFile(path); err != nil {
		return nil, err
	} else {
		return LoadDAEData(fileData, options)
	}

}

// LoadDAEData takes a []byte consisting of the contents of a DAE file, and returns a *Scene populated with the .dae file's objects and meshes.
// If the call couldn't complete for any reason, like due to a malformed DAE file, it will return an error.
func LoadDAEData(data []byte, options *DaeLoadOptions) (*Scene, error) {

	if options == nil {
		options = DefaultDaeLoadOptions()
	}

	daeGeo := &daeLibraryGeometries{}
	daeScene := &daeVisualLibraryScene{}
	daeMaterials := &daeLibraryMaterials{}

	err := xml.Unmarshal(data, daeGeo)

	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal(data, &daeScene)

	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal(data, &daeMaterials)

	if err != nil {
		return nil, err
	}

	scene := NewScene(daeScene.LibraryScene.Name)

	toYUp := NewMatrix4Rotate(1, 0, 0, math.Pi/2)

	daeURLsToMeshes := map[string]*Mesh{}
	daeURLsToMaterialNames := map[string]string{}

	for _, mat := range daeMaterials.MaterialNames {
		daeURLsToMaterialNames[mat.ID] = mat.Name
	}

	for _, geo := range daeGeo.Geometries {

		sourceData := map[string][]float64{}

		for _, source := range geo.Sources {

			if parsedName := parseDAESourceName(source.ID); parsedName != "" {
				sourceData[parsedName] = source.Parse()
			} else {
				fmt.Println("Unknown geometry data, ", source.ID)
			}

		}

		// triangleOrder := append([]daeTrianglesInput{}, geo.TriangleInputs...)
		// sort.Slice(triangleOrder, func(i, j int) bool { return triangleOrder[i].Offset < triangleOrder[j].Offset })
		triangleOrder := []string{}
		for _, t := range geo.Triangles.Inputs {
			triangleOrder = append(triangleOrder, parseDAESourceName(t.Source))
		}
		sort.Slice(triangleOrder, func(i, j int) bool { return geo.Triangles.Inputs[i].Offset < geo.Triangles.Inputs[j].Offset })

		triangleIndices := []int64{}

		for _, t := range strings.Split(strings.TrimSpace(geo.Triangles.String), " ") {

			var ti int64
			ti, err = strconv.ParseInt(t, 0, 32)
			if err != nil {
				return nil, err
			}

			triangleIndices = append(triangleIndices, ti)

		}

		verts := []*Vertex{}
		normals := map[*Vertex]vector.Vector{}

		x, y, z := 0.0, 0.0, 0.0
		u, v := 0.0, 0.0
		r, g, b, a := float32(1.0), float32(1.0), float32(1.0), float32(1.0)
		nx, ny, nz := 0.0, 0.0, 0.0

		for i := 0; i < len(triangleIndices); i++ {

			triIndex := triangleIndices[i]

			tv := int(triIndex)

			switch triangleOrder[i%len(triangleOrder)] {

			case "vertex":
				x = sourceData["vertex"][tv*3]
				y = sourceData["vertex"][(tv*3)+1]
				z = sourceData["vertex"][(tv*3)+2]

			case "uv":
				u = sourceData["uv"][tv*2]
				v = sourceData["uv"][(tv*2)+1]

			case "color":
				r = float32(sourceData["color"][tv*4])
				g = float32(sourceData["color"][(tv*4)+1])
				b = float32(sourceData["color"][(tv*4)+2])
				a = float32(sourceData["color"][(tv*4)+3])

			case "normal":

				nx = sourceData["normal"][tv*3]
				ny = sourceData["normal"][(tv*3)+1]
				nz = sourceData["normal"][(tv*3)+2]

			}

			if i%len(triangleOrder) == len(triangleOrder)-1 {

				vert := NewVertex(x, y, z, u, v)

				vert.Color.R = r
				vert.Color.G = g
				vert.Color.B = b
				vert.Color.A = a

				verts = append(verts, vert)

				normals[vert] = vector.Vector{nx, ny, nz}

			}

		}

		mesh := NewMesh(geo.Name, verts...)

		if len(normals) > 0 {

			for _, tri := range mesh.Triangles {

				normal := vector.Vector{0, 0, 0}
				for _, vert := range tri.Vertices {
					normal = normal.Add(normals[vert])
				}
				normal = normal.Scale(1.0 / 3.0).Unit()

				normal = toYUp.MultVec(normal)
				normal[2] *= -1
				normal[1] *= -1
				tri.Normal = normal

			}

		}

		if options.CorrectYUp {
			mesh.ApplyMatrix(NewMatrix4Rotate(1, 0, 0, -math.Pi/2))
		}

		mesh.MaterialName = daeURLsToMaterialNames[geo.Triangles.MaterialName]
		scene.Meshes[geo.Name] = mesh
		daeURLsToMeshes[geo.URL] = mesh

	}

	var parseDAENode func(node daeNode) Node

	parseDAENode = func(node daeNode) Node {

		var mesh *Mesh

		if node.Mesh.URL != "" {

			meshURL := strings.Split(node.Mesh.URL, "#")[1]
			for url, m := range daeURLsToMeshes {
				if url == meshURL {
					mesh = m
					break
				}
			}

		}

		var model Node

		if mesh != nil {
			model = NewModel(mesh, node.Name)
		} else {
			model = NewNodeBase(node.Name)
		}

		mat := node.ParseTransform()

		// Correct transform matrix
		if options.CorrectYUp {

			// Tetra's +Y is Blender's +Z
			by := mat.Column(1)
			bz := mat.Column(2)

			mat = mat.SetColumn(1, bz)
			mat = mat.SetColumn(2, by.Invert())

			by = mat.Row(1)
			bz = mat.Row(2)

			mat = mat.SetRow(1, bz)
			mat = mat.SetRow(2, by.Invert())

		}

		// We parse and parent children before setting position, scale, and rotation because by doing it in this order,
		// we get the child being in its final transform as a result of the parent, rather than parenting leaving the child
		// at its original position. In other words, in the 3D modeler, the transform of children is the result of transforms
		// having been set AFTER parenting.
		for _, child := range node.Children {
			model.AddChildren(parseDAENode(child))
		}

		p, s, r := mat.Decompose()

		model.SetLocalPosition(p)
		model.SetLocalScale(s)
		model.SetLocalRotation(r)

		return model

	}

	for _, node := range daeScene.LibraryScene.Nodes {
		scene.Root.AddChildren(parseDAENode(node))
	}

	return scene, nil

}
