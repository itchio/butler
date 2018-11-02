package damage

import (
	"fmt"
	"strings"

	"github.com/itchio/damage/hdiutil"
	"github.com/itchio/httpkit/progress"
	"github.com/pkg/errors"
)

type DiskInfo struct {
	Format            string              `plist:"Format"`
	FormatDescription string              `plist:"Format Description"`
	Partitions        Partitions          `plist:"partitions"`
	Properties        DiskProperties      `plist:"Properties"`
	SizeInformation   DiskSizeInformation `plist:"Size Information"`
}

type DiskSizeInformation struct {
	CompressedBytes    int64 `plist:"Compressed Bytes"`
	TotalBytes         int64 `plist:"Total Bytes"`
	TotalNonEmptyBytes int64 `plist:"Total Non-Empty Bytes"`
}

type DiskProperties struct {
	// true if has a software license agreement
	SoftwareLicenseAgreement bool `plist:"Software License Agreement" json:",omitempty"`
	// true if is compressed
	Checksummed bool `plist:"Checksummed" json:",omitempty"`
	// true if includes checksums
	Compressed bool `plist:"Compressed" json:",omitempty"`
	Encrypted  bool `plist:"Encrypted" json:",omitempty"`
}

type Partitions struct {
	Partitions []Partition `plist:"partitions"`
}

type Partition struct {
	Hint        string                 `plist:"partition-hint" json:",omitempty"`
	Name        string                 `plist:"partition-name"`
	Length      int64                  `plist:"partition-length"`
	Synthesized bool                   `plist:"synthesized" json:",omitempty"`
	Filesystems map[string]interface{} `plist:"partition-filesystems" json:",omitempty"`
}

func GetDiskInfo(host hdiutil.Host, dmgpath string) (*DiskInfo, error) {
	var info DiskInfo
	err := host.Command("imageinfo").WithArgs("-plist", dmgpath).RunAndDecode(&info)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &info, nil
}

func (di *DiskInfo) String() string {
	var lines []string
	summary := fmt.Sprintf("%s image", di.Format)
	if di.Properties.Checksummed {
		summary += " [checksum]"
	}
	if di.Properties.Compressed {
		summary += " [compress]"
	}
	if di.Properties.Encrypted {
		summary += " [encrypt]"
	}
	if di.Properties.SoftwareLicenseAgreement {
		summary += " [sla]"
	}
	lines = append(lines, summary)
	lines = append(lines, fmt.Sprintf("Format:\t%s", di.FormatDescription))

	size := fmt.Sprintf("Size:\t%s compressed, %s decompressed",
		progress.FormatBytes(di.SizeInformation.CompressedBytes),
		progress.FormatBytes(di.SizeInformation.TotalNonEmptyBytes),
	)
	lines = append(lines, size)

	for _, p := range di.Partitions.Partitions {
		fs := p.Filesystems
		if len(fs) == 0 {
			continue
		}
		var fsNames []string
		for fsName := range fs {
			fsNames = append(fsNames, fsName)
		}

		lines = append(lines, fmt.Sprintf("FS:\t%s",
			strings.Join(fsNames, "-"),
		))
	}

	return strings.Join(lines, "\n")
}
