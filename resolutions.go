package main

import (
	"fmt"
	"sort"
)

func GetTopResolutions(currentWidth, currentHeight int) []string {
	resolutions := []struct {
		width, height int
	}{
		{3840, 2160}, // 4K
		{2560, 1440}, // QHD
		{1920, 1080}, // Full HD
		{1680, 1050}, // WSXGA+
		{1600, 900},  // HD+
		{1366, 768},  // HD
		{1280, 1024}, // SXGA
		{1280, 800},  // WXGA
		{1280, 720},  // HD
		{1024, 768},  // XGA
		{800, 600},   // SVGA
		{640, 480},   // VGA
		{320, 240},   // QVGA
		{2560, 1600}, // WQXGA
		{2048, 1536}, // QXGA
		{1920, 1200}, // WUXGA
		{1440, 900},  // WXGA+
		{1280, 960},  // SXGA
		{1024, 600},  // WSVGA
		{800, 480},   // WVGA
		{640, 360},   // nHD
	}

	// Filter and sort resolutions
	var filteredResolutions []string
	for _, res := range resolutions {
		if res.width <= currentWidth && res.height <= currentHeight {
			filteredResolutions = append(filteredResolutions, fmt.Sprintf("%dx%d", res.width, res.height))
		}
	}

	// Sort resolutions by size (width * height)
	sort.Slice(filteredResolutions, func(i, j int) bool {
		width1, height1 := parseResolution(filteredResolutions[i])
		width2, height2 := parseResolution(filteredResolutions[j])
		return (width1 * height1) > (width2 * height2)
	})

	return filteredResolutions
}

func parseResolution(res string) (int, int) {
	var width, height int
	fmt.Sscanf(res, "%dx%d", &width, &height)
	return width, height
}
