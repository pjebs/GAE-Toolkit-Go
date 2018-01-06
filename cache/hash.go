package cache

import (
	"fmt"
	"hash/crc32"
	"runtime"
)

type Key string

// CreateKey will generate a key based on the input arguments.
// When prefix is true, the caller's name will be used to prefix the key in an attempt to make it unique.
// The args can also be separated using sep. visual does not do anything. It is used at code level to easily
// see how the key should look
func CreateKey(prefix bool, sep string, visual string, args ...interface{}) Key {
	var output string

	if prefix {
		pc, file, line, ok := runtime.Caller(1)
		if !ok {
			return Key(fmt.Sprint(args...))
		}
		details := runtime.FuncForPC(pc)
		output = fmt.Sprintf("%s_%s_%d_", details.Name(), file, line)
	}

	if sep == "" {
		output = output + fmt.Sprint(args...)
	} else {
		for i, v := range args {
			if i != 0 {
				output = output + sep
			}
			output = output + fmt.Sprint(v)
		}
	}

	return Key(output)
}

// Hash returns a crc32 version of key
func (k Key) Hash() Key {
	return Key(fmt.Sprintf("%08x\n", crc32.ChecksumIEEE([]byte(k))))
}

// String returns a string version of the key
func (k Key) String() string {
	return string(k)
}
