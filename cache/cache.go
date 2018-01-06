package cache

import (
	"context"
	"encoding/gob"
	"log"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/memcache"
)

//http://www.funcmain.com/gob_encoding_an_interface
//http://stackoverflow.com/questions/13264555/store-an-object-in-memcache-of-gae-in-go

type SlowRetrieve func(ctx context.Context) (interface{}, error)

type CacheOptions struct {
	// DisableCacheUsage disables the cache. (Usually for debugging)
	DisableCacheUsage bool
	// UseFreshData will ignore content in the cache and pull fresh data.
	// The pulled data will subsequently be saved in the cache
	UseFreshData bool
	//Turn on logging
	Log bool
}

//Options:
//param 1 (bool): disable caching. (Usually for debugging)
//Param 2 (bool): Obtain fresh copy. Ignore content in cache but store fresh copy in cache.
//NB: In order for Param 2 to be activated, param 1 must be false.
func Remember(ctx context.Context, key string, expiration time.Duration, p SlowRetrieve, options ...CacheOptions) (interface{}, error) {

	var (
		disableCache bool
		fresh        bool
		mustLog      bool
	)

	if appengine.IsDevAppServer() {
		mustLog = true
	}

	if options != nil {
		disableCache = options[0].DisableCacheUsage
		fresh = options[0].UseFreshData
		mustLog = options[0].Log
	}

	//For debugging, you can disable cache
	if disableCache {
		return p(ctx)
	}

	var v interface{}

	if fresh {
		if mustLog {
			log.Println("\x1b[31mGrabbing (fresh) from SlowRetrieve key:", key, "\x1b[39;49m")
		}
		goto fresh
	}

	//Check if item exists
	if _, err := memcache.Gob.Get(ctx, key, &v); err == nil {
		//Item exists in cache
		if mustLog {
			log.Println("\x1b[36mFound in Cache key:", key, "\x1b[39;49m")
		}
		return v, nil
	} else {
		if mustLog {
			log.Println("\x1b[31mGrabbing from SlowRetrieve key:", key, err, "\x1b[39;49m")
		}
	}

fresh:
	//Item does not exist in cache so grab it from the persistent store
	itemToStore, err := p(ctx)
	if err != nil {
		return nil, err
	}
	func(itemToStore interface{}) {
		defer func() {
			recover()
		}()
		gob.Register(itemToStore)
	}(itemToStore)

	//Store item in Cache
	item := &memcache.Item{
		Key:        key,
		Object:     &itemToStore,
		Expiration: expiration,
	}

	err = memcache.Gob.Set(ctx, item)
	if err != nil {
		//Memcache storage failed
		if mustLog {
			log.Println("\x1b[31mCould not store item to memcache key:", key, err, itemToStore, "\x1b[39;49m")
		}
	}
	return itemToStore, nil
}

//Delete key from memcache
func Delete(ctx context.Context, key string) error {
	return memcache.Delete(ctx, key)
}

func DeleteMulti(ctx context.Context, keys []string) error {
	return memcache.DeleteMulti(ctx, keys)
}
