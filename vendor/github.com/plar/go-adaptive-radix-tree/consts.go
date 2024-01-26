package art

// node constraints
const (
	node4Min = 2
	node4Max = 4

	node16Min = node4Max + 1
	node16Max = 16

	node48Min = node16Max + 1
	node48Max = 48

	node256Min = node48Max + 1
	node256Max = 256
)

const (
	// MaxPrefixLen is maximum prefix length for internal nodes.
	MaxPrefixLen = 10
)
