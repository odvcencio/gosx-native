package scene3dhtml

func HTMLDemo() Node {
	return <Scene3D width={640} height={360} background="#081119">
		<Camera z={7} fov={64} near={0.1} far={96} />
		<Mesh id="hero" kind="box" width={1.4} height={1.0} depth={0.7} color="#8de1ff" />
		<Html id="hud" x={0} y={1.1} z={0.2} width={1.6} height={0.8} opacity={0.95} pointerEvents="auto" class="scene-hud">
			<aside class="scene-hud__card"><strong>Hull</strong><span>stable</span></aside>
		</Html>
	</Scene3D>
}
