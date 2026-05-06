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
				Children: []nir.View{
					&nir.Element{Tag: "camera"},
					&nir.Element{Tag: "mesh"},
					&nir.Element{Tag: "points"},
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
				Children: []nir.View{
					&nir.Element{Tag: "computeParticles"},
				},
			},
		}},
	}
	err := Validate(mod, IOS)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "Scene3D native backend does not support <ComputeParticles> yet") {
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
