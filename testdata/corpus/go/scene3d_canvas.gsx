package scene3dcanvas

func CanvasBackendDemo() Node {
	return <Scene3D width={640} height={360} background="#101820" backend="canvas">
		<Camera z={7} fov={64} near={0.1} far={96} />
		<Mesh id="hero" kind="box" width={1.4} height={1.0} depth={0.7} color="#8de1ff" />
	</Scene3D>
}
