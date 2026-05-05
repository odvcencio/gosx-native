module github.com/odvcencio/gosx-native

go 1.26

replace (
	github.com/odvcencio/gosx => ../gosx
	github.com/odvcencio/gotreesitter => ../gotreesitter
)

require (
	github.com/odvcencio/gosx v0.0.0-00010101000000-000000000000
	github.com/odvcencio/gotreesitter v0.15.3
)
