package interfaces

import "mime/multipart"

type UploadRequest struct {
	File        *multipart.FileHeader `form:"file"`
	Category    string                `form:"category"`
	Type        string                `form:"type"`
	Tags        []string              `form:"tags"`
	Description string                `form:"description"`
}
