struct CounterProps { var start: Int }

func Counter(props: CounterProps) -> Node {
    let count = signal(props.start)
    return <vstack>
        <button onTap={ count.set(count.get() - 1) }>-</button>
        <text>{count.get()}</text>
        <button onTap={ count.set(count.get() + 1) }>+</button>
    </vstack>
}
