package greeter

type GreeterProps struct {
	InitialName string
}

//gosx:island
func Greeter(props GreeterProps) Node {
	name := signal.New(props.InitialName)

	updateName := func() {
		name.Set(value)
	}

	return <vstack>
		<input type="text" value={name.Get()} placeholder="Name" onInput={updateName} />
		<text>Hello, {name.Get()}</text>
	</vstack>
}
