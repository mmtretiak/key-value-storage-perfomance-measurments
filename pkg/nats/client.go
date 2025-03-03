package nats

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"strings"
)

type client struct {
	conn   *nats.Conn
	js     nats.JetStreamContext
	bucket nats.KeyValue
}

func NewClient(url, bucket string) (*client, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}

	js, err := conn.JetStream()
	if err != nil {
		return nil, err
	}

	osCh := js.KeyValueStores()

	found := false
	for osInfo := range osCh {
		if osInfo.Bucket() == bucket {
			found = true
		}
	}

	var os nats.KeyValue
	if !found {
		os, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket:  bucket,
			Storage: nats.MemoryStorage,
		})
		if err != nil {
			return nil, err
		}
	} else {
		os, err = js.KeyValue(bucket)
		if err != nil {
			return nil, err
		}
	}

	return &client{
		conn:   conn,
		js:     js,
		bucket: os,
	}, nil
}

func (c *client) Set(ctx context.Context, key, value string) error {
	_, err := c.bucket.PutString(key, value)
	if err != nil {
		return err
	}

	return nil
}

func (c *client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.bucket.Get(key)
	if err != nil {
		fmt.Println(key, err)
		return "", err
	}

	return string(value.Value()), nil
}

func (c *client) FlushAll(ctx context.Context) error {
	obj, err := c.bucket.ListKeys()
	if err != nil {
		return err
	}

	for key := range obj.Keys() {
		if err := c.bucket.Delete(key); err != nil {
			return err
		}
	}

	return nil
}

func (c *client) DeleteByPrefix(ctx context.Context, prefix string) error {
	obj, err := c.bucket.ListKeys()
	if err != nil {
		return err
	}

	for key := range obj.Keys() {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		if err := c.bucket.Delete(key); err != nil {
			return err
		}
	}

	return nil
}
