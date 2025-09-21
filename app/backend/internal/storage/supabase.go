package storage

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "net/http"
)

// SupabaseClient interacts with Supabase Storage via REST API.
type SupabaseClient struct {
    baseURL    string
    bucket     string
    serviceKey string
    httpClient *http.Client
}

func NewSupabaseClient(baseURL, bucket, serviceKey string) *SupabaseClient {
    return &SupabaseClient{
        baseURL:    fmt.Sprintf("%s/storage/v1", baseURL),
        bucket:     bucket,
        serviceKey: serviceKey,
        httpClient: http.DefaultClient,
    }
}

func (c *SupabaseClient) Upload(ctx context.Context, objectPath string, body []byte, contentType string) error {
    url := fmt.Sprintf("%s/object/%s/%s", c.baseURL, c.bucket, objectPath)
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.serviceKey))
    req.Header.Set("Content-Type", contentType)
    req.Header.Set("x-upsert", "true")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= http.StatusBadRequest {
        data, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("supabase upload failed: %s", string(data))
    }
    return nil
}

func (c *SupabaseClient) Delete(ctx context.Context, objectPath string) error {
    url := fmt.Sprintf("%s/object/%s/%s", c.baseURL, c.bucket, objectPath)
    req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.serviceKey))

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= http.StatusBadRequest {
        data, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("supabase delete failed: %s", string(data))
    }
    return nil
}

func (c *SupabaseClient) Download(ctx context.Context, objectPath string) ([]byte, string, error) {
    url := fmt.Sprintf("%s/object/%s/%s", c.baseURL, c.bucket, objectPath)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, "", err
    }
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.serviceKey))

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= http.StatusBadRequest {
        data, _ := io.ReadAll(resp.Body)
        return nil, "", fmt.Errorf("supabase download failed: %s", string(data))
    }

    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, "", err
    }
    return data, resp.Header.Get("Content-Type"), nil
}
