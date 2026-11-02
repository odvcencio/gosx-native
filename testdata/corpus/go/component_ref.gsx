package componentref

type BadgeProps struct {
	Label string
}

component Badge(props: BadgeProps) {
	return <text>{props.Label}</text>
}

type ProfileProps struct {
	Name string
}

//gosx:island
component Profile(props: ProfileProps) {
	return <vstack>
		<Badge label={props.Name} />
	</vstack>
}
