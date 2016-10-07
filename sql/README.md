**INTRODUCTION**

This package is an experimental package designed to allow Google App Engine (Standard Environment) to use an non-cloudSQL database hosted elsewhere.

The key is to use Socket API. It is trivial to connect to an external database without connection pooling. That is not recommended because it involves making a connection to the database and and then closing it for every request cycle.

Unfortunately, socket API can't be easily used with connection pooling using the `database/sql` + `go-sql-driver/mysql` package combination (the driver requires heavy modifications).

This is an attempt to do the connection pooling outside of database/sql.


**HOW TO USE SQL PACKAGE**

I will assume you are using the `o-sql-driver/mysql` driver.

```
import (
	"google.golang.org/appengine"
	exSql "github.com/pjebs/GAE-Toolkit-Go/sql"
)
```

Firstly:

At the start of the request cycle add this (you can create a middleware at top of stack to make it easier):

```
//NB: "external" in the RegisterDial() and sql.Open()
//sql.Open("mysql", "username:password@external(your-amazonaws-uri.com:3306)/dbname")

ctx := appengine.NewContext(req)
mysql.RegisterDial("external", exSql.Dial(req, 10))
```

Secondly:

Create a Request Handler:

```
import (
	"fmt"
	"log"
	"database/sql"
	"net/http"
	exSql "github.com/pjebs/GAE-Toolkit-Go/sql"
	"sync"
)

func Test(w http.ResponseWriter, req *http.Request) {
	var wg sync.WaitGroup

	wg.Add(50)
	for i := 0; i < 50; i++ {

		go func(i int) {
			defer wg.Done()
			db, err := exSql.Open("mysql", username:password@external(your-amazonaws-uri.com:3306)/dbname", req)
			log.Println("Opened Database:", i)
			if err != nil {
				log.Println("Open error:", err)
				fmt.Fprintln(w, "Open error:", err)
				return
			}
			defer db.Close()

			id := 123
			var username string
			err = db.QueryRow("SELECT username FROM hello WHERE id=?", id).Scan(&username)
			switch {
			case err == sql.ErrNoRows:
				log.Printf("No user with that ID.")
				fmt.Fprintln(w, "Open error:", err)
			case err != nil:
				log.Println("error:", err)
				fmt.Fprintln(w, "error:", err)
			default:
				log.Printf("Username is %s\n", username)
				fmt.Printf("Username is %s\n", username)
			}
		}(i)
	}

	wg.Wait()

	log.Println("Request Finished")
}

```


**ISSUES**

This library 'mostly' works. There are many issues which I don't have time to solve.
If you can solve it, let me know.

The code is purely a 'quick-and-dirty' proof of concept.
I attempted to make the api consistent with also using cloudSQL. Hence the interface is not designed as well as it could be. (If you leave out the `req` in the `Open` function, your cloudsql code should work perfectly as before.)

Obviously this library should not have to worry about cloudSQL compatibility in the final version.

There appears to be issues with the mutex locks and the connection pooling library used: `"gopkg.in/fatih/pool.v2"`.

I think the better approach may be to use something like this: (Redis library's pool)[https://godoc.org/github.com/garyburd/redigo/redis#Pool]


**CONTACT**

Contact me on pj@pjebs.com.au if you want to try and finish this work off. I may have extra insight from my failures.