package sysload

type Sysload struct {
	RamUsedBytes  uint64
	RamTotalBytes uint64

	CpuCoresLogical  int
	CpuCoresPhysical int

	LoadAverage1M  float64
	LoadAverage5M  float64
	LoadAverage15M float64
}

func GetSysload() (Sysload, error) {
	sysload := Sysload{}

	ramUseBytes, ramTotalBytes, err := getMemoryUsage()
	if err != nil {
		return Sysload{}, err
	}
	sysload.RamUsedBytes = ramUseBytes
	sysload.RamTotalBytes = ramTotalBytes

	cpuCoresLogical, cpuCoresPhysical, err := getCpuCoreCounts()
	if err != nil {
		return Sysload{}, err
	}
	sysload.CpuCoresLogical = cpuCoresLogical
	sysload.CpuCoresPhysical = cpuCoresPhysical

	sysload1M, sysload5M, sysload15M, err := getLoadAverages()
	if err != nil {
		return Sysload{}, err
	}
	sysload.LoadAverage1M = sysload1M
	sysload.LoadAverage5M = sysload5M
	sysload.LoadAverage15M = sysload15M

	return sysload, nil
}
