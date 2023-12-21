package types

const (
	// ModuleName defines the module name
	ModuleName = "ugdvesting"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_ugdvesting"
)

var (
	ParamsKey = []byte("p_ugdvesting")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
