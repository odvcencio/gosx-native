package conditional

type ToggleProps struct {
	InitiallyVisible bool
}

//gosx:island
func Toggle(props ToggleProps) Node {
	visible := signal.New(props.InitiallyVisible)

	hide := func() {
		visible.Set(false)
	}

	return <vstack>
		<If when={visible.Get()}>
			<text>Visible</text>
		</If>
		<button onClick={hide}>Hide</button>
	</vstack>
}
