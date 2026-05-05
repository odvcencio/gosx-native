package derived

type DerivedProps struct {
	Start int
}

//gosx:island
func Derived(props DerivedProps) Node {
	count := signal.New(props.Start)
	doubled := signal.Derive(func() int {
		return count.Get() * 2
	})

	increment := func() {
		count.Set(count.Get() + 1)
	}

	return <vstack>
		<text>{doubled.Get()}</text>
		<button onClick={increment}>+</button>
	</vstack>
}
