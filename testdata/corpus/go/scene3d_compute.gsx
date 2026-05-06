package scene3dcompute

func ParticleDemo() Node {
	return <Scene3D width={640} height={360}>
		<ComputeParticles id="sparks" count={128} />
	</Scene3D>
}
