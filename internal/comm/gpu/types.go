package gpu

// Info contains information about a GPU.
// This type is defined in the gpu package to avoid import cycles.
type Info struct {
	Index         int     `json:"index"`
	Name          string  `json:"name"`
	Vendor        string  `json:"vendor"`
	TotalMemory   int64   `json:"totalMemory"` // bytes
	UsedMemory    int64   `json:"usedMemory"`  // bytes
	Temperature   float64 `json:"temperature"` // celsius
	Utilization   float64 `json:"utilization"` // percentage 0-100
	PowerUsage    float64 `json:"powerUsage"`  // watts
	DriverVersion string  `json:"driverVersion,omitempty"`
}
