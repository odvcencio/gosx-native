package target

import (
	"strings"
	"testing"

	"github.com/odvcencio/gosx/nir"
)

func TestValidateScene3DTagReportsMissingBackend(t *testing.T) {
	mod := &nir.Module{
		Components: []*nir.Component{{
			Name: "SceneDemo",
			Body: &nir.Element{Tag: "scene3d"},
		}},
	}
	err := Validate(mod, IOS)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "Scene3D native backend is not implemented for <Scene3D>") {
		t.Fatalf("expected Scene3D diagnostic, got %v", err)
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
