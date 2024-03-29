package linodego

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
)

// ObjectStorageKey represents a linode object storage key object
type ObjectStorageKey struct {
	ID           int                             `json:"id"`
	Label        string                          `json:"label"`
	AccessKey    string                          `json:"access_key"`
	SecretKey    string                          `json:"secret_key"`
	Limited      bool                            `json:"limited"`
	BucketAccess *[]ObjectStorageKeyBucketAccess `json:"bucket_access"`
}

// ObjectStorageKeyBucketAccess represents a linode limited object storage key's bucket access
type ObjectStorageKeyBucketAccess struct {
	Cluster     string `json:"cluster"`
	BucketName  string `json:"bucket_name"`
	Permissions string `json:"permissions"`
}

// ObjectStorageKeyCreateOptions fields are those accepted by CreateObjectStorageKey
type ObjectStorageKeyCreateOptions struct {
	Label        string                          `json:"label"`
	BucketAccess *[]ObjectStorageKeyBucketAccess `json:"bucket_access"`
}

// ObjectStorageKeyUpdateOptions fields are those accepted by UpdateObjectStorageKey
type ObjectStorageKeyUpdateOptions struct {
	Label string `json:"label"`
}

// ObjectStorageKeysPagedResponse represents a linode API response for listing
type ObjectStorageKeysPagedResponse struct {
	*PageOptions
	Data []ObjectStorageKey `json:"data"`
}

// endpoint gets the endpoint URL for Object Storage keys
func (ObjectStorageKeysPagedResponse) endpoint(c *Client, _ ...any) string {
	endpoint, err := c.ObjectStorageKeys.Endpoint()
	if err != nil {
		panic(err)
	}
	return endpoint
}

func (resp *ObjectStorageKeysPagedResponse) castResult(r *resty.Request, e string) (int, int, error) {
	res, err := coupleAPIErrors(r.SetResult(ObjectStorageKeysPagedResponse{}).Get(e))
	if err != nil {
		return 0, 0, err
	}
	castedRes := res.Result().(*ObjectStorageKeysPagedResponse)
	resp.Data = append(resp.Data, castedRes.Data...)
	return castedRes.Pages, castedRes.Results, nil
}

// ListObjectStorageKeys lists ObjectStorageKeys
func (c *Client) ListObjectStorageKeys(ctx context.Context, opts *ListOptions) ([]ObjectStorageKey, error) {
	response := ObjectStorageKeysPagedResponse{}
	err := c.listHelper(ctx, &response, opts)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

// CreateObjectStorageKey creates a ObjectStorageKey
func (c *Client) CreateObjectStorageKey(ctx context.Context, createOpts ObjectStorageKeyCreateOptions) (*ObjectStorageKey, error) {
	var body string
	e, err := c.ObjectStorageKeys.Endpoint()
	if err != nil {
		return nil, err
	}

	req := c.R(ctx).SetResult(&ObjectStorageKey{})

	if bodyData, err := json.Marshal(createOpts); err == nil {
		body = string(bodyData)
	} else {
		return nil, NewError(err)
	}

	r, err := coupleAPIErrors(req.
		SetBody(body).
		Post(e))
	if err != nil {
		return nil, err
	}
	return r.Result().(*ObjectStorageKey), nil
}

// GetObjectStorageKey gets the object storage key with the provided ID
func (c *Client) GetObjectStorageKey(ctx context.Context, id int) (*ObjectStorageKey, error) {
	e, err := c.ObjectStorageKeys.Endpoint()
	if err != nil {
		return nil, err
	}
	e = fmt.Sprintf("%s/%d", e, id)
	r, err := coupleAPIErrors(c.R(ctx).SetResult(&ObjectStorageKey{}).Get(e))
	if err != nil {
		return nil, err
	}
	return r.Result().(*ObjectStorageKey), nil
}

// UpdateObjectStorageKey updates the object storage key with the specified id
func (c *Client) UpdateObjectStorageKey(ctx context.Context, id int, updateOpts ObjectStorageKeyUpdateOptions) (*ObjectStorageKey, error) {
	var body string
	e, err := c.ObjectStorageKeys.Endpoint()
	if err != nil {
		return nil, err
	}
	e = fmt.Sprintf("%s/%d", e, id)

	req := c.R(ctx).SetResult(&ObjectStorageKey{})

	if bodyData, err := json.Marshal(updateOpts); err == nil {
		body = string(bodyData)
	} else {
		return nil, NewError(err)
	}

	r, err := coupleAPIErrors(req.
		SetBody(body).
		Put(e))
	if err != nil {
		return nil, err
	}
	return r.Result().(*ObjectStorageKey), nil
}

// DeleteObjectStorageKey deletes the ObjectStorageKey with the specified id
func (c *Client) DeleteObjectStorageKey(ctx context.Context, id int) error {
	e, err := c.ObjectStorageKeys.Endpoint()
	if err != nil {
		return err
	}
	e = fmt.Sprintf("%s/%d", e, id)

	_, err = coupleAPIErrors(c.R(ctx).Delete(e))
	return err
}
