package expressions

type ExpressionsProps struct {
	Title string
	Count int
	Price string
	Tags  []string
}

//gosx:island
func Expressions(props ExpressionsProps) Node {
	title := signal.New(props.Title)
	count := signal.New(props.Count)
	status := signal.New("")

	normalized := signal.Derive(func() string {
		return title.Get().Trim().Replace(" ", "-").ToLower()
	})
	countText := signal.Derive(func() string {
		return count.Get().ToString()
	})
	adjustedCount := signal.Derive(func() int {
		return countText.Get().ToInt() + 2
	})
	price := signal.Derive(func() float64 {
		return props.Price.ToFloat()
	})
	tagLine := signal.Derive(func() string {
		return props.Tags.Join(",")
	})
	hasGo := signal.Derive(func() bool {
		return title.Get().StartsWith("Go")
	})
	hasSX := signal.Derive(func() bool {
		return title.Get().Contains("SX")
	})
	tagCount := signal.Derive(func() int {
		return len(props.Tags)
	})

	return <vstack>
		<button data-on-click="status.Set(count.Get() > 0 ? title.Get().Trim().ToUpper() : normalized.Get())">Refresh</button>
		<text>{status.Get()}</text>
		<text>{normalized.Get()}</text>
		<text>{countText.Get()}</text>
		<text>{adjustedCount.Get()}</text>
		<text>{price.Get()}</text>
		<text>{tagLine.Get()}</text>
		<text>{hasGo.Get()}</text>
		<text>{hasSX.Get()}</text>
		<text>{tagCount.Get()}</text>
	</vstack>
}
