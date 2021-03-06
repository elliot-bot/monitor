package host

import (
	"context"
	"strconv"
	"strings"

	"github.com/json-iterator/go"
	"github.com/luoyunpeng/monitor/common"
	"golang.org/x/sys/unix"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

// Memory usage statistics. Total, Available and Used contain numbers of bytes
// for human consumption.
//
// The other fields in this struct contain kernel specific values.
type VirtualMemoryStat struct {
	// Total amount of RAM on this system
	Total uint64 `json:"total"`

	// RAM available for programs to allocate
	//
	// This value is computed from the kernel specific values.
	Available uint64 `json:"available"`

	// RAM used by programs
	//
	// This value is computed from the kernel specific values.
	Used uint64 `json:"used"`

	// Percentage of RAM used by programs
	//
	// This value is computed from the kernel specific values.
	UsedPercent float64 `json:"usedPercent"`

	// This is the kernel's notion of free memory; RAM chips whose bits nobody
	// cares about the value of right now. For a human consumable number,
	// Available is what you really want.
	Free uint64 `json:"free"`

	// OS X / BSD specific numbers:
	// http://www.macyourself.com/2010/02/17/what-is-free-wired-active-and-inactive-system-memory-ram/
	Active   uint64 `json:"active"`
	Inactive uint64 `json:"inactive"`
	Wired    uint64 `json:"wired"`

	// Linux specific numbers
	// https://www.centos.org/docs/5/html/5.1/Deployment_Guide/s2-proc-meminfo.html
	// https://www.kernel.org/doc/Documentation/filesystems/proc.txt
	// https://www.kernel.org/doc/Documentation/vm/overcommit-accounting
	Buffers      uint64 `json:"buffers"`
	Cached       uint64 `json:"cached"`
	Writeback    uint64 `json:"writeback"`
	Dirty        uint64 `json:"dirty"`
	WritebackTmp uint64 `json:"writebacktmp"`
	Shared       uint64 `json:"shared"`
	Slab         uint64 `json:"slab"`
	PageTables   uint64 `json:"pagetables"`
	SwapCached   uint64 `json:"swapcached"`
	CommitLimit  uint64 `json:"commitlimit"`
	CommittedAS  uint64 `json:"committedas"`
}

type SwapMemoryStat struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"usedPercent"`
	Sin         uint64  `json:"sin"`
	Sout        uint64  `json:"sout"`
}

func (m VirtualMemoryStat) String() string {
	s, _ := json.Marshal(m)
	return string(s)
}

func (m SwapMemoryStat) String() string {
	s, _ := json.Marshal(m)
	return string(s)
}

func VirtualMemory() (*VirtualMemoryStat, error) {
	return VirtualMemoryWithContext(context.Background())
}

func VirtualMemoryWithContext(ctx context.Context) (*VirtualMemoryStat, error) {
	filename := common.HostProc("meminfo")
	lines, _ := common.ReadLines(filename)
	// flag if MemAvailable is in /proc/meminfo (kernel 3.14+)
	memavail := false

	ret := &VirtualMemoryStat{}
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) != 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		value := strings.TrimSpace(fields[1])
		value = strings.Replace(value, " kB", "", -1)

		t, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return ret, err
		}
		switch key {
		case "MemTotal":
			ret.Total = t
		case "MemFree":
			ret.Free = t
		case "MemAvailable":
			memavail = true
			ret.Available = t
		case "Buffers":
			ret.Buffers = t
		case "Cached":
			ret.Cached = t
		case "Active":
			ret.Active = t
		case "Inactive":
			ret.Inactive = t
		case "Writeback":
			ret.Writeback = t
		case "WritebackTmp":
			ret.WritebackTmp = t
		case "Dirty":
			ret.Dirty = t
		case "Shmem":
			ret.Shared = t
		case "Slab":
			ret.Slab = t
		case "PageTables":
			ret.PageTables = t
		case "SwapCached":
			ret.SwapCached = t
		case "CommitLimit":
			ret.CommitLimit = t
		case "Committed_AS":
			ret.CommittedAS = t
		}
	}
	if !memavail {
		ret.Available = ret.Free + ret.Buffers + ret.Cached
	}
	ret.Used = ret.Total - ret.Free - ret.Buffers - ret.Cached
	ret.UsedPercent = float64(ret.Used) / float64(ret.Total) * 100.0

	return ret, nil
}

func SwapMemory() (*SwapMemoryStat, error) {
	return SwapMemoryWithContext(context.Background())
}

func SwapMemoryWithContext(ctx context.Context) (*SwapMemoryStat, error) {
	sysinfo := &unix.Sysinfo_t{}

	if err := unix.Sysinfo(sysinfo); err != nil {
		return nil, err
	}
	ret := &SwapMemoryStat{
		Total: uint64(sysinfo.Totalswap) * uint64(sysinfo.Unit),
		Free:  uint64(sysinfo.Freeswap) * uint64(sysinfo.Unit),
	}
	ret.Used = ret.Total - ret.Free
	//check Infinity
	if ret.Total != 0 {
		ret.UsedPercent = float64(ret.Total-ret.Free) / float64(ret.Total) * 100.0
	} else {
		ret.UsedPercent = 0
	}
	filename := common.HostProc("vmstat")
	lines, _ := common.ReadLines(filename)
	for _, l := range lines {
		fields := strings.Fields(l)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "pswpin":
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				continue
			}
			ret.Sin = value * 4 * 1024
		case "pswpout":
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				continue
			}
			ret.Sout = value * 4 * 1024
		}
	}
	return ret, nil
}
