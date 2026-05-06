package target

import (
	"strings"
	"testing"

	"github.com/odvcencio/gosx/nir"
)

func TestValidateScene3DStaticSurfaceAccepted(t *testing.T) {
	mod := &nir.Module{
		Components: []*nir.Component{{
			Name: "SceneDemo",
			Body: &nir.Element{
				Tag: "scene3d",
				Scene3D: &nir.Scene3DPayload{
					Items: []nir.Scene3DItem{
						{Tag: "camera"},
						{Tag: "mesh"},
						{Tag: "instancedMesh"},
						{Tag: "points"},
						{Tag: "postFX.Bloom"},
						{Tag: "computeParticles"},
						{Tag: "html"},
					},
				},
			},
		}},
	}
	if err := Validate(mod, IOS); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidateScene3DUnsupportedTagReportsDiagnostic(t *testing.T) {
	mod := &nir.Module{
		Components: []*nir.Component{{
			Name: "SceneDemo",
			Body: &nir.Element{
				Tag: "scene3d",
				Scene3D: &nir.Scene3DPayload{
					Items: []nir.Scene3DItem{
						{Tag: "physicsWorld"},
					},
				},
			},
		}},
	}
	err := Validate(mod, IOS)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "unsupported Scene3D item <physicsWorld>") {
		t.Fatalf("expected Scene3D diagnostic, got %v", err)
	}
}

func TestValidateScene3DChildOutsideSceneReportsDiagnostic(t *testing.T) {
	mod := &nir.Module{
		Components: []*nir.Component{{
			Name: "CameraOnly",
			Body: &nir.Element{Tag: "camera"},
		}},
	}
	err := Validate(mod, Android)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "Scene3D tag <camera> must be inside <Scene3D>") {
		t.Fatalf("expected containment diagnostic, got %v", err)
	}
}

func TestValidateUnknownComponentReportsDiagnostic(t *testing.T) {
	mod := &nir.Module{
		Components: []*nir.Component{{
			Name: "Profile",
			Body: &nir.ComponentRef{Name: "MissingBadge"},
		}},
	}
	err := Validate(mod, Android)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "unsupported native component <MissingBadge>") {
		t.Fatalf("expected unknown component diagnostic, got %v", err)
	}
}
