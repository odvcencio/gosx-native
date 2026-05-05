package panel

type PanelProps struct {
	Start int
	Label string
}

//gosx:island
func Panel(props PanelProps) Node {
	count := signal.New(props.Start)
	label := signal.New(props.Label)

	reset := func() {
		count.Set(props.Start)
		label.Set(props.Label)
	}
	advance := func() {
		count.Set(count.Get() + 1)
		label.Set("advanced")
	}

	return <vstack>
		<text>{label.Get()}</text>
		<hstack>
			<button onClick={reset}>Reset</button>
			<text>{count.Get()}</text>
			<button data-on-click="count.Set(count.Get() + 1)">+</button>
			<button onClick={advance}>Go</button>
		</hstack>
	</vstack>
}
