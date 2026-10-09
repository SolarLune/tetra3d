package tetra3d

import (
	"bytes"
	"testing"

	"github.com/qmuntal/gltf"
	"github.com/qmuntal/gltf/modeler"
)

// vertexDataGLTF builds a glTF with one triangle of three vertices.
// The attributes function adds extra attributes and can change the indices.
func vertexDataGLTF(t *testing.T, indices []uint16, attributes func(doc *gltf.Document, attrs gltf.PrimitiveAttributes)) []byte {
	t.Helper()
	doc := gltf.NewDocument()
	attrs := gltf.PrimitiveAttributes{
		gltf.POSITION: modeler.WritePosition(doc, [][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}),
	}
	if attributes != nil {
		attributes(doc, attrs)
	}
	idx := modeler.WriteIndices(doc, indices)
	doc.Meshes = append(doc.Meshes, &gltf.Mesh{Name: "m", Primitives: []*gltf.Primitive{{
		Attributes: attrs, Indices: gltf.Index(idx),
	}}})
	doc.Nodes = append(doc.Nodes, &gltf.Node{Name: "n", Mesh: gltf.Index(0)})
	doc.Scenes = append(doc.Scenes, &gltf.Scene{Nodes: []int{0}})
	doc.Scene = gltf.Index(0)
	var buf bytes.Buffer
	if err := gltf.NewEncoder(&buf).Encode(doc); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestLoadGLTFDataInvalidVertexData(t *testing.T) {
	triangle := []uint16{0, 1, 2}
	cases := []struct {
		name       string
		indices    []uint16
		attributes func(doc *gltf.Document, attrs gltf.PrimitiveAttributes)
	}{
		{"index out of bounds", []uint16{0, 1, 5}, nil},
		{"TEXCOORD_0 longer than POSITION", triangle, func(doc *gltf.Document, attrs gltf.PrimitiveAttributes) {
			attrs[gltf.TEXCOORD_0] = modeler.WriteTextureCoord(doc, [][2]float32{{0, 0}, {1, 0}, {0, 1}, {1, 1}})
		}},
		{"NORMAL longer than POSITION", triangle, func(doc *gltf.Document, attrs gltf.PrimitiveAttributes) {
			attrs[gltf.NORMAL] = modeler.WriteNormal(doc, [][3]float32{{0, 0, 1}, {0, 0, 1}, {0, 0, 1}, {0, 0, 1}})
		}},
		{"COLOR_0 longer than POSITION", triangle, func(doc *gltf.Document, attrs gltf.PrimitiveAttributes) {
			attrs["COLOR_0"] = modeler.WriteColor(doc, [][4]uint8{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {255, 255, 255, 255}})
		}},
		{"WEIGHTS_0 longer than POSITION", triangle, func(doc *gltf.Document, attrs gltf.PrimitiveAttributes) {
			attrs[gltf.WEIGHTS_0] = modeler.WriteWeights(doc, [][4]float32{{1, 0, 0, 0}, {1, 0, 0, 0}, {1, 0, 0, 0}, {1, 0, 0, 0}})
			attrs[gltf.JOINTS_0] = modeler.WriteJoints(doc, [][4]uint16{{0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}})
		}},
		{"JOINTS_0 shorter than WEIGHTS_0", triangle, func(doc *gltf.Document, attrs gltf.PrimitiveAttributes) {
			attrs[gltf.WEIGHTS_0] = modeler.WriteWeights(doc, [][4]float32{{1, 0, 0, 0}, {1, 0, 0, 0}, {1, 0, 0, 0}})
			attrs[gltf.JOINTS_0] = modeler.WriteJoints(doc, [][4]uint16{{0, 0, 0, 0}, {0, 0, 0, 0}})
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := vertexDataGLTF(t, c.indices, c.attributes)
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("LoadGLTFData panicked: %v", r)
				}
			}()
			if _, err := LoadGLTFData(bytes.NewReader(data), nil); err == nil {
				t.Fatal("LoadGLTFData returned no error")
			}
		})
	}
}

func TestLoadGLTFDataValidVertexData(t *testing.T) {
	data := vertexDataGLTF(t, []uint16{0, 1, 2}, func(doc *gltf.Document, attrs gltf.PrimitiveAttributes) {
		attrs[gltf.TEXCOORD_0] = modeler.WriteTextureCoord(doc, [][2]float32{{0, 0}, {1, 0}, {0, 1}})
		attrs[gltf.NORMAL] = modeler.WriteNormal(doc, [][3]float32{{0, 0, 1}, {0, 0, 1}, {0, 0, 1}})
		attrs["COLOR_0"] = modeler.WriteColor(doc, [][4]uint8{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}})
	})
	if _, err := LoadGLTFData(bytes.NewReader(data), nil); err != nil {
		t.Fatal(err)
	}
}
