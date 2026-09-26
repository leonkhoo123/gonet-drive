package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/h2non/bimg"
)

// LogoSizes are the icon sizes exposed to the PWA manifest. Chrome requires both
// a 192px and a 512px icon for installability, so the single admin-uploaded logo
// is resized into square PNGs at these sizes by EnsureLogoIcon.
var LogoSizes = []int{192, 512}

// GetIconPath returns the cache path for the square logo icon at the given size.
func GetIconPath(size int) string {
	return filepath.Join(filepath.Dir(GetLogoPath()), fmt.Sprintf("icon-%d.png", size))
}

// EnsureLogoIcon returns the path to a size×size PNG rendering of the current
// logo, regenerating the cached file when it is missing or older than the logo.
// The result is square (centre-cropped) so it is safe to advertise as an app
// icon of exactly that size.
func EnsureLogoIcon(size int) (string, error) {
	EnsureDefaultLogo()

	src := GetLogoPath()
	dst := GetIconPath(size)

	srcInfo, err := os.Stat(src)
	if err != nil {
		return "", fmt.Errorf("stat logo: %w", err)
	}

	// Reuse the cached icon while it is at least as new as the source logo.
	if dstInfo, err := os.Stat(dst); err == nil && !dstInfo.ModTime().Before(srcInfo.ModTime()) {
		return dst, nil
	}

	buffer, err := bimg.Read(src)
	if err != nil {
		return "", fmt.Errorf("read logo: %w", err)
	}

	icon, err := bimg.NewImage(buffer).Process(bimg.Options{
		Width:         size,
		Height:        size,
		Crop:          true,
		Enlarge:       true,
		Gravity:       bimg.GravityCentre,
		Quality:       90,
		Type:          bimg.PNG,
		StripMetadata: true,
	})
	if err != nil {
		return "", fmt.Errorf("resize logo: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return "", fmt.Errorf("create icon dir: %w", err)
	}
	if err := os.WriteFile(dst, icon, 0644); err != nil {
		return "", fmt.Errorf("write icon: %w", err)
	}

	return dst, nil
}
