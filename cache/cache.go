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

func Remember(ctx context.Context, key string, expiration time.Duration, p SlowRetrieve, disable ...bool) (interface{}, error) {

	//For debugging, you can disable cache
	if len(disable) != 0 && disable[0] == true {
		return p(ctx)
	}

	//Check if item exists
	var v interface{}
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
