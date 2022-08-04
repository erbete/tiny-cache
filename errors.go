package tinycache

import (
	"fmt"
)

type ErrorKeyNotExist struct {
	Key string
}

func (e *ErrorKeyNotExist) Error() string {
	return fmt.Sprintf("key \"%s\" does not exist", e.Key)
}
