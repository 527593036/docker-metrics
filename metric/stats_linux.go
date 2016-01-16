package metric

import (
	"bufio"
	"fmt"
	//"log"
	"strings"
)

// Get disk stats from /proc/{pid}/net/dev
func (self *Metric) getNetStats(result map[string]uint64) (err error) {
	s := bufio.NewScanner(self.statNetFile)
	defer self.statNetFile.Seek(0, 0)
	var d uint64
	for s.Scan() {
		var name string
		var n [8]uint64
		text := s.Text()
		if strings.Index(text, ":") < 1 {
			continue
		}
		ts := strings.Split(text, ":")
		fmt.Sscanf(ts[0], "%s", &name)
		if !strings.HasPrefix(name, gset.vlanPrefix) && name != gset.defaultVlan {
			continue
		}
		fmt.Sscanf(ts[1],
			"%d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d",
			&n[0], &n[1], &n[2], &n[3], &d, &d, &d, &d,
			&n[4], &n[5], &n[6], &n[7], &d, &d, &d, &d,
		)
		result[name+".inbytes"] = n[0]
		result[name+".inpackets"] = n[1]
		result[name+".inerrs"] = n[2]
		result[name+".indrop"] = n[3]
		result[name+".outbytes"] = n[4]
		result[name+".outpackets"] = n[5]
		result[name+".outerrs"] = n[6]
		result[name+".outdrop"] = n[7]
	}
	return
}

// Get disk stats from /proc/{pid}/io
// may failed by permission denied.
func (self *Metric) getDiskStats(result map[string]uint64) (err error) {
	s := bufio.NewScanner(self.statDiskFile)
	defer self.statDiskFile.Seek(0, 0)
	var d uint64
	for s.Scan() {
		var name string
		text := s.Text()
		if strings.Index(text, ":") < 1 {
			continue
		}
		ts := strings.Split(text, ":")
		fmt.Sscanf(ts[0], "%s", &name)
		name = strings.TrimSpace(name)
		if name != "write_bytes" || name != "read_bytes" {
			continue
		}
		fmt.Sscanf(ts[1], "%d", &d)
		result["disk.io."+name] = d
	}
	return

}
