package scene3d

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/odvcencio/gosx/nir"
)

func TestRenderSignatureIncludesNodesHTMLAndPostFX(t *testing.T) {
	sceneElement := &nir.Element{
		Tag: "scene3d",
		Scene3D: &nir.Scene3DPayload{Items: []nir.Scene3DItem{
			{
				Tag: "environment",
				Attrs: []nir.Attr{
					literalAttr("background", "string", "#081119"),
				},
			},
			{
				Tag: "mesh",
				Attrs: []nir.Attr{
					literalAttr("id", "string", "hero"),
					literalAttr("kind", "string", "box"),
					literalAttr("width", "float", "1.4"),
					literalAttr("height", "float", "1"),
					literalAttr("depth", "float", "0.7"),
					literalAttr("x", "float", "1"),
					literalAttr("y", "float", "0.5"),
					literalAttr("color", "string", "#8de1ff"),
				},
			},
			{
				Tag: "html",
				Attrs: []nir.Attr{
					literalAttr("id", "string", "hud"),
					literalAttr("html", "string", "<aside><strong>Hull</strong><span>&amp; stable</span></aside>"),
					literalAttr("class", "string", "scene-hud"),
					literalAttr("x", "float", "0"),
					literalAttr("y", "float", "1.1"),
					literalAttr("z", "float", "0.2"),
					literalAttr("width", "float", "1.6"),
					literalAttr("height", "float", "0.8"),
					literalAttr("opacity", "float", "0.95"),
					literalAttr("pointerEvents", "string", "auto"),
				},
			},
			{
				Tag: "postFX.Bloom",
				Attrs: []nir.Attr{
					literalAttr("threshold", "float", "0.72"),
					literalAttr("intensity", "float", "1.4"),
					literalAttr("radius", "float", "0.25"),
				},
			},
		}},
	}

	ir, err := CanonicalIR(sceneElement)
	if err != nil {
		t.Fatalf("CanonicalIR: %v", err)
	}
	signature := BuildRenderSignature(ir)
	if signature.Version != 1 || signature.Background != "#081119" {
		t.Fatalf("signature header = %+v", signature)
	}
	if len(signature.Commands) != 3 {
		t.Fatalf("commands = %+v", signature.Commands)
	}

	mesh := signature.Commands[0]
	if mesh.Op != "node" || mesh.ID != "hero" || mesh.Tag != "mesh" || mesh.Shape != "roundRect" || mesh.Color != "#8de1ff" {
		t.Fatalf("mesh command = %+v", mesh)
	}
	if mesh.Size == nil || mesh.Size.Width != 1.4 || mesh.Size.Height != 1 || mesh.Size.Depth != 0.7 {
		t.Fatalf("mesh size = %+v", mesh.Size)
	}
	if mesh.Screen == nil || !approx(mesh.Screen.X, 298.64) || !approx(mesh.Screen.Y, 135.6) {
		t.Fatalf("mesh screen = %+v", mesh.Screen)
	}

	postfx := signature.Commands[1]
	if postfx.Op != "postfx" || postfx.Kind != "bloom" || postfx.Effect == nil {
		t.Fatalf("postfx command = %+v", postfx)
	}
	if !approx(postfx.Effect.Alpha, 0.224) || postfx.Effect.LineWidth != 8 {
		t.Fatalf("postfx effect = %+v", postfx.Effect)
	}

	overlay := signature.Commands[2]
	if overlay.Op != "htmlText" || overlay.ID != "hud" || overlay.Text != "Hull & stable" || overlay.PointerEvents != "auto" {
		t.Fatalf("html command = %+v", overlay)
	}
	if overlay.Screen == nil || !approx(overlay.Screen.X, 320) || !approx(overlay.Screen.Y, 141.2) {
		t.Fatalf("html screen = %+v", overlay.Screen)
	}
}

func TestRenderSignatureJSONIsDeterministic(t *testing.T) {
	sceneElement := &nir.Element{
		Tag: "scene3d",
		Scene3D: &nir.Scene3DPayload{Items: []nir.Scene3DItem{{
			Tag: "computeParticles",
			Attrs: []nir.Attr{
				literalAttr("id", "string", "sparks"),
				literalAttr("count", "int", "128"),
				literalAttr("kind", "string", "spiral"),
				literalAttr("color", "string", "#ffd48f"),
				literalAttr("size", "float", "0.12"),
			},
		}}},
	}
	ir, err := CanonicalIR(sceneElement)
	if err != nil {
		t.Fatalf("CanonicalIR: %v", err)
	}
	first, err := RenderSignature(ir)
	if err != nil {
		t.Fatalf("RenderSignature first: %v", err)
	}
	second, err := RenderSignature(ir)
	if err != nil {
		t.Fatalf("RenderSignature second: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("render signature was not deterministic\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if !strings.Contains(string(first), `"visibleCount": 48`) || !strings.Contains(string(first), `"radius": 2`) {
		t.Fatalf("expected compute placeholder signature, got:\n%s", first)
	}
}

func approx(got, want float64) bool {
	return math.Abs(got-want) < 0.0001
}
