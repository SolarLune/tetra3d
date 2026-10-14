package tetra3d

import (
	"bytes"
	"testing"

	"github.com/qmuntal/gltf"
)

// textureGLTF builds a glTF with one triangle and one material that uses the given base colour texture index.
// The setup function can add textures and images.
func textureGLTF(t *testing.T, index int, setup func(doc *gltf.Document)) []byte {
	t.Helper()
	return vertexDataGLTF(t, []uint16{0, 1, 2}, func(doc *gltf.Document, attrs gltf.PrimitiveAttributes) {
		doc.Materials = append(doc.Materials, &gltf.Material{
			Name:                 "mat",
			PBRMetallicRoughness: &gltf.PBRMetallicRoughness{BaseColorTexture: &gltf.TextureInfo{Index: index}},
		})
		if setup != nil {
			setup(doc)
		}
	})
}

func TestLoadGLTFDataInvalidTexture(t *testing.T) {
	cases := []struct {
		name  string
		index int
		setup func(doc *gltf.Document)
	}{
		{"texture index out of bounds", 4294967295, nil},
		{"negative texture index", -1, nil},
		{"texture without source", 0, func(doc *gltf.Document) {
			doc.Textures = append(doc.Textures, &gltf.Texture{})
		}},
		{"texture source out of bounds", 0, func(doc *gltf.Document) {
			doc.Textures = append(doc.Textures, &gltf.Texture{Source: gltf.Index(1)})
			doc.Images = append(doc.Images, &gltf.Image{URI: "texture.png"})
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := textureGLTF(t, c.index, c.setup)
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

func TestLoadGLTFDataValidTexture(t *testing.T) {
	data := textureGLTF(t, 0, func(doc *gltf.Document) {
		doc.Textures = append(doc.Textures, &gltf.Texture{Source: gltf.Index(0)})
		doc.Images = append(doc.Images, &gltf.Image{URI: "texture.png"})
	})
	library, err := LoadGLTFData(bytes.NewReader(data), nil)
	if err != nil {
		t.Fatal(err)
	}
	if mat := library.MaterialByName("mat"); mat == nil || mat.TexturePath != "texture.png" {
		t.Fatalf("material texture path not loaded: %+v", mat)
	}
}
