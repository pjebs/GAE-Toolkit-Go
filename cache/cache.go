package cache

import (
	"encoding/gob"
	"golang.org/x/net/context"
	"log"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/memcache"
)

//http://www.funcmain.com/gob_encoding_an_interface
//http://stackoverflow.com/questions/13264555/store-an-object-in-memcache-of-gae-in-go

type SlowRetrieve func(ctx context.Context) (interface{}, error)

//Options:
//param 1 (bool): disable caching. (Usually for debugging)
//Param 2 (bool): Obtain fresh copy. Ignore content in cache but store fresh copy in cache.
//NB: In order for Param 2 to be activated, param 1 must be false.
func Remember(ctx context.Context, key string, expiration time.Duration, p SlowRetrieve, options ...bool) (interface{}, error) {

	disableCache := false
	fresh := false

	if len(options) != 0 {
		disableCache = options[0]
		if len(options) >= 2 {
			fresh = options[1]
		}
	}

	//For debugging, you can disable cache
	if disableCache {
		return p(ctx)
	}

	var v interface{}

	if fresh {
		if appengine.IsDevAppServer() {
			log.Println("\x1b[31mGrabbing (fresh) from SlowRetrieve key:", key, "\x1b[39;49m")
		}
		goto fresh
	}

	//Check if item exists
	if _, err := memcache.Gob.Get(ctx, key, &v); err == nil {
		//Item exists in cache
		if appengine.IsDevAppServer() {
			log.Println("\x1b[36mFound in Cache key:", key, "\x1b[39;49m")
		}
		return v, nil
	} else {
		if appengine.IsDevAppServer() {
			log.Println("\x1b[31mGrabbing from SlowRetrieve key:", key, err, "\x1b[39;49m")
		}
	}

fresh:
	//Item does not exist in cache so grab it from the persistent store
	itemToStore, err := p(ctx)
	func(itemToStore interface{}) {
		defer func() {
			recover()
		}()
		gob.Register(itemToStore)
	}(itemToStore)
	if err != nil {
		return nil, err
	}

	//Store item in Cache
	item := &memcache.Item{
		Key:        key,
		Object:     &itemToStore,
		Expiration: expiration,
	}

	err = memcache.Gob.Set(ctx, item)
	if err != nil {
		//Memcache storage failed
		if appengine.IsDevAppServer() {
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
