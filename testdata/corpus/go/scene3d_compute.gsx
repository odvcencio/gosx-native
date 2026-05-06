package scene3dcompute

func ParticleDemo() Node {
	return <Scene3D width={640} height={360}>
		<Camera z={7} fov={64} near={0.1} far={96} />
		<ComputeParticles
			id="sparks"
			count={128}
			kind="spiral"
			x={0}
			y={0.2}
			z={0}
			radius={1.5}
			rate={64}
			lifetime={2.5}
			arms={3}
			wind={0.1}
			scatter={0.2}
			color="#ffd48f"
			colorEnd="#5eb7ff"
			style="focus"
			size={0.12}
			sizeEnd={0.02}
			opacity={0.9}
			opacityEnd={0.1}
			blendMode="additive"
			attenuation
			force="orbit"
			forceStrength={0.4}
			forceY={1}
			forceFrequency={0.7}
			bounds={8}
		/>
	</Scene3D>
}
