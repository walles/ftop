package sysload

type Sysload struct {
	RamUsedBytes  uint64
	RamTotalBytes uint64
}

func GetSysload() (Sysload, error) {
	sysload := Sysload{}

	ramUseBytes, ramTotalBytes, err := getMemoryUsage()
	if err != nil {
		return Sysload{}, err
	}

	sysload.RamUsedBytes = ramUseBytes
	sysload.RamTotalBytes = ramTotalBytes

	return sysload, nil
}
