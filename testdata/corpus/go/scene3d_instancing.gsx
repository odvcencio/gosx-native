package scene3dinstancing

func InstancingDemo() Node {
	return <Scene3D width={640} height={360} background="#101820">
		<Camera z={9} fov={70} near={0.1} far={128} />
		<Environment ambientColor="#f4fbff" ambientIntensity={0.2} />
		<DirectionalLight id="sun" color="#fff1d6" intensity={1.2} />
		<InstancedMesh
			id="asteroids"
			kind="box"
			count={3}
			width={0.8}
			height={0.5}
			depth={0.4}
			color="#f7c76b"
			roughness={0.4}
			metalness={0.1}
			colors="#f7c76b,#93c5fd,#fca5a5"
			transforms="1,0,0,0,0,1,0,0,0,0,1,0,-1.2,0,0,1,1,0,0,0,0,1,0,0,0,0,1,0,0,0.2,0,1,1,0,0,0,0,1,0,0,0,0,1,0,1.2,-0.1,0,1"
		/>
	</Scene3D>
}
