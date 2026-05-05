package loop

type RosterProps struct {
	Items []string
}

//gosx:island
func Roster(props RosterProps) Node {
	return <vstack>
		<Each of={props.Items} as="item">
			<text>{item}</text>
		</Each>
	</vstack>
}
