// Package target validates whether NIR can be emitted for a native target.
package target

import (
	"fmt"
	"strings"

	"github.com/odvcencio/gosx/nir"
)

type Target string

const (
	IOS     Target = "ios"
	Android Target = "android"
)

func Parse(name string) (Target, error) {
	switch Target(strings.ToLower(strings.TrimSpace(name))) {
	case IOS:
		return IOS, nil
	case Android:
		return Android, nil
	default:
		return "", fmt.Errorf("unknown target: %s (supported: ios, android)", name)
	}
}

type Diagnostic struct {
	Target    Target
	Component string
	Message   string
	Span      nir.Span
}

type Diagnostics []Diagnostic

func (d Diagnostics) Error() string {
	if len(d) == 0 {
		return ""
	}
	if len(d) == 1 {
		return d[0].String()
	}
	lines := make([]string, len(d))
	for i := range d {
		lines[i] = d[i].String()
	}
	return strings.Join(lines, "\n")
}

func (d Diagnostic) String() string {
	where := string(d.Target)
	if d.Component != "" {
		where += " " + d.Component
	}
	if d.Span.StartLine > 0 {
		where += fmt.Sprintf(":%d:%d", d.Span.StartLine, d.Span.StartCol)
	}
	return where + ": " + d.Message
}

// Validate returns diagnostics for NIR that cannot be emitted faithfully for a
// native target. Emitters and build tooling call this before writing code so
// unsupported features fail instead of becoming comments in generated source.
func Validate(mod *nir.Module, tgt Target) error {
	if _, err := Parse(string(tgt)); err != nil {
		return err
	}
	if mod == nil {
		return fmt.Errorf("nil NIR module")
	}
	v := validator{
		target:     tgt,
		components: make(map[string]bool),
	}
	for _, c := range mod.Components {
		if c != nil {
			v.components[c.Name] = true
		}
	}
	for _, c := range mod.Components {
		if c == nil {
			continue
		}
		v.component = c.Name
		v.view(c.Body)
	}
	if len(v.diagnostics) > 0 {
		return v.diagnostics
	}
	return nil
}

type validator struct {
	target      Target
	component   string
	components  map[string]bool
	inScene3D   bool
	diagnostics Diagnostics
}

func (v *validator) view(view nir.View) {
	switch n := view.(type) {
	case *nir.Element:
		v.element(n)
	case *nir.Conditional:
		for _, child := range n.Then {
			v.view(child)
		}
		for _, child := range n.Else {
			v.view(child)
		}
	case *nir.ComponentRef:
		v.componentRef(n)
	case *nir.Loop:
		for _, child := range n.Body {
			v.view(child)
		}
		for _, child := range n.Empty {
			v.view(child)
		}
	case *nir.Text, *nir.ExprHole, nil:
		return
	default:
		v.add(nir.Span{}, "unsupported NIR view node")
	}
}

func (v *validator) element(n *nir.Element) {
	if IsScene3DTag(n.Tag) {
		if !Scene3DNativeTagSupported(n.Tag) {
			v.add(n.Span, fmt.Sprintf("Scene3D native backend does not support <%s> yet", Scene3DDisplayName(n.Tag)))
			return
		}
		if !v.inScene3D && normalizeScene3DTag(n.Tag) != "scene3d" {
			v.add(n.Span, fmt.Sprintf("Scene3D tag <%s> must be inside <Scene3D>", Scene3DDisplayName(n.Tag)))
			return
		}
		wasInScene3D := v.inScene3D
		if normalizeScene3DTag(n.Tag) == "scene3d" {
			v.inScene3D = true
			v.scene3DBackend(n.Attrs, n.Span)
		}
		defer func() {
			v.inScene3D = wasInScene3D
		}()
		if n.Scene3D != nil {
			for _, item := range n.Scene3D.Items {
				v.scene3DItem(item)
			}
		}
		for _, child := range n.Children {
			v.view(child)
		}
		return
	}
	if !tagSupported(v.target, n.Tag) {
		v.add(n.Span, fmt.Sprintf("unsupported native tag <%s>", n.Tag))
		return
	}
	v.handlers(n.Tag, n.Handlers, n.Span)
	for _, child := range n.Children {
		v.view(child)
	}
}

func (v *validator) scene3DItem(n nir.Scene3DItem) {
	if !IsScene3DTag(n.Tag) {
		v.add(n.Span, fmt.Sprintf("unsupported Scene3D item <%s>", n.Tag))
		return
	}
	if !Scene3DNativeTagSupported(n.Tag) {
		v.add(n.Span, fmt.Sprintf("Scene3D native backend does not support <%s> yet", Scene3DDisplayName(n.Tag)))
	}
}

func (v *validator) componentRef(n *nir.ComponentRef) {
	if IsScene3DTag(n.Name) {
		if !Scene3DNativeTagSupported(n.Name) {
			v.add(n.Span, fmt.Sprintf("Scene3D native backend does not support <%s> yet", Scene3DDisplayName(n.Name)))
			return
		}
		if !v.inScene3D && normalizeScene3DTag(n.Name) != "scene3d" {
			v.add(n.Span, fmt.Sprintf("Scene3D tag <%s> must be inside <Scene3D>", Scene3DDisplayName(n.Name)))
			return
		}
		for _, child := range n.Children {
			v.view(child)
		}
		return
	}
	if !v.components[n.Name] {
		v.add(n.Span, fmt.Sprintf("unsupported native component <%s>", n.Name))
		return
	}
	for _, child := range n.Children {
		v.view(child)
	}
}

func (v *validator) handlers(tag string, handlers []nir.Handler, span nir.Span) {
	for _, handler := range handlers {
		if handlerSupported(tag, handler.Event) {
			continue
		}
		event := handler.Event
		if event == "" {
			event = "<empty>"
		}
		v.add(span, fmt.Sprintf("unsupported <%s> handler %q", tag, event))
	}
}

func (v *validator) scene3DBackend(attrs []nir.Attr, span nir.Span) {
	expr := attrExpr(attrs, "backend")
	if expr == nil {
		return
	}
	if expr.Kind != "literal" || expr.Literal == nil || expr.Literal.Type != "string" {
		v.add(span, `Scene3D backend must be a literal string`)
		return
	}
	backend := strings.ToLower(strings.TrimSpace(expr.Literal.Value))
	if backend == "" {
		return
	}
	switch v.target {
	case IOS:
		switch backend {
		case "native", "scenekit", "canvas":
			return
		}
		v.add(span, fmt.Sprintf("unsupported Scene3D backend %q for ios (supported: native, scenekit, canvas)", expr.Literal.Value))
	case Android:
		switch backend {
		case "native", "opengl", "gles", "canvas":
			return
		}
		v.add(span, fmt.Sprintf("unsupported Scene3D backend %q for android (supported: native, opengl, gles, canvas)", expr.Literal.Value))
	}
}

func (v *validator) add(span nir.Span, message string) {
	v.diagnostics = append(v.diagnostics, Diagnostic{
		Target:    v.target,
		Component: v.component,
		Message:   message,
		Span:      span,
	})
}

func tagSupported(tgt Target, tag string) bool {
	switch tag {
	case "text", "textinput", "textarea", "checkbox", "select", "option", "vstack", "hstack", "button":
		return true
	case "view":
		return tgt == IOS
	default:
		return false
	}
}

func handlerSupported(tag, event string) bool {
	switch tag {
	case "textinput":
		return event == "input" || event == "change" || event == "key"
	case "textarea":
		return event == "input" || event == "change"
	case "checkbox":
		return event == "input" || event == "change"
	case "select":
		return event == "change"
	default:
		return event == "tap"
	}
}

func IsScene3DTag(tag string) bool {
	switch normalizeScene3DTag(tag) {
	case "scene3d", "camera", "environment", "mesh", "model", "points", "html",
		"instancedmesh", "computeparticles", "directionallight", "pointlight",
		"ambientlight", "spotlight", "hemispherelight", "material",
		"postfx.bloom", "postfx.vignette", "postfx.colorgrading", "postfx.tonemap":
		return true
	default:
		return false
	}
}

func Scene3DDisplayName(tag string) string {
	switch normalizeScene3DTag(tag) {
	case "scene3d":
		return "Scene3D"
	case "instancedmesh":
		return "InstancedMesh"
	case "computeparticles":
		return "ComputeParticles"
	case "directionallight":
		return "DirectionalLight"
	case "pointlight":
		return "PointLight"
	case "ambientlight":
		return "AmbientLight"
	case "spotlight":
		return "SpotLight"
	case "hemispherelight":
		return "HemisphereLight"
	case "postfx.bloom":
		return "PostFX.Bloom"
	case "postfx.vignette":
		return "PostFX.Vignette"
	case "postfx.colorgrading":
		return "PostFX.ColorGrading"
	case "postfx.tonemap":
		return "PostFX.Tonemap"
	default:
		return tag
	}
}

func Scene3DNativeTagSupported(tag string) bool {
	switch normalizeScene3DTag(tag) {
	case "scene3d", "camera", "environment", "mesh", "model", "points",
		"html", "instancedmesh", "computeparticles", "directionallight", "pointlight", "ambientlight",
		"spotlight", "hemispherelight", "material",
		"postfx.bloom", "postfx.vignette", "postfx.colorgrading", "postfx.tonemap":
		return true
	default:
		return false
	}
}

func normalizeScene3DTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

func attrExpr(attrs []nir.Attr, name string) *nir.RxExpr {
	for i := len(attrs) - 1; i >= 0; i-- {
		if attrs[i].Name == name {
			return &attrs[i].Value
		}
	}
	return nil
}
