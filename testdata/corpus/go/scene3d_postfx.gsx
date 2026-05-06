package scene3dpostfx

type PostFXDemoProps struct {
	Width  int
	Height int
}

func PostFXDemo(props PostFXDemoProps) Node {
	return <Scene3D width={props.Width} height={props.Height} background="#080f1f">
		<Camera z={7} fov={64} near={0.1} far={96} />
		<Mesh id="hero" kind="box" width={1.4} height={1.0} depth={0.7} color="#8de1ff" />
		<PostFX.Bloom threshold={0.72} intensity={1.4} radius={0.25} scale={0.5} />
		<PostFX.Vignette intensity={0.35} />
		<PostFX.ColorGrading saturation={1.1} contrast={1.05} exposure={0.08} />
		<PostFX.Tonemap mode="aces" exposure={1.0} />
	</Scene3D>
}
