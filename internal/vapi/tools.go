package vapi

import (
	"mime"
	"path/filepath"
)

func getMimeType(filename string) string {
	ext := filepath.Ext(filename)
	return mime.TypeByExtension(ext)
}
