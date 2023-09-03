package types

const (
	// ModuleName defines the module name
	ModuleName = "ugdvesting"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_ugdvesting"
)

var (
	// ParamsKey is the prefix for params key
	ParamsKey      = []byte{0x00}
	VestingKey     = []byte{0x01}
	VestingDataKey = []byte{0x02}
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
