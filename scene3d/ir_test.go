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
			{
				Tag: "instancedMesh",
				Attrs: []nir.Attr{
					literalAttr("id", "string", "asteroids"),
					literalAttr("kind", "string", "box"),
					literalAttr("count", "int", "2"),
					literalAttr("width", "float", "0.8"),
					literalAttr("height", "float", "0.5"),
					literalAttr("depth", "float", "0.4"),
					literalAttr("color", "string", "#f7c76b"),
					literalAttr("transforms", "string", "1,0,0,0,0,1,0,0,0,0,1,0,-1,0,0,1,1,0,0,0,0,1,0,0,0,0,1,0,1,0,0,1"),
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
	if len(ir.Materials) != 2 || ir.Materials[0].Color != "#8de1ff" || ir.Materials[1].Color != "#f7c76b" {
		t.Fatalf("materials = %+v", ir.Materials)
	}
	if len(ir.Nodes) != 3 || ir.Nodes[0].Kind != "mesh" || ir.Nodes[0].Mesh.Kind != "box" || ir.Nodes[1].Kind != "points" || ir.Nodes[2].Kind != "instanced-mesh" {
		t.Fatalf("nodes = %+v", ir.Nodes)
	}
	if ir.Nodes[1].Points.Count != 2 || ir.Nodes[1].Points.Size != 0.5 {
		t.Fatalf("points = %+v", ir.Nodes[1].Points)
	}
	if ir.Nodes[2].InstancedMesh.Count != 2 || len(ir.Nodes[2].InstancedMesh.Transforms) != 32 {
		t.Fatalf("instanced mesh = %+v", ir.Nodes[2].InstancedMesh)
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

func TestCanonicalIRLowersPostFXEffects(t *testing.T) {
	sceneElement := &nir.Element{
		Tag: "scene3d",
		Scene3D: &nir.Scene3DPayload{Items: []nir.Scene3DItem{
			{
				Tag: "postFX.Bloom",
				Attrs: []nir.Attr{
					literalAttr("threshold", "float", "0.72"),
					literalAttr("intensity", "float", "1.4"),
					literalAttr("radius", "float", "0.25"),
				},
			},
			{
				Tag: "PostFX.ColorGrading",
				Attrs: []nir.Attr{
					literalAttr("saturation", "float", "1.1"),
					literalAttr("contrast", "float", "1.05"),
					literalAttr("exposure", "float", "0.08"),
				},
			},
			{
				Tag: "PostFX.Tonemap",
				Attrs: []nir.Attr{
					literalAttr("mode", "string", "aces"),
					literalAttr("exposure", "float", "1.0"),
				},
			},
		}},
	}

	ir, err := CanonicalIR(sceneElement)
	if err != nil {
		t.Fatalf("CanonicalIR: %v", err)
	}
	if len(ir.PostFX) != 3 {
		t.Fatalf("postFX len = %d, want 3", len(ir.PostFX))
	}
	if ir.PostFX[0].Kind != "bloom" || ir.PostFX[0].Threshold != 0.72 || ir.PostFX[0].Intensity != 1.4 || ir.PostFX[0].Radius != 0.25 {
		t.Fatalf("bloom effect = %+v", ir.PostFX[0])
	}
	if ir.PostFX[1].Kind != "colorGrade" || ir.PostFX[1].Saturation != 1.1 || ir.PostFX[1].Contrast != 1.05 || ir.PostFX[1].Exposure != 0.08 {
		t.Fatalf("color grading effect = %+v", ir.PostFX[1])
	}
	if ir.PostFX[2].Kind != "toneMapping" || ir.PostFX[2].Mode != "aces" || ir.PostFX[2].Exposure != 1.0 {
		t.Fatalf("tonemap effect = %+v", ir.PostFX[2])
	}
}

func TestCanonicalIRRejectsInstancedTransformCountMismatch(t *testing.T) {
	sceneElement := &nir.Element{
		Tag: "scene3d",
		Scene3D: &nir.Scene3DPayload{Items: []nir.Scene3DItem{{
			Tag: "instancedMesh",
			Attrs: []nir.Attr{
				literalAttr("count", "int", "2"),
				literalAttr("transforms", "string", "1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1"),
			},
		}}},
	}
	_, err := CanonicalIR(sceneElement)
	if err == nil {
		t.Fatal("expected conformance error")
	}
	if !strings.Contains(err.Error(), "count=2 but transforms describe 1 instances") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func literalAttr(name, typ, value string) nir.Attr {
	return nir.Attr{
		Name:  name,
		Value: nir.RxExpr{Kind: "literal", Literal: &nir.Literal{Type: typ, Value: value}},
	}
}
