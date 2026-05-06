package scene3d

type SceneDemoProps struct {
	Width  int
	Height int
}

func SceneDemo(props SceneDemoProps) Node {
	return <Scene3D class="native-scene" width={props.Width} height={props.Height}>
		<Camera z={7} fov={64} near={0.1} far={96} />
		<Environment ambientColor="#f4fbff" ambientIntensity={0.2} />
		<DirectionalLight id="sun" color="#fff1d6" intensity={1.2} />
		<Mesh id="hero" kind="box" width={1.8} height={1.2} depth={0.8} color="#8de1ff" />
		<Model id="ship" src="/models/ship.glb" animation="idle" static />
		<Points id="stars" count={2} size={0.5} />
	</Scene3D>
}
