package sql

import (
	"database/sql"
	"google.golang.org/appengine"
	"google.golang.org/appengine/socket"
	// "log"
	"net"
	"net/http"
	"sync"

	connPool "gopkg.in/fatih/pool.v2"
)

//Temporary place to store connection
var tempConn *socket.Conn
var connLock *sync.Mutex = &sync.Mutex{}

var pool connPool.Pool
var poolLock *sync.RWMutex = &sync.RWMutex{}

//Maximum of 12 connections to a cloudsql connection according to: https://cloud.google.com/sql/faq#sizeqps
//Presumeably 10 is a good number for an external database
var maxOpenConns int = 10

//Register with driver at start of request cycle
//eg. mysql.RegisterDial("external", sql.Dial(req, 10)), where sql is this package
func Dial(req *http.Request, setMaxOpenConns int) func(addr string) (net.Conn, error) {
	maxOpenConns = setMaxOpenConns
	return func(addr string) (net.Conn, error) {
		// log.Println("\x1b[36mDial", addr, "\x1b[39;49m")
		ctx := appengine.NewContext(req)
		var err error
		tempConn, err = socket.Dial(ctx, "tcp", addr)
		return tempConn, err
	}
}

type DB struct {
	*sql.DB
	*socket.Conn
	*connPool.PoolConn //Beware of retain cycles
}

//Used to protect the embedded sql.DB struct from having it's connection pooling settings
//interfered with.

func (db *DB) SetMaxIdleConns(n int) {
	if db.Conn == nil {
		db.DB.SetMaxIdleConns(n)
	}
}

func (db *DB) SetMaxOpenConns(n int) {
	if db.Conn == nil {
		db.DB.SetMaxOpenConns(n)
	}
}

//Closes connection if `database/sql`'s native connection pooling is used.
//Otherwise puts the connection back into pool
func (db *DB) Close() error {

	if db.Conn == nil {
		return db.DB.Close()
	}

	//Automatically puts back in connection pool
	if db.PoolConn != nil {
		db.PoolConn.Close()
		db.PoolConn = nil
	}
	// temp := db.PoolConn
	// db.PoolConn = nil //Break Retain Cycle
	// temp.Close()

	return nil
}

// Close all connections inside pool
func (db *DB) Destroy() error {
	if db.Conn == nil {
		return nil
	}

	//At the moment, this doesn't destroy retain cycle by closing all connections.
	//I may need to fork pool library.

	poolLock.Lock()
	defer poolLock.Unlock()
	if pool != nil {
		pool.Close()
	}

	return nil
}

func Open(driverName, dataSourceName string, req ...*http.Request) (*DB, error) {

	if len(req) == 0 {
		db, err := sql.Open(driverName, dataSourceName)
		if err != nil {
			return nil, err
		}
		return &DB{db, nil, nil}, nil
	}

	factory := func() (net.Conn, error) {
		connLock.Lock()
		defer connLock.Unlock()
		_db, err := sql.Open(driverName, dataSourceName)
		if err != nil {
			tempConn = nil
			connLock.Unlock()
			return nil, err
		}
		_db.SetMaxOpenConns(1)
		_db.SetMaxIdleConns(1)
		err = _db.Ping() //Force an actual connection to be created
		if err != nil {
			tempConn = nil
			return nil, err
		}

		db := &DB{_db, tempConn, nil}

		tempConn = nil

		return db, nil
	}

	ctx := appengine.NewContext(req[0])

	poolLock.RLock()
	if pool == nil {
		poolLock.RUnlock()
		poolLock.Lock()
		defer poolLock.Unlock()

		//Perform second test in case another goroutine creates the pool in between `poolLock.RUnlock()` and `poolLock.Lock()`
		if pool == nil {
			//Create a new pool
			p, err := connPool.NewChannelPool(1, maxOpenConns, factory)
			if err != nil {
				return nil, err
			}
			pool = p
		}

		//Use current pool
		conn, err := pool.Get()
		if err != nil {
			return nil, err
		}

		//Test the connection
		err = conn.(*connPool.PoolConn).Conn.(*DB).DB.Ping()
		if err != nil {
			if conn.(*connPool.PoolConn).Conn.(*DB).PoolConn != nil {
				conn.(*connPool.PoolConn).Conn.(*DB).PoolConn = nil
			}
			conn.(*connPool.PoolConn).Conn.(*DB).Close()
			conn.(*connPool.PoolConn).Conn.(*DB).Conn.Close()
			conn.(*connPool.PoolConn).MarkUnusable()
			return nil, err
		}

		// log.Println("Active Connections:", pool.Len())

		conn.(*connPool.PoolConn).Conn.(*DB).Conn.SetContext(ctx)

		//WARNING: RETAIN CYCLES
		conn.(*connPool.PoolConn).Conn.(*DB).PoolConn = conn.(*connPool.PoolConn)
		return conn.(*connPool.PoolConn).Conn.(*DB), nil
	} else {
		defer poolLock.RUnlock()

		//Use current pool
		conn, err := pool.Get()
		if err != nil {
			return nil, err
		}

		//Test the connection
		err = conn.(*connPool.PoolConn).Conn.(*DB).DB.Ping()
		if err != nil {
			if conn.(*connPool.PoolConn).Conn.(*DB).PoolConn != nil {
				conn.(*connPool.PoolConn).Conn.(*DB).PoolConn = nil
			}
			conn.(*connPool.PoolConn).Conn.(*DB).Close()
			conn.(*connPool.PoolConn).Conn.(*DB).Conn.Close()
			conn.(*connPool.PoolConn).MarkUnusable()
			return nil, err
		}

		// log.Println("Active Connections:", pool.Len())

		conn.(*connPool.PoolConn).Conn.(*DB).Conn.SetContext(ctx)

		//WARNING: RETAIN CYCLES
		conn.(*connPool.PoolConn).Conn.(*DB).PoolConn = conn.(*connPool.PoolConn)
		return conn.(*connPool.PoolConn).Conn.(*DB), nil
	}
}
