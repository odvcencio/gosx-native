package formcontrols

type FormControlsProps struct {
	Bio           string
	Enabled      bool
	SelectedIndex int
}

//gosx:island
func FormControls(props FormControlsProps) Node {
	bio := signal.New(props.Bio)
	enabled := signal.New(props.Enabled)
	lastKey := signal.New("")
	choice := signal.New(props.SelectedIndex)

	updateBio := func() {
		bio.Set(value)
	}
	toggleEnabled := func() {
		enabled.Set(checked)
	}
	recordKey := func() {
		lastKey.Set(key)
	}
	choose := func() {
		choice.Set(selectedIndex)
	}

	return <vstack>
		<textarea value={bio.Get()} placeholder="Bio" onInput={updateBio}></textarea>
		<input type="checkbox" checked={enabled.Get()} onChange={toggleEnabled} />
		<input type="text" value={lastKey.Get()} placeholder="Press key" onKey={recordKey} />
		<select selectedIndex={choice.Get()} onChange={choose}>
			<option>One</option>
			<option>Two</option>
		</select>
		<text>{bio.Get()}</text>
		<text>{enabled.Get()}</text>
		<text>{lastKey.Get()}</text>
		<text>{choice.Get()}</text>
	</vstack>
}
