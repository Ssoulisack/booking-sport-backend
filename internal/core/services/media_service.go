package services

import (
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/internal/core/port/output"
	"backend/pkg/response"
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"regexp"

	"github.com/google/uuid"
)

const maxUploadBytes = 2 << 20 // 2 MB

var (
	// folders a file may be uploaded to
	mediaFolders = map[string]bool{"sports": true, "coaches": true, "classes": true}
	// content type (detected from the bytes, not the filename) -> extension
	imageExtensions = map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}
	// the only shape of key the server ever generates
	mediaKeyPattern = regexp.MustCompile(`^(sports|coaches|classes)/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(jpg|png|webp)$`)
)

type MediaServiceImpl struct {
	storage output.FileStorage
}

func NewMediaService(storage output.FileStorage) input.MediaService {
	return &MediaServiceImpl{storage: storage}
}

// Upload accepts jpg / png / webp up to 2 MB. The key is generated here, so callers cannot choose paths.
func (s *MediaServiceImpl) Upload(ctx context.Context, folder string, file io.Reader) (*dto.MediaDTO, error) {
	if !mediaFolders[folder] {
		return nil, badRequest("INVALID_FOLDER")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		return nil, badRequest("INVALID_FILE")
	}
	if len(data) == 0 {
		return nil, badRequest("FILE_IS_EMPTY")
	}
	if len(data) > maxUploadBytes {
		return nil, response.NewError(http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE")
	}
	contentType := http.DetectContentType(data)
	ext, ok := imageExtensions[contentType]
	if !ok {
		return nil, response.NewError(http.StatusUnsupportedMediaType, "UNSUPPORTED_FILE_TYPE")
	}

	key := folder + "/" + uuid.NewString() + "." + ext
	if err := s.storage.Upload(ctx, key, contentType, bytes.NewReader(data)); err != nil {
		log.Printf("❌ file upload failed: %v", err)
		return nil, response.NewError(http.StatusBadGateway, "STORAGE_UPLOAD_FAILED")
	}
	return &dto.MediaDTO{Key: key, URL: s.storage.URL(key)}, nil
}

func (s *MediaServiceImpl) Delete(ctx context.Context, key string) error {
	if !isValidMediaKey(key) {
		return badRequest("INVALID_FILE_KEY")
	}
	if err := s.storage.Delete(ctx, key); err != nil {
		log.Printf("❌ file delete failed: %v", err)
		return response.NewError(http.StatusBadGateway, "STORAGE_DELETE_FAILED")
	}
	return nil
}

func isValidMediaKey(key string) bool { return mediaKeyPattern.MatchString(key) }
