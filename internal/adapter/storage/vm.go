package storage

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	"github.com/sirupsen/logrus"
)

const maxFileSizeBeforeResize = 1 * 1024 * 1024

type VMStorage struct {
	basePath string
	log      *logrus.Logger
}

func NewVMStorage(basePath string, log *logrus.Logger) *VMStorage {
	return &VMStorage{basePath: basePath, log: log}
}

func (s *VMStorage) Upload(_ context.Context, objectName string, data []byte, _ string) (string, error) {
	reader, err := prepareReader(data, s.log)
	if err != nil {
		return "", err
	}

	dir := filepath.Join(s.basePath, filepath.Dir(objectName))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create storage directory: %w", err)
	}

	filePath := filepath.Join(s.basePath, objectName)
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, reader); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	publicURL := fmt.Sprintf("/storage/%s", objectName)
	s.log.Infof("VM upload successful: %s", publicURL)
	return publicURL, nil
}

func prepareReader(data []byte, log *logrus.Logger) (io.Reader, error) {
	if len(data) <= maxFileSizeBeforeResize {
		return bytes.NewReader(data), nil
	}

	log.Infof("file size %d exceeds limit; resizing", len(data))
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image for resize: %w", err)
	}

	if img.Bounds().Dx() > 1000 {
		img = imaging.Resize(img, 1000, 0, imaging.Lanczos)
	}

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, img, imaging.JPEG); err != nil {
		return nil, fmt.Errorf("encode resized image: %w", err)
	}

	log.Infof("resize complete; new size %d bytes", buf.Len())
	return bytes.NewReader(buf.Bytes()), nil
}
