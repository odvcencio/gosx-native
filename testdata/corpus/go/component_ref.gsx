package componentref

type BadgeProps struct {
	Label string
}

func Badge(props BadgeProps) Node {
	return <text>{props.Label}</text>
}

type ProfileProps struct {
	Name string
}

//gosx:island
func Profile(props ProfileProps) Node {
	return <vstack>
		<Badge label={props.Name} />
	</vstack>
}
