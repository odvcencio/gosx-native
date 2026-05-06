package scene3d

import (
	"strings"
	"testing"

	"github.com/odvcencio/gosx/nir"
)

func TestCanonicalIRLowersTypedScenePayload(t *testing.T) {
	sceneElement := &nir.Element{
		Tag: "scene3d",
		Scene3D: &nir.Scene3DPayload{Items: []nir.Scene3DItem{
			{
				Tag: "camera",
				Attrs: []nir.Attr{
					literalAttr("z", "int", "7"),
					literalAttr("near", "float", "0.1"),
				},
			},
			{
				Tag: "directionalLight",
				Attrs: []nir.Attr{
					literalAttr("id", "string", "sun"),
					literalAttr("intensity", "float", "1.2"),
				},
			},
			{
				Tag: "mesh",
				Attrs: []nir.Attr{
					literalAttr("id", "string", "hero"),
					literalAttr("kind", "string", "box"),
					literalAttr("width", "float", "1.8"),
					literalAttr("color", "string", "#8de1ff"),
				},
			},
			{
				Tag: "points",
				Attrs: []nir.Attr{
					literalAttr("id", "string", "stars"),
					literalAttr("count", "int", "2"),
					literalAttr("size", "float", "0.5"),
				},
			},
		}},
	}

	ir, err := CanonicalIR(sceneElement)
	if err != nil {
		t.Fatalf("CanonicalIR: %v", err)
	}
	if ir.Camera.Kind != "perspective" || ir.Camera.Z != 7 || ir.Camera.Near != 0.1 {
		t.Fatalf("camera = %+v", ir.Camera)
	}
	if len(ir.Lights) != 1 || ir.Lights[0].Kind != "directional" || ir.Lights[0].ID != "sun" {
		t.Fatalf("lights = %+v", ir.Lights)
	}
	if len(ir.Materials) != 1 || ir.Materials[0].Color != "#8de1ff" {
		t.Fatalf("materials = %+v", ir.Materials)
	}
	if len(ir.Nodes) != 2 || ir.Nodes[0].Kind != "mesh" || ir.Nodes[0].Mesh.Kind != "box" || ir.Nodes[1].Kind != "points" {
		t.Fatalf("nodes = %+v", ir.Nodes)
	}
	if ir.Nodes[1].Points.Count != 2 || ir.Nodes[1].Points.Size != 0.5 {
		t.Fatalf("points = %+v", ir.Nodes[1].Points)
	}
}

func TestCanonicalIRRejectsDynamicConsumedAttribute(t *testing.T) {
	sceneElement := &nir.Element{
		Tag: "scene3d",
		Scene3D: &nir.Scene3DPayload{Items: []nir.Scene3DItem{{
			Tag: "mesh",
			Attrs: []nir.Attr{{
				Name:  "width",
				Value: nir.RxExpr{Kind: "ref", Ref: "props.width"},
			}},
		}}},
	}
	_, err := CanonicalIR(sceneElement)
	if err == nil {
		t.Fatal("expected conformance error")
	}
	if !strings.Contains(err.Error(), `attribute "width" must be literal`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func literalAttr(name, typ, value string) nir.Attr {
	return nir.Attr{
		Name:  name,
		Value: nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: typ, Value: value}},
	}
}
