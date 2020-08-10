package dump

import (
	"os"

	"github.com/fsamin/go-dump"
)

func Print(data ...interface{}) {
	dumper := dump.NewEncoder(os.Stdout)
	dumper.ExtraFields.Len = true
	dumper.ExtraFields.Type = true
	dumper.Fdump(data)
}
