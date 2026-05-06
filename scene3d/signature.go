package scene3d

import (
	"encoding/json"
	"html"
	"math"
	"regexp"
	"strings"

	"github.com/odvcencio/gosx/scene"
)

const (
	renderSignatureVersion  = 1
	defaultSceneBackground  = "#101820"
	defaultSceneColor       = "#8de1ff"
	signatureViewportWidth  = 640.0
	signatureViewportHeight = 360.0
)

var (
	htmlTagPattern        = regexp.MustCompile(`<[^>]+>`)
	htmlWhitespacePattern = regexp.MustCompile(`\s+`)
)

// RenderSignature marshals a deterministic, renderer-agnostic Scene3D render
// contract for conformance fixtures.
func RenderSignature(ir scene.IR) ([]byte, error) {
	if err := ir.Validate(); err != nil {
		return nil, err
	}
	doc := BuildRenderSignature(ir)
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// BuildRenderSignature converts canonical scene.IR into normalized draw
// commands that define the expected renderer-agnostic output contract.
func BuildRenderSignature(ir scene.IR) RenderSignatureDoc {
	doc := RenderSignatureDoc{
		Version: renderSignatureVersion,
		Viewport: RenderViewport{
			Width:  signatureViewportWidth,
			Height: signatureViewportHeight,
		},
		Background: sceneBackground(ir.Environment.Background),
		Camera:     ir.Camera,
		Commands:   make([]RenderCommand, 0, len(ir.Lights)+len(ir.Nodes)+len(ir.PostFX)+len(htmlOverlays(ir.Metadata))),
	}

	for _, light := range ir.Lights {
		doc.Commands = append(doc.Commands, RenderCommand{
			Op:        "light",
			ID:        light.ID,
			Kind:      light.Kind,
			Color:     light.Color,
			Intensity: light.Intensity,
			Position: &RenderVector3{
				X: light.X,
				Y: light.Y,
				Z: light.Z,
			},
			Direction: &RenderVector3{
				X: light.DirectionX,
				Y: light.DirectionY,
				Z: light.DirectionZ,
			},
		})
	}

	renderables := renderableNodes(ir.Nodes)
	for index, node := range renderables {
		doc.Commands = append(doc.Commands, nodeRenderCommand(ir, node, index, len(renderables)))
	}

	for _, effect := range ir.PostFX {
		doc.Commands = append(doc.Commands, postEffectRenderCommand(effect))
	}

	for _, overlay := range htmlOverlays(ir.Metadata) {
		doc.Commands = append(doc.Commands, htmlRenderCommand(overlay))
	}

	return roundRenderSignature(doc)
}

type RenderSignatureDoc struct {
	Version    int             `json:"version"`
	Viewport   RenderViewport  `json:"viewport"`
	Background string          `json:"background"`
	Camera     scene.IRCamera  `json:"camera"`
	Commands   []RenderCommand `json:"commands"`
}

type RenderViewport struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type RenderCommand struct {
	Op            string            `json:"op"`
	ID            string            `json:"id,omitempty"`
	Tag           string            `json:"tag,omitempty"`
	Kind          string            `json:"kind,omitempty"`
	Shape         string            `json:"shape,omitempty"`
	Source        string            `json:"source,omitempty"`
	Color         string            `json:"color,omitempty"`
	MaterialKind  string            `json:"materialKind,omitempty"`
	Alpha         float64           `json:"alpha,omitempty"`
	Intensity     float64           `json:"intensity,omitempty"`
	Count         int               `json:"count,omitempty"`
	VisibleCount  int               `json:"visibleCount,omitempty"`
	Position      *RenderVector3    `json:"position,omitempty"`
	Direction     *RenderVector3    `json:"direction,omitempty"`
	Size          *RenderSize3      `json:"size,omitempty"`
	Screen        *RenderScreenRect `json:"screen,omitempty"`
	Instances     []RenderInstance  `json:"instances,omitempty"`
	Particle      *RenderParticle   `json:"particle,omitempty"`
	Text          string            `json:"text,omitempty"`
	ClassName     string            `json:"className,omitempty"`
	PointerEvents string            `json:"pointerEvents,omitempty"`
	Effect        *RenderEffect     `json:"effect,omitempty"`
}

type RenderVector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type RenderSize3 struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Depth  float64 `json:"depth"`
}

type RenderScreenRect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type RenderInstance struct {
	Index  int              `json:"index"`
	Color  string           `json:"color,omitempty"`
	World  RenderVector3    `json:"world"`
	Screen RenderScreenRect `json:"screen"`
}

type RenderParticle struct {
	EmitterKind string  `json:"emitterKind,omitempty"`
	Radius      float64 `json:"radius"`
}

type RenderEffect struct {
	Threshold     float64 `json:"threshold,omitempty"`
	Intensity     float64 `json:"intensity,omitempty"`
	Radius        float64 `json:"radius,omitempty"`
	Scale         float64 `json:"scale,omitempty"`
	Saturation    float64 `json:"saturation,omitempty"`
	Contrast      float64 `json:"contrast,omitempty"`
	Exposure      float64 `json:"exposure,omitempty"`
	Mode          string  `json:"mode,omitempty"`
	FocusDistance float64 `json:"focusDistance,omitempty"`
	Aperture      float64 `json:"aperture,omitempty"`
	MaxBlur       float64 `json:"maxBlur,omitempty"`
	Alpha         float64 `json:"alpha,omitempty"`
	LineWidth     float64 `json:"lineWidth,omitempty"`
	Thickness     float64 `json:"thickness,omitempty"`
	Tint          string  `json:"tint,omitempty"`
}

func renderableNodes(nodes []scene.IRNode) []scene.IRNode {
	out := make([]scene.IRNode, 0, len(nodes))
	for _, node := range nodes {
		switch node.Kind {
		case "mesh", "points", "instanced-mesh", "compute-particles":
			out = append(out, node)
		}
	}
	return out
}

func nodeRenderCommand(ir scene.IR, node scene.IRNode, index, total int) RenderCommand {
	switch node.Kind {
	case "points":
		return pointsRenderCommand(node, index, total)
	case "instanced-mesh":
		return instancedRenderCommand(ir, node, index, total)
	case "compute-particles":
		return computeRenderCommand(node, index, total)
	default:
		return meshRenderCommand(ir, node, index, total)
	}
}

func meshRenderCommand(ir scene.IR, node scene.IRNode, index, total int) RenderCommand {
	mesh := node.Mesh
	tag := "mesh"
	kind := ""
	source := ""
	width, height, depth := 1.0, 1.0, 1.0
	if mesh != nil {
		kind = mesh.Kind
		source = mesh.Src
		width = dimensionOrDefault(mesh.Width)
		height = dimensionOrDefault(mesh.Height)
		depth = dimensionOrDefault(mesh.Depth)
		if source != "" {
			tag = "model"
		}
	}
	color, materialKind := materialRenderAttrs(ir, node.MaterialIndex)
	position := vectorFromTransform(node.Transform)
	screen := screenRect(position, width, height, index, total)
	shape := "roundRect"
	alpha := 0.86
	if tag == "model" {
		shape = "oval"
		alpha = 0.82
	}
	return RenderCommand{
		Op:           "node",
		ID:           node.ID,
		Tag:          tag,
		Kind:         kind,
		Shape:        shape,
		Source:       source,
		Color:        color,
		MaterialKind: materialKind,
		Alpha:        alpha,
		Position:     &position,
		Size:         &RenderSize3{Width: width, Height: height, Depth: depth},
		Screen:       &screen,
	}
}

func pointsRenderCommand(node scene.IRNode, index, total int) RenderCommand {
	points := node.Points
	count := 0
	size := 0.0
	color := defaultSceneColor
	if points != nil {
		count = points.Count
		size = points.Size
		if strings.TrimSpace(points.Color) != "" {
			color = points.Color
		}
	}
	position := vectorFromTransform(node.Transform)
	screen := screenRect(position, 1, 1, index, total)
	return RenderCommand{
		Op:           "node",
		ID:           node.ID,
		Tag:          "points",
		Shape:        "circles",
		Color:        color,
		Alpha:        0.9,
		Count:        count,
		VisibleCount: maxInt(count, 1),
		Position:     &position,
		Size:         &RenderSize3{Width: 1, Height: 1, Depth: 1},
		Screen:       &screen,
		Particle:     &RenderParticle{Radius: math.Max(size*8, 3)},
	}
}

func instancedRenderCommand(ir scene.IR, node scene.IRNode, index, total int) RenderCommand {
	instanced := node.InstancedMesh
	count := 0
	kind := ""
	width, height, depth := 1.0, 1.0, 1.0
	var transforms []float64
	var colors []string
	if instanced != nil {
		count = instanced.Count
		kind = instanced.Kind
		width = dimensionOrDefault(instanced.Width)
		height = dimensionOrDefault(instanced.Height)
		depth = dimensionOrDefault(instanced.Depth)
		transforms = instanced.Transforms
		colors = instanced.Colors
	}
	color, materialKind := materialRenderAttrs(ir, node.MaterialIndex)
	position := vectorFromTransform(node.Transform)
	screen := screenRect(position, width, height, index, total)
	instanceWidth := math.Max(screen.Width*0.42, 10)
	instanceHeight := math.Max(screen.Height*0.42, 10)
	instances := renderInstances(count, transforms, colors, color, screen, instanceWidth, instanceHeight)
	return RenderCommand{
		Op:           "node",
		ID:           node.ID,
		Tag:          "instancedMesh",
		Kind:         kind,
		Shape:        "roundRectBatch",
		Color:        color,
		MaterialKind: materialKind,
		Alpha:        0.84,
		Count:        count,
		VisibleCount: maxInt(count, 1),
		Position:     &position,
		Size:         &RenderSize3{Width: width, Height: height, Depth: depth},
		Screen:       &screen,
		Instances:    instances,
	}
}

func computeRenderCommand(node scene.IRNode, index, total int) RenderCommand {
	compute := node.Compute
	count := 0
	kind := ""
	size := 0.0
	color := defaultSceneColor
	position := vectorFromTransform(node.Transform)
	if compute != nil {
		count = compute.Count
		kind = compute.Emitter.Kind
		size = compute.Material.Size
		if strings.TrimSpace(compute.Material.Color) != "" {
			color = compute.Material.Color
		}
		position = RenderVector3{X: compute.Emitter.X, Y: compute.Emitter.Y, Z: compute.Emitter.Z}
	}
	screen := screenRect(position, 1, 1, index, total)
	return RenderCommand{
		Op:           "node",
		ID:           node.ID,
		Tag:          "computeParticles",
		Kind:         kind,
		Shape:        "spiralParticles",
		Color:        color,
		Alpha:        0.72,
		Count:        count,
		VisibleCount: maxInt(minInt(count, 48), 1),
		Position:     &position,
		Size:         &RenderSize3{Width: 1, Height: 1, Depth: 1},
		Screen:       &screen,
		Particle:     &RenderParticle{EmitterKind: kind, Radius: math.Max(size*8, 2)},
	}
}

func renderInstances(count int, transforms []float64, colors []string, fallbackColor string, parent RenderScreenRect, width, height float64) []RenderInstance {
	visible := maxInt(count, 1)
	instances := make([]RenderInstance, 0, visible)
	for i := 0; i < visible; i++ {
		offset := (float64(i) - float64(visible-1)/2) * width * 0.72
		rise := float64((i%2)*2-1) * height * 0.18
		world := instanceWorld(transforms, i)
		color := fallbackColor
		if i < len(colors) && strings.TrimSpace(colors[i]) != "" {
			color = colors[i]
		}
		instances = append(instances, RenderInstance{
			Index: i,
			Color: color,
			World: world,
			Screen: RenderScreenRect{
				X:      parent.X + offset - width/2,
				Y:      parent.Y + rise - height/2,
				Width:  width,
				Height: height,
			},
		})
	}
	return instances
}

func instanceWorld(transforms []float64, index int) RenderVector3 {
	offset := index * 16
	if offset+14 >= len(transforms) {
		return RenderVector3{}
	}
	return RenderVector3{
		X: transforms[offset+12],
		Y: transforms[offset+13],
		Z: transforms[offset+14],
	}
}

func htmlRenderCommand(overlay map[string]any) RenderCommand {
	x := floatMetadata(overlay, "x")
	y := floatMetadata(overlay, "y")
	z := floatMetadata(overlay, "z")
	offsetX := floatMetadata(overlay, "offsetX")
	offsetY := floatMetadata(overlay, "offsetY")
	width := dimensionOrDefault(floatMetadata(overlay, "width"))
	height := dimensionOrDefault(floatMetadata(overlay, "height"))
	opacity := floatMetadataDefault(overlay, "opacity", 1)
	screen := RenderScreenRect{
		X:      signatureViewportWidth*0.5 + x*36 + offsetX,
		Y:      signatureViewportHeight*0.5 - y*36 + z*4 + offsetY,
		Width:  width,
		Height: height,
	}
	return RenderCommand{
		Op:            "htmlText",
		ID:            stringMetadata(overlay, "id"),
		Text:          plainHTMLText(stringMetadata(overlay, "html")),
		ClassName:     stringMetadata(overlay, "className"),
		PointerEvents: stringMetadata(overlay, "pointerEvents"),
		Alpha:         clamp(opacity, 0, 1),
		Position:      &RenderVector3{X: x, Y: y, Z: z},
		Screen:        &screen,
	}
}

func postEffectRenderCommand(effect scene.IRPostEffect) RenderCommand {
	renderEffect := RenderEffect{
		Threshold:     effect.Threshold,
		Intensity:     effect.Intensity,
		Radius:        effect.Radius,
		Scale:         effect.Scale,
		Saturation:    effect.Saturation,
		Contrast:      effect.Contrast,
		Exposure:      effect.Exposure,
		Mode:          effect.Mode,
		FocusDistance: effect.FocusDistance,
		Aperture:      effect.Aperture,
		MaxBlur:       effect.MaxBlur,
	}
	color := ""
	switch effect.Kind {
	case "bloom":
		renderEffect.Alpha = clamp(effect.Intensity*0.16, 0, 0.34)
		renderEffect.LineWidth = math.Max(effect.Radius*18, 8)
		color = "#ffffff"
	case "vignette":
		renderEffect.Alpha = clamp(effect.Intensity*0.46, 0, 0.58)
		renderEffect.Thickness = math.Min(signatureViewportWidth, signatureViewportHeight) * clamp(effect.Radius, 0.12, 0.5)
		color = "#000000"
	case "colorGrade":
		renderEffect.Alpha = clamp(math.Abs(effect.Saturation-1)*0.08+math.Abs(effect.Contrast-1)*0.05+math.Abs(effect.Exposure)*0.04, 0, 0.18)
		if effect.Saturation >= 1 {
			renderEffect.Tint = "#ffd18a"
		} else {
			renderEffect.Tint = "#75b8ff"
		}
		color = renderEffect.Tint
	case "toneMapping":
		renderEffect.Alpha = clamp(math.Abs(effect.Exposure-1)*0.08, 0, 0.2)
		if effect.Exposure >= 1 {
			color = "#ffffff"
		} else {
			color = "#000000"
		}
	}
	return RenderCommand{
		Op:     "postfx",
		Kind:   effect.Kind,
		Color:  color,
		Alpha:  renderEffect.Alpha,
		Effect: &renderEffect,
	}
}

func materialRenderAttrs(ir scene.IR, index int) (string, string) {
	if index >= 0 && index < len(ir.Materials) {
		material := ir.Materials[index]
		color := material.Color
		if strings.TrimSpace(color) == "" {
			color = defaultSceneColor
		}
		return color, material.Kind
	}
	return defaultSceneColor, ""
}

func screenRect(position RenderVector3, width, height float64, index, total int) RenderScreenRect {
	slotCount := maxInt(total, 1)
	slotWidth := signatureViewportWidth / float64(slotCount+1)
	centerX := slotWidth*float64(index+1) + position.X*24
	centerY := signatureViewportHeight*0.5 - position.Y*24 + position.Z*4
	scale := math.Min(signatureViewportWidth, signatureViewportHeight) * 0.18
	screenWidth := math.Max(width*scale, 16)
	screenHeight := math.Max(height*scale, 16)
	return RenderScreenRect{
		X:      centerX - screenWidth/2,
		Y:      centerY - screenHeight/2,
		Width:  screenWidth,
		Height: screenHeight,
	}
}

func vectorFromTransform(transform scene.IRTransform) RenderVector3 {
	return RenderVector3{X: transform.X, Y: transform.Y, Z: transform.Z}
}

func sceneBackground(background string) string {
	if strings.TrimSpace(background) == "" {
		return defaultSceneBackground
	}
	return background
}

func dimensionOrDefault(value float64) float64 {
	if value == 0 {
		return 1
	}
	return value
}

func htmlOverlays(metadata map[string]any) []map[string]any {
	raw, ok := metadata["html"]
	if !ok || raw == nil {
		return nil
	}
	switch overlays := raw.(type) {
	case []map[string]any:
		return overlays
	case []any:
		out := make([]map[string]any, 0, len(overlays))
		for _, item := range overlays {
			if overlay, ok := item.(map[string]any); ok {
				out = append(out, overlay)
			}
		}
		return out
	default:
		return nil
	}
}

func plainHTMLText(markup string) string {
	text := htmlTagPattern.ReplaceAllString(markup, " ")
	text = html.UnescapeString(text)
	text = htmlWhitespacePattern.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func stringMetadata(metadata map[string]any, key string) string {
	value, _ := metadata[key].(string)
	return value
}

func floatMetadata(metadata map[string]any, key string) float64 {
	return floatMetadataDefault(metadata, key, 0)
}

func floatMetadataDefault(metadata map[string]any, key string, fallback float64) float64 {
	switch value := metadata[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		parsed, err := value.Float64()
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func clamp(value, lo, hi float64) float64 {
	return math.Min(math.Max(value, lo), hi)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func roundRenderSignature(doc RenderSignatureDoc) RenderSignatureDoc {
	doc.Viewport.Width = roundFloat(doc.Viewport.Width)
	doc.Viewport.Height = roundFloat(doc.Viewport.Height)
	doc.Camera = roundCamera(doc.Camera)
	for i := range doc.Commands {
		command := &doc.Commands[i]
		command.Alpha = roundFloat(command.Alpha)
		command.Intensity = roundFloat(command.Intensity)
		if command.Position != nil {
			roundVector(command.Position)
		}
		if command.Direction != nil {
			roundVector(command.Direction)
		}
		if command.Size != nil {
			command.Size.Width = roundFloat(command.Size.Width)
			command.Size.Height = roundFloat(command.Size.Height)
			command.Size.Depth = roundFloat(command.Size.Depth)
		}
		if command.Screen != nil {
			roundScreen(command.Screen)
		}
		for j := range command.Instances {
			roundVector(&command.Instances[j].World)
			roundScreen(&command.Instances[j].Screen)
		}
		if command.Particle != nil {
			command.Particle.Radius = roundFloat(command.Particle.Radius)
		}
		if command.Effect != nil {
			roundEffect(command.Effect)
		}
	}
	return doc
}

func roundCamera(camera scene.IRCamera) scene.IRCamera {
	camera.X = roundFloat(camera.X)
	camera.Y = roundFloat(camera.Y)
	camera.Z = roundFloat(camera.Z)
	camera.RotationX = roundFloat(camera.RotationX)
	camera.RotationY = roundFloat(camera.RotationY)
	camera.RotationZ = roundFloat(camera.RotationZ)
	camera.FOV = roundFloat(camera.FOV)
	camera.Left = roundFloat(camera.Left)
	camera.Right = roundFloat(camera.Right)
	camera.Top = roundFloat(camera.Top)
	camera.Bottom = roundFloat(camera.Bottom)
	camera.Zoom = roundFloat(camera.Zoom)
	camera.Near = roundFloat(camera.Near)
	camera.Far = roundFloat(camera.Far)
	camera.TransitionMS = roundFloat(camera.TransitionMS)
	return camera
}

func roundVector(vector *RenderVector3) {
	vector.X = roundFloat(vector.X)
	vector.Y = roundFloat(vector.Y)
	vector.Z = roundFloat(vector.Z)
}

func roundScreen(screen *RenderScreenRect) {
	screen.X = roundFloat(screen.X)
	screen.Y = roundFloat(screen.Y)
	screen.Width = roundFloat(screen.Width)
	screen.Height = roundFloat(screen.Height)
}

func roundEffect(effect *RenderEffect) {
	effect.Threshold = roundFloat(effect.Threshold)
	effect.Intensity = roundFloat(effect.Intensity)
	effect.Radius = roundFloat(effect.Radius)
	effect.Scale = roundFloat(effect.Scale)
	effect.Saturation = roundFloat(effect.Saturation)
	effect.Contrast = roundFloat(effect.Contrast)
	effect.Exposure = roundFloat(effect.Exposure)
	effect.FocusDistance = roundFloat(effect.FocusDistance)
	effect.Aperture = roundFloat(effect.Aperture)
	effect.MaxBlur = roundFloat(effect.MaxBlur)
	effect.Alpha = roundFloat(effect.Alpha)
	effect.LineWidth = roundFloat(effect.LineWidth)
	effect.Thickness = roundFloat(effect.Thickness)
}

func roundFloat(value float64) float64 {
	rounded := math.Round(value*1e6) / 1e6
	if rounded == 0 {
		return 0
	}
	return rounded
}
