package receipt

import (
	"bytes"
	"fmt"
	"net/http"
)

type MinIOStore struct {
	endpoint, bucket string
	client           *http.Client
}

func NewMinIOStore(endpoint, bucket string) *MinIOStore {
	return &MinIOStore{endpoint: endpoint, bucket: bucket, client: &http.Client{}}
}
func (m *MinIOStore) PutWebP(key string, data []byte) error {
	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s/%s", m.endpoint, m.bucket, key), bytes.NewReader(data))
	req.Header.Set("Content-Type", "image/webp")
	res, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("minio put failed: %d", res.StatusCode)
	}
	return nil
}
