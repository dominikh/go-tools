package lint

const (
	StateInitializing = 0
	StateGraph        = 1
	StateProcessing   = 2
	StateCumulative   = 3
)

type Stats struct {
	State uint64

	InitialPackages          uint64
	TotalPackages            uint64
	ProcessedPackages        uint64
	ProcessedInitialPackages uint64
	Problems                 uint64
	ActiveWorkers            uint64
	TotalWorkers             uint64
}
