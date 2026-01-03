package sysload

type Sysload struct {
	RamUsedBytes  uint64
	RamTotalBytes uint64

	CpuCoresLogical  int
	CpuCoresPhysical int
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

	return sysload, nil
}
