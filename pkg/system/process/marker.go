package process

import (
	"os"
	"strconv"

	"github.com/achandrapaul/digest/pkg/system"
)

func Mark(marker string) (func(), error) {
	if err := system.Write(marker, []byte(strconv.Itoa(os.Getpid()))); err != nil {
		return nil, err
	}
	return func() { _ = system.Remove(marker) }, nil
}

func Alive(marker string) bool {
	pid, ok := ReadInt(marker)
	return ok && PIDAlive(pid)
}
