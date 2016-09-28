**HOW TO USE CACHE PACKAGE**

Use it to elegantly retrieve something from the cache if it's already there, OR fetch it if it isn't there. If it's not there, the package will elegantly store it in the cache for next time.
It need not be used for database queries. It could be used for any potentially slow operation.


1) Create a file to make it easier to generate keys for the cache. If you intend to cache database queries, then the function should have your query variables as parameters.

```
===keygenerator.go===
import (
  "fmt"
 )

func GetUpcomingEvent(customerId uint64) string {
	return "GetUpcomingEvent." + fmt.Sprintf("%d", customerId)
}

```


2) When you want to use the cache do something like this:

```
cacheKey := cache.GetUpcomingEvent(appScopedId)

type FetchModel struct {
		EventId        uint32     `gorm:"column:event_id;"`
		TimeStart      time.Time  `gorm:"column:time_start;"`
		RegisteredTime *time.Time `gorm:"column:registered_time;"`
}


slowFetchUpcomingEvent := func() (interface{}, error) {
    gorm, _ := g.Open("mysql", "XXX")

    serverUTCTimeString := time.Now().UTC().Format("2006-01-02 15:04:05")
		
    sql_stmt :=
			" SELECT event_id, time_start, registered_time" +
				" FROM " + c.PIVOT_CUSTOMER_EVENT +
				" INNER JOIN " + c.TEVENT +
				" ON " + c.EVENT + ".id = " + c.PIVOT_CUSTOMER_EVENT + ".event_id" +
				" WHERE " + c.PIVOT_CUSTOMER_EVENT + ".customer_id = ?" +
				" AND " + c.EVENT + fmt.Sprintf(".time_end > '%s'", serverUTCTimeString) +
				" AND " + c.EVENT + ".cancelled = ?" +
				" ORDER BY time_start" +
				" LIMIT 1"

		fm := &FetchModel{}
		if err := gorm.Raw(sql_stmt, appScopedId, false).Scan(fm).Error; err != nil {
			if err == g.ErrRecordNotFound {
				return nil, nil //We have a result (no event upcoming)
			} else {
				return nil, err //Error occurred
			}
		}
		return fm, nil //We have a result
}

p, err := cache.Remember(appengine.NewContext(r), cacheKey, time.Duration(5*time.Minute), slowFetchUpcomingEvent)
if err != nil {
  e.ReturnError(w, e.UnknownDatabaseError, err.Error())
  return
}

if p == nil {
  //No upcoming event
  //...
} else {
  fm := p.(*FetchModel)
  //Use fm
}
```
 
