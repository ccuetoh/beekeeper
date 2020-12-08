module github.com/CamiloHernandez/beekeeper/bee

go 1.13

require (
	github.com/CamiloHernandez/beekeeper/lib v0.2.0
	github.com/spf13/cobra v1.1.1
)

replace github.com/CamiloHernandez/beekeeper/lib => ./../lib
