package image

import "strings"

const defaultSize = "1:1"

var allowedSizes = []string{"1:1", "16:9", "9:16"}

func normalize(req GenerateRequest) GenerateRequest {
	req.Size = normalizeSize(req.Size)
	return req
}

func normalizeSize(size string) string {
	size = strings.TrimSpace(size)
	for _, allowed := range allowedSizes {
		if size == allowed {
			return size
		}
	}
	return defaultSize
}
