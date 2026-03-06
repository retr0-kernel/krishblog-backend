package analytics

import (
	"strings"

	"github.com/mssola/useragent"
)

// DeviceType represents the broad device category.
type DeviceType string

const (
	DeviceDesktop DeviceType = "desktop"
	DeviceTablet  DeviceType = "tablet"
	DeviceMobile  DeviceType = "mobile"
	DeviceUnknown DeviceType = "unknown"
)

// ParsedUA holds the parsed User-Agent fields we store.
type ParsedUA struct {
	Device  DeviceType
	Browser string
	OS      string
}

// ParseUserAgent extracts device, browser, and OS from a raw UA string.
func ParseUserAgent(rawUA string) ParsedUA {
	if rawUA == "" {
		return ParsedUA{Device: DeviceUnknown, Browser: "unknown", OS: "unknown"}
	}

	ua := useragent.New(rawUA)

	device := DeviceDesktop
	if ua.Mobile() {
		lower := strings.ToLower(rawUA)
		if strings.Contains(lower, "tablet") ||
			strings.Contains(lower, "ipad") ||
			strings.Contains(lower, "kindle") ||
			strings.Contains(lower, "playbook") {
			device = DeviceTablet
		} else {
			device = DeviceMobile
		}
	}

	browserName, _ := ua.Browser()
	if browserName == "" {
		browserName = "unknown"
	}

	osInfo := ua.OS()
	if osInfo == "" {
		osInfo = "unknown"
	}

	lower := strings.ToLower(osInfo)
	switch {
	case strings.Contains(lower, "windows"):
		osInfo = "Windows"
	case strings.Contains(lower, "mac os"):
		osInfo = "macOS"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad"):
		osInfo = "iOS"
	case strings.Contains(lower, "android"):
		osInfo = "Android"
	case strings.Contains(lower, "linux"):
		osInfo = "Linux"
	case strings.Contains(lower, "chrome os"):
		osInfo = "ChromeOS"
	}

	return ParsedUA{
		Device:  device,
		Browser: browserName,
		OS:      osInfo,
	}
}
