module github.com/CamiloHernandez/beekeeper/cmd

go 1.15

require (
	github.com/CamiloHernandez/beekeeper/lib v0.2.0
	github.com/spf13/cobra v1.1.1
)

replace github.com/CamiloHernandez/beekeeper/lib => ./../../lib
