package state

import (
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type SystemInfo struct {
	Version   string
	Commit    string
	CreatedAt time.Time
	OS        string
	Arch      string
	Kernel    string
	CPUs      int
	GoVersion string
	Hostname  string
	Username  string
	UID       string
	Home      string
	Shell     string
}

func CollectSystemInfo(version, commit string, now time.Time) SystemInfo {
	info := SystemInfo{
		Version:   version,
		Commit:    commit,
		CreatedAt: now,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Kernel:    kernelRelease(),
		CPUs:      runtime.NumCPU(),
		GoVersion: runtime.Version(),
		Hostname:  "unknown",
		Username:  "unknown",
		UID:       "unknown",
		Home:      "unknown",
		Shell:     os.Getenv("SHELL"),
	}

	if host, err := os.Hostname(); err == nil && host != "" {
		info.Hostname = host
	}
	if u, err := user.Current(); err == nil {
		if u.Username != "" {
			info.Username = u.Username
		}
		if u.Uid != "" {
			info.UID = u.Uid
		}
		if u.HomeDir != "" {
			info.Home = u.HomeDir
		}
	}
	if info.Shell == "" {
		info.Shell = "unknown"
	}

	return info
}

func FormatSystemInfo(info SystemInfo) string {
	fields := [][2]string{
		{"version", info.Version},
		{"commit", info.Commit},
		{"created_at", info.CreatedAt.UTC().Format(time.RFC3339)},
		{"os", info.OS},
		{"arch", info.Arch},
		{"kernel", info.Kernel},
		{"cpus", strconv.Itoa(info.CPUs)},
		{"go", info.GoVersion},
		{"hostname", info.Hostname},
		{"user", info.Username},
		{"uid", info.UID},
		{"home", info.Home},
		{"shell", info.Shell},
	}

	var b strings.Builder
	for _, f := range fields {
		b.WriteString(f[0])
		b.WriteString(": ")
		b.WriteString(f[1])
		b.WriteByte('\n')
	}
	return b.String()
}
