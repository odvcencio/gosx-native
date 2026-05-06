// Package scene3d bridges gosx-native's typed NIR payloads to the canonical
// gosx Scene3D IR contract.
package scene3d

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/odvcencio/gosx/nir"
	"github.com/odvcencio/gosx/scene"
)

// CanonicalIR lowers a NIR <Scene3D> element into gosx's canonical scene.IR.
func CanonicalIR(element *nir.Element) (scene.IR, error) {
	if element == nil {
		return scene.IR{}, fmt.Errorf("nil scene3d element")
	}
	if normalizeTag(element.Tag) != "scene3d" {
		return scene.IR{}, fmt.Errorf("expected scene3d element, got <%s>", element.Tag)
	}
	props := scene.Scene3DProps{
		Camera: scene.IRCamera{Kind: "perspective"},
	}
	if background, ok, err := stringAttr(element.Attrs, "background"); err != nil {
		return scene.IR{}, err
	} else if ok {
		props.Environment.Background = background
	}
	for _, item := range scene3DItems(element) {
		if err := lowerItem(&props, item); err != nil {
			return scene.IR{}, err
		}
	}
	ir := scene.LowerScene3D(props)
	if err := ir.Validate(); err != nil {
		return scene.IR{}, err
	}
	return ir, nil
}

func scene3DItems(element *nir.Element) []nir.Scene3DItem {
	if element.Scene3D != nil {
		return element.Scene3D.Items
	}
	items := make([]nir.Scene3DItem, 0, len(element.Children))
	for _, child := range element.Children {
		childElement, ok := child.(*nir.Element)
		if !ok {
			continue
		}
		items = append(items, nir.Scene3DItem{
			Tag:   childElement.Tag,
			Attrs: childElement.Attrs,
			Span:  childElement.Span,
		})
	}
	return items
}

func lowerItem(props *scene.Scene3DProps, item nir.Scene3DItem) error {
	switch normalizeTag(item.Tag) {
	case "camera":
		camera, err := lowerCamera(item.Attrs)
		if err != nil {
			return err
		}
		props.Camera = camera
	case "environment":
		env, err := lowerEnvironment(item.Attrs)
		if err != nil {
			return err
		}
		props.Environment = mergeEnvironment(props.Environment, env)
	case "directionallight", "pointlight", "ambientlight", "spotlight", "hemispherelight":
		light, err := lowerLight(item.Tag, item.Attrs)
		if err != nil {
			return err
		}
		props.Lights = append(props.Lights, light)
	case "mesh":
		node, material, err := lowerMesh(item.Attrs, false)
		if err != nil {
			return err
		}
		node.MaterialIndex = appendMaterial(&props.Materials, material)
		props.Nodes = append(props.Nodes, node)
	case "model":
		node, material, err := lowerMesh(item.Attrs, true)
		if err != nil {
			return err
		}
		node.MaterialIndex = appendMaterial(&props.Materials, material)
		props.Nodes = append(props.Nodes, node)
	case "instancedmesh":
		node, material, err := lowerInstancedMesh(item.Attrs)
		if err != nil {
			return err
		}
		node.MaterialIndex = appendMaterial(&props.Materials, material)
		props.Nodes = append(props.Nodes, node)
	case "points":
		node, err := lowerPoints(item.Attrs)
		if err != nil {
			return err
		}
		props.Nodes = append(props.Nodes, node)
	default:
		return fmt.Errorf("Scene3D conformance does not support <%s> yet", item.Tag)
	}
	return nil
}

func lowerCamera(attrs []nir.Attr) (scene.IRCamera, error) {
	camera := scene.IRCamera{Kind: "perspective"}
	var err error
	if camera.Kind, err = stringAttrDefault(attrs, "kind", camera.Kind); err != nil {
		return scene.IRCamera{}, err
	}
	if camera.X, err = floatAttrDefault(attrs, "x", camera.X); err != nil {
		return scene.IRCamera{}, err
	}
	if camera.Y, err = floatAttrDefault(attrs, "y", camera.Y); err != nil {
		return scene.IRCamera{}, err
	}
	if camera.Z, err = floatAttrDefault(attrs, "z", camera.Z); err != nil {
		return scene.IRCamera{}, err
	}
	if camera.RotationX, err = floatAttrDefault(attrs, "rotationX", camera.RotationX); err != nil {
		return scene.IRCamera{}, err
	}
	if camera.RotationY, err = floatAttrDefault(attrs, "rotationY", camera.RotationY); err != nil {
		return scene.IRCamera{}, err
	}
	if camera.RotationZ, err = floatAttrDefault(attrs, "rotationZ", camera.RotationZ); err != nil {
		return scene.IRCamera{}, err
	}
	if camera.FOV, err = floatAttrDefault(attrs, "fov", camera.FOV); err != nil {
		return scene.IRCamera{}, err
	}
	if camera.Near, err = floatAttrDefault(attrs, "near", camera.Near); err != nil {
		return scene.IRCamera{}, err
	}
	if camera.Far, err = floatAttrDefault(attrs, "far", camera.Far); err != nil {
		return scene.IRCamera{}, err
	}
	if camera.Zoom, err = floatAttrDefault(attrs, "zoom", camera.Zoom); err != nil {
		return scene.IRCamera{}, err
	}
	return camera, nil
}

func lowerEnvironment(attrs []nir.Attr) (scene.IREnvironment, error) {
	var env scene.IREnvironment
	var err error
	if env.AmbientColor, err = stringAttrDefault(attrs, "ambientColor", env.AmbientColor); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.AmbientIntensity, err = floatAttrDefault(attrs, "ambientIntensity", env.AmbientIntensity); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.SkyColor, err = stringAttrDefault(attrs, "skyColor", env.SkyColor); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.SkyIntensity, err = floatAttrDefault(attrs, "skyIntensity", env.SkyIntensity); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.GroundColor, err = stringAttrDefault(attrs, "groundColor", env.GroundColor); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.GroundIntensity, err = floatAttrDefault(attrs, "groundIntensity", env.GroundIntensity); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.EnvMap, err = stringAttrDefault(attrs, "envMap", env.EnvMap); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.EnvIntensity, err = floatAttrDefault(attrs, "envIntensity", env.EnvIntensity); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.EnvRotation, err = floatAttrDefault(attrs, "envRotation", env.EnvRotation); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.Background, err = stringAttrDefault(attrs, "background", env.Background); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.Exposure, err = floatAttrDefault(attrs, "exposure", env.Exposure); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.ToneMapping, err = stringAttrDefault(attrs, "toneMapping", env.ToneMapping); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.FogColor, err = stringAttrDefault(attrs, "fogColor", env.FogColor); err != nil {
		return scene.IREnvironment{}, err
	}
	if env.FogDensity, err = floatAttrDefault(attrs, "fogDensity", env.FogDensity); err != nil {
		return scene.IREnvironment{}, err
	}
	return env, nil
}

func mergeEnvironment(base, overlay scene.IREnvironment) scene.IREnvironment {
	if overlay.AmbientColor != "" {
		base.AmbientColor = overlay.AmbientColor
	}
	if overlay.AmbientIntensity != 0 {
		base.AmbientIntensity = overlay.AmbientIntensity
	}
	if overlay.SkyColor != "" {
		base.SkyColor = overlay.SkyColor
	}
	if overlay.SkyIntensity != 0 {
		base.SkyIntensity = overlay.SkyIntensity
	}
	if overlay.GroundColor != "" {
		base.GroundColor = overlay.GroundColor
	}
	if overlay.GroundIntensity != 0 {
		base.GroundIntensity = overlay.GroundIntensity
	}
	if overlay.EnvMap != "" {
		base.EnvMap = overlay.EnvMap
	}
	if overlay.EnvIntensity != 0 {
		base.EnvIntensity = overlay.EnvIntensity
	}
	if overlay.EnvRotation != 0 {
		base.EnvRotation = overlay.EnvRotation
	}
	if overlay.Background != "" {
		base.Background = overlay.Background
	}
	if overlay.Exposure != 0 {
		base.Exposure = overlay.Exposure
	}
	if overlay.ToneMapping != "" {
		base.ToneMapping = overlay.ToneMapping
	}
	if overlay.FogColor != "" {
		base.FogColor = overlay.FogColor
	}
	if overlay.FogDensity != 0 {
		base.FogDensity = overlay.FogDensity
	}
	return base
}

func lowerLight(tag string, attrs []nir.Attr) (scene.IRLight, error) {
	light := scene.IRLight{Kind: lightKind(tag)}
	var err error
	if light.ID, err = stringAttrDefault(attrs, "id", light.ID); err != nil {
		return scene.IRLight{}, err
	}
	if light.Color, err = stringAttrDefault(attrs, "color", light.Color); err != nil {
		return scene.IRLight{}, err
	}
	if light.GroundColor, err = stringAttrDefault(attrs, "groundColor", light.GroundColor); err != nil {
		return scene.IRLight{}, err
	}
	if light.Intensity, err = floatAttrDefault(attrs, "intensity", light.Intensity); err != nil {
		return scene.IRLight{}, err
	}
	if light.X, err = floatAttrDefault(attrs, "x", light.X); err != nil {
		return scene.IRLight{}, err
	}
	if light.Y, err = floatAttrDefault(attrs, "y", light.Y); err != nil {
		return scene.IRLight{}, err
	}
	if light.Z, err = floatAttrDefault(attrs, "z", light.Z); err != nil {
		return scene.IRLight{}, err
	}
	if light.DirectionX, err = floatAttrDefault(attrs, "directionX", light.DirectionX); err != nil {
		return scene.IRLight{}, err
	}
	if light.DirectionY, err = floatAttrDefault(attrs, "directionY", light.DirectionY); err != nil {
		return scene.IRLight{}, err
	}
	if light.DirectionZ, err = floatAttrDefault(attrs, "directionZ", light.DirectionZ); err != nil {
		return scene.IRLight{}, err
	}
	if light.Angle, err = floatAttrDefault(attrs, "angle", light.Angle); err != nil {
		return scene.IRLight{}, err
	}
	if light.Penumbra, err = floatAttrDefault(attrs, "penumbra", light.Penumbra); err != nil {
		return scene.IRLight{}, err
	}
	if light.Range, err = floatAttrDefault(attrs, "range", light.Range); err != nil {
		return scene.IRLight{}, err
	}
	if light.Decay, err = floatAttrDefault(attrs, "decay", light.Decay); err != nil {
		return scene.IRLight{}, err
	}
	if light.CastShadow, err = boolAttrDefault(attrs, "castShadow", light.CastShadow); err != nil {
		return scene.IRLight{}, err
	}
	if light.ShadowSize, err = intAttrDefault(attrs, "shadowSize", light.ShadowSize); err != nil {
		return scene.IRLight{}, err
	}
	return light, nil
}

func lowerMesh(attrs []nir.Attr, model bool) (scene.IRNode, scene.IRMaterial, error) {
	var props scene.MeshElementProps
	var err error
	if props.ID, err = stringAttrDefault(attrs, "id", props.ID); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Kind, err = stringAttrDefault(attrs, "kind", props.Kind); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Src, err = stringAttrDefault(attrs, "src", props.Src); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Size, err = floatAttrDefault(attrs, "size", props.Size); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Width, err = floatAttrDefault(attrs, "width", props.Width); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Height, err = floatAttrDefault(attrs, "height", props.Height); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Depth, err = floatAttrDefault(attrs, "depth", props.Depth); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Radius, err = floatAttrDefault(attrs, "radius", props.Radius); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Segments, err = intAttrDefault(attrs, "segments", props.Segments); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Transform, err = lowerTransform(attrs); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.CastShadow, err = boolAttrDefault(attrs, "castShadow", props.CastShadow); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.ReceiveShadow, err = boolAttrDefault(attrs, "receiveShadow", props.ReceiveShadow); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Animation, err = stringAttrDefault(attrs, "animation", props.Animation); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if static, ok, err := boolAttr(attrs, "static"); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	} else if ok {
		props.Static = &static
	}
	material, err := lowerMaterial(attrs)
	if err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if model && props.Src == "" {
		return scene.IRNode{}, scene.IRMaterial{}, fmt.Errorf("Scene3D <Model> requires src for canonical IR conformance")
	}
	return scene.LowerMesh(props), material, nil
}

func lowerInstancedMesh(attrs []nir.Attr) (scene.IRNode, scene.IRMaterial, error) {
	var props scene.InstancedMeshElementProps
	var err error
	if props.ID, err = stringAttrDefault(attrs, "id", props.ID); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Kind, err = stringAttrDefault(attrs, "kind", props.Kind); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	countSet := false
	if count, ok, err := intAttr(attrs, "count"); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	} else if ok {
		props.Count = count
		countSet = true
	}
	if props.Width, err = floatAttrDefault(attrs, "width", props.Width); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Height, err = floatAttrDefault(attrs, "height", props.Height); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Depth, err = floatAttrDefault(attrs, "depth", props.Depth); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Radius, err = floatAttrDefault(attrs, "radius", props.Radius); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.Segments, err = intAttrDefault(attrs, "segments", props.Segments); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.CastShadow, err = boolAttrDefault(attrs, "castShadow", props.CastShadow); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if props.ReceiveShadow, err = boolAttrDefault(attrs, "receiveShadow", props.ReceiveShadow); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	if transforms, ok, err := floatListAttr(attrs, "transforms"); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	} else if ok {
		props.Transforms = transforms
	}
	if len(props.Transforms)%16 != 0 {
		return scene.IRNode{}, scene.IRMaterial{}, fmt.Errorf("Scene3D <InstancedMesh> transforms must contain 16 floats per instance")
	}
	transformCount := len(props.Transforms) / 16
	if !countSet && transformCount > 0 {
		props.Count = transformCount
	}
	if props.Count > 0 && transformCount == 0 {
		props.Transforms = identityTransforms(props.Count)
		transformCount = props.Count
	}
	if transformCount > 0 && transformCount != props.Count {
		return scene.IRNode{}, scene.IRMaterial{}, fmt.Errorf("Scene3D <InstancedMesh> count=%d but transforms describe %d instances", props.Count, transformCount)
	}
	if colors, ok, err := stringListAttr(attrs, "colors"); err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	} else if ok {
		props.Colors = colors
	}
	material, err := lowerMaterial(attrs)
	if err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	transform, err := lowerTransform(attrs)
	if err != nil {
		return scene.IRNode{}, scene.IRMaterial{}, err
	}
	node := scene.LowerInstancedMesh(props)
	node.Transform = transform
	return node, material, nil
}

func lowerPoints(attrs []nir.Attr) (scene.IRNode, error) {
	var props scene.PointsElementProps
	var err error
	if props.ID, err = stringAttrDefault(attrs, "id", props.ID); err != nil {
		return scene.IRNode{}, err
	}
	if props.Transform, err = lowerTransform(attrs); err != nil {
		return scene.IRNode{}, err
	}
	if props.Count, err = intAttrDefault(attrs, "count", props.Count); err != nil {
		return scene.IRNode{}, err
	}
	if props.Color, err = stringAttrDefault(attrs, "color", props.Color); err != nil {
		return scene.IRNode{}, err
	}
	if props.Style, err = stringAttrDefault(attrs, "style", props.Style); err != nil {
		return scene.IRNode{}, err
	}
	if props.Size, err = floatAttrDefault(attrs, "size", props.Size); err != nil {
		return scene.IRNode{}, err
	}
	if props.Opacity, err = floatAttrDefault(attrs, "opacity", props.Opacity); err != nil {
		return scene.IRNode{}, err
	}
	if props.BlendMode, err = stringAttrDefault(attrs, "blendMode", props.BlendMode); err != nil {
		return scene.IRNode{}, err
	}
	if props.Attenuation, err = boolAttrDefault(attrs, "attenuation", props.Attenuation); err != nil {
		return scene.IRNode{}, err
	}
	return scene.LowerPoints(props), nil
}

func lowerMaterial(attrs []nir.Attr) (scene.IRMaterial, error) {
	material := scene.IRMaterial{Kind: "standard"}
	var err error
	if color, ok, err := stringAttr(attrs, "color"); err != nil {
		return scene.IRMaterial{}, err
	} else if ok {
		material.Color = color
	}
	if materialKind, ok, err := stringAttr(attrs, "material"); err != nil {
		return scene.IRMaterial{}, err
	} else if ok {
		material.Kind = materialKind
	}
	if materialKind, ok, err := stringAttr(attrs, "materialKind"); err != nil {
		return scene.IRMaterial{}, err
	} else if ok {
		material.Kind = materialKind
	}
	if material.Roughness, err = floatAttrDefault(attrs, "roughness", material.Roughness); err != nil {
		return scene.IRMaterial{}, err
	}
	if material.Metalness, err = floatAttrDefault(attrs, "metalness", material.Metalness); err != nil {
		return scene.IRMaterial{}, err
	}
	return material, nil
}

func lowerTransform(attrs []nir.Attr) (scene.IRTransform, error) {
	var transform scene.IRTransform
	var err error
	if transform.X, err = floatAttrDefault(attrs, "x", transform.X); err != nil {
		return scene.IRTransform{}, err
	}
	if transform.Y, err = floatAttrDefault(attrs, "y", transform.Y); err != nil {
		return scene.IRTransform{}, err
	}
	if transform.Z, err = floatAttrDefault(attrs, "z", transform.Z); err != nil {
		return scene.IRTransform{}, err
	}
	if transform.RotationX, err = floatAttrDefault(attrs, "rotationX", transform.RotationX); err != nil {
		return scene.IRTransform{}, err
	}
	if transform.RotationY, err = floatAttrDefault(attrs, "rotationY", transform.RotationY); err != nil {
		return scene.IRTransform{}, err
	}
	if transform.RotationZ, err = floatAttrDefault(attrs, "rotationZ", transform.RotationZ); err != nil {
		return scene.IRTransform{}, err
	}
	if transform.SpinX, err = floatAttrDefault(attrs, "spinX", transform.SpinX); err != nil {
		return scene.IRTransform{}, err
	}
	if transform.SpinY, err = floatAttrDefault(attrs, "spinY", transform.SpinY); err != nil {
		return scene.IRTransform{}, err
	}
	if transform.SpinZ, err = floatAttrDefault(attrs, "spinZ", transform.SpinZ); err != nil {
		return scene.IRTransform{}, err
	}
	if transform.ScaleX, err = floatAttrDefault(attrs, "scaleX", transform.ScaleX); err != nil {
		return scene.IRTransform{}, err
	}
	if transform.ScaleY, err = floatAttrDefault(attrs, "scaleY", transform.ScaleY); err != nil {
		return scene.IRTransform{}, err
	}
	if transform.ScaleZ, err = floatAttrDefault(attrs, "scaleZ", transform.ScaleZ); err != nil {
		return scene.IRTransform{}, err
	}
	return transform, nil
}

func appendMaterial(materials *[]scene.IRMaterial, material scene.IRMaterial) int {
	*materials = append(*materials, material)
	return len(*materials) - 1
}

func lightKind(tag string) string {
	switch normalizeTag(tag) {
	case "directionallight":
		return "directional"
	case "pointlight":
		return "point"
	case "ambientlight":
		return "ambient"
	case "spotlight":
		return "spot"
	case "hemispherelight":
		return "hemisphere"
	default:
		return normalizeTag(tag)
	}
}

func stringAttrDefault(attrs []nir.Attr, name, fallback string) (string, error) {
	value, ok, err := stringAttr(attrs, name)
	if err != nil || !ok {
		return fallback, err
	}
	return value, nil
}

func stringAttr(attrs []nir.Attr, name string) (string, bool, error) {
	expr, ok := attr(attrs, name)
	if !ok {
		return "", false, nil
	}
	if expr.Kind != "literal" || expr.Literal == nil {
		return "", false, fmt.Errorf("Scene3D conformance attribute %q must be literal", name)
	}
	return expr.Literal.Value, true, nil
}

func floatAttrDefault(attrs []nir.Attr, name string, fallback float64) (float64, error) {
	value, ok, err := floatAttr(attrs, name)
	if err != nil || !ok {
		return fallback, err
	}
	return value, nil
}

func floatAttr(attrs []nir.Attr, name string) (float64, bool, error) {
	expr, ok := attr(attrs, name)
	if !ok {
		return 0, false, nil
	}
	if expr.Kind != "literal" || expr.Literal == nil {
		return 0, false, fmt.Errorf("Scene3D conformance attribute %q must be literal", name)
	}
	value, err := strconv.ParseFloat(expr.Literal.Value, 64)
	if err != nil {
		return 0, false, fmt.Errorf("Scene3D conformance attribute %q must be numeric: %w", name, err)
	}
	return value, true, nil
}

func intAttrDefault(attrs []nir.Attr, name string, fallback int) (int, error) {
	value, ok, err := intAttr(attrs, name)
	if err != nil || !ok {
		return fallback, err
	}
	return value, nil
}

func intAttr(attrs []nir.Attr, name string) (int, bool, error) {
	expr, ok := attr(attrs, name)
	if !ok {
		return 0, false, nil
	}
	if expr.Kind != "literal" || expr.Literal == nil {
		return 0, false, fmt.Errorf("Scene3D conformance attribute %q must be literal", name)
	}
	value, err := strconv.Atoi(expr.Literal.Value)
	if err != nil {
		return 0, false, fmt.Errorf("Scene3D conformance attribute %q must be integer: %w", name, err)
	}
	return value, true, nil
}

func boolAttrDefault(attrs []nir.Attr, name string, fallback bool) (bool, error) {
	value, ok, err := boolAttr(attrs, name)
	if err != nil || !ok {
		return fallback, err
	}
	return value, nil
}

func boolAttr(attrs []nir.Attr, name string) (bool, bool, error) {
	expr, ok := attr(attrs, name)
	if !ok {
		return false, false, nil
	}
	if expr.Kind != "literal" || expr.Literal == nil {
		return false, false, fmt.Errorf("Scene3D conformance attribute %q must be literal", name)
	}
	value, err := strconv.ParseBool(expr.Literal.Value)
	if err != nil {
		return false, false, fmt.Errorf("Scene3D conformance attribute %q must be bool: %w", name, err)
	}
	return value, true, nil
}

func floatListAttr(attrs []nir.Attr, name string) ([]float64, bool, error) {
	raw, ok, err := stringAttr(attrs, name)
	if err != nil || !ok {
		return nil, false, err
	}
	fields := listFields(raw)
	values := make([]float64, 0, len(fields))
	for _, field := range fields {
		value, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return nil, false, fmt.Errorf("Scene3D conformance attribute %q must be a numeric list: %w", name, err)
		}
		values = append(values, value)
	}
	return values, true, nil
}

func stringListAttr(attrs []nir.Attr, name string) ([]string, bool, error) {
	raw, ok, err := stringAttr(attrs, name)
	if err != nil || !ok {
		return nil, false, err
	}
	fields := listFields(raw)
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		if field != "" {
			values = append(values, field)
		}
	}
	return values, true, nil
}

func listFields(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ' ', '\n', '\r', '\t':
			return true
		default:
			return false
		}
	})
}

func identityTransforms(count int) []float64 {
	if count <= 0 {
		return nil
	}
	values := make([]float64, count*16)
	for i := 0; i < count; i++ {
		offset := i * 16
		values[offset] = 1
		values[offset+5] = 1
		values[offset+10] = 1
		values[offset+15] = 1
	}
	return values
}

func attr(attrs []nir.Attr, name string) (nir.RxExpr, bool) {
	for _, attr := range attrs {
		if attr.Name == name {
			return attr.Value, true
		}
	}
	return nir.RxExpr{}, false
}

func normalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}
