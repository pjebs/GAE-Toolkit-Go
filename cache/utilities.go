package cache

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/memcache"
)

// StoreGob will store val in the cache after encoding it as a Gob.
func StoreGob(ctx context.Context, key string, expiration time.Duration, val interface{}, options ...CacheOptions) error {

	var mustLog bool

	if appengine.IsDevAppServer() {
		mustLog = true
	}

	if options != nil {
		mustLog = options[0].Log
	}

	if mustLog {
		log.Println("\x1b[36mStoring in cache key:", key, "\x1b[39;49m")
	}

	func(val interface{}) {
		defer func() {
			recover()
		}()
		gob.Register(val)
	}(val)

	//Store item in Cache
	item := &memcache.Item{
		Key:        key,
		Object:     &val,
		Expiration: expiration,
	}

	err := memcache.Gob.Set(ctx, item)
	if err != nil {
		//Memcache storage failed
		if mustLog {
			log.Println("\x1b[31mCould not store item to cache key:", key, err, val, "\x1b[39;49m")
		}
		return err
	}
	return nil
}

// RetrieveGob will retrieve the value from the cache.
// Type assertion will need to be performed in order to use the value
func RetrieveGob(ctx context.Context, key string, options ...CacheOptions) (interface{}, error) {

	var mustLog bool

	if appengine.IsDevAppServer() {
		mustLog = true
	}

	if options != nil {
		mustLog = options[0].Log
	}

	if mustLog {
		log.Println("\x1b[36mRetrieving from cache key:", key, "\x1b[39;49m")
	}

	//Check if item exists
	var v interface{}
	if _, err := memcache.Gob.Get(ctx, key, &v); err == nil {
		//Item exists in cache
		if mustLog {
			log.Println("\x1b[36mFound in Cache key:", key, "\x1b[39;49m")
		}
		return v, nil
	} else {
		if mustLog {
			log.Println("\x1b[31mUnable to retrieve from cache key:", key, err, "\x1b[39;49m")
		}
		return nil, err
	}
}

// Store val in cache without any Gob encoding.
func Store(ctx context.Context, key string, expiration time.Duration, val interface{}, options ...CacheOptions) error {

	var mustLog bool

	if appengine.IsDevAppServer() {
		mustLog = true
	}

	if options != nil {
		mustLog = options[0].Log
	}

	item := &memcache.Item{
		Key:        key,
		Value:      []byte(fmt.Sprint(val)),
		Expiration: expiration,
	}

	err := memcache.Set(ctx, item)
	if err != nil {
		if mustLog {
			log.Println("\x1b[31mCould not store item to cache key:", key, err, val, "\x1b[39;49m")
		}
		return err
	}

	return nil
}

// Retrieve val in cache without any Gob encoding
func Retrieve(ctx context.Context, key string, options ...CacheOptions) (string, error) {

	var mustLog bool

	if appengine.IsDevAppServer() {
		mustLog = true
	}

	if options != nil {
		mustLog = options[0].Log
	}

	if mustLog {
		log.Println("\x1b[36mRetrieving from cache key:", key, "\x1b[39;49m")
	}

	item, err := memcache.Get(ctx, key)
	if err != nil {
		if mustLog {
			log.Println("\x1b[31mUnable to retrieve from cache key:", key, err, "\x1b[39;49m")
		}
		return "", err
	}

	return string(item.Value), nil
}

// IncrementOrSet works like Increment but assumes that the key already exists in memcache.
// If the key doesn't exist in memcache, iv is used to set the initial value and delta is not applied.
// The value in the cache (after delta or set) is returned.
func IncrementOrSet(ctx context.Context, key string, delta int64, iv func(ctx context.Context) (uint64, time.Duration, error), options ...CacheOptions) (uint64, error) {

	var mustLog bool

	if appengine.IsDevAppServer() {
		mustLog = true
	}

	if options != nil {
		mustLog = options[0].Log
	}

	newValue, err := memcache.IncrementExisting(ctx, key, delta)
	if err == nil {
		if mustLog {
			log.Println("\x1b[36mIncremented key:", key, "\x1b[39;49m")
		}
		return newValue, nil
	}

	//Retrieve initial value
	initVal, expiration, err := iv(ctx)
	if err != nil {
		if mustLog {
			log.Println("\x1b[31mUnable to set initial value key:", key, err, "\x1b[39;49m")
		}
		return 0, err
	}

	err = Store(ctx, key, expiration, initVal, CacheOptions{Log: mustLog})
	if err != nil {
		return 0, err
	}

	return initVal, nil
}

func Increment(ctx context.Context, key string, delta int64, initialValue uint64) (newValue uint64, err error) {
	return memcache.Increment(ctx, key, delta, initialValue)
}

func IncrementExisting(ctx context.Context, key string, delta int64) (newValue uint64, err error) {
	return memcache.IncrementExisting(ctx, key, delta)
}

//Delete key from cache
func Delete(ctx context.Context, key string) error {
	return memcache.Delete(ctx, key)
}

func DeleteMulti(ctx context.Context, keys []string) error {
	return memcache.DeleteMulti(ctx, keys)
}
