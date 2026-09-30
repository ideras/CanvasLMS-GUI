package grades

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type conversionRecord struct{ sourceHash, pdfHash string }

type cachedConverter struct {
	cache     *UploadCache
	converter MarkdownConverter
}

// Converter reuses unchanged conversions across retries. Re-conversion can
// otherwise change PDF metadata/timestamps and defeat upload deduplication.
func (c *UploadCache) Converter(converter MarkdownConverter) MarkdownConverter {
	return &cachedConverter{cache: c, converter: converter}
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (c *cachedConverter) ConvertFile(input, output string) error {
	sourceHash, err := fileHash(input)
	if err != nil {
		return err
	}
	inputAbs, err := filepath.Abs(input)
	if err != nil {
		return err
	}
	outputAbs, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	key := inputAbs + "\x00" + outputAbs
	c.cache.mu.Lock()
	record, ok := c.cache.conversions[key]
	c.cache.mu.Unlock()
	if ok && record.sourceHash == sourceHash {
		if pdfHash, err := fileHash(output); err == nil && pdfHash == record.pdfHash {
			return nil
		}
	}
	if err := c.converter.ConvertFile(input, output); err != nil {
		return err
	}
	pdfHash, err := fileHash(output)
	if err != nil {
		return err
	}
	c.cache.mu.Lock()
	c.cache.conversions[key] = conversionRecord{sourceHash, pdfHash}
	c.cache.mu.Unlock()
	return nil
}
