package zsub

import (
	"fmt"
	"testing"
)

func TestName(t *testing.T) {
	sub := ZSub{
		topics: map[string]*ZTopic{},
	}

	sub.subscribe(&ZConn{
		groupid: "a",
	}, "ab")

	sub.subscribe(&ZConn{
		groupid: "b",
	}, "ab")

	// -----------------

	sub.subscribe(&ZConn{
		groupid: "b",
	}, "abx")

	conn := ZConn{
		groupid: "a",
	}

	sub.subscribe(&conn, "abx")

	sub.unsubscribe(&conn, "abx")

	fmt.Println(1)
}
