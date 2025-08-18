package cache

import (
	"net/url"
)

func NewQueueItem(loc *url.URL) *RedisQueueItem {
	var locString string = loc.String()
	var retries int32 = 0
	return &RedisQueueItem{
		Location: &locString,
		Retries:  &retries,
	}
}

func (item *RedisQueueItem) SetRetries(retries int32) {
	item.Retries = &retries
}
