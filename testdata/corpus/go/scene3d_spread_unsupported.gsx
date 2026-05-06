package scene3dspread

type SpreadDemoProps struct {
	Mesh map[string]any
}

func SpreadDemo(props SpreadDemoProps) Node {
	return <Scene3D width={640} height={360}>
		<Mesh {...props.Mesh} />
	</Scene3D>
}
