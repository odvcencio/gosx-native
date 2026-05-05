package counter

type CounterProps struct {
	Start int
}

//gosx:island
func Counter(props CounterProps) Node {
	count := signal.New(props.Start)

	decrement := func() {
		count.Set(count.Get() - 1)
	}
	increment := func() {
		count.Set(count.Get() + 1)
	}

	return <div class="counter">
		<button onClick={decrement}>-</button>
		<span class="count">{count.Get()}</span>
		<button onClick={increment}>+</button>
	</div>
}
