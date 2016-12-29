package output

import (
	"encoding/json"
	"fmt"

	"github.com/mongodb/greenbay"
	"github.com/tychoish/grip/message"
)

type jsonOutput struct {
	output       greenbay.CheckOutput
	message.Base `bson:"-" json:"-" yaml:"-"`
}

func (o *jsonOutput) Raw() interface{} { return o.output }
func (o *jsonOutput) Loggable() bool   { return true }
func (o *jsonOutput) Resolve() string {
	out, err := json.Marshal(o)
	if err != nil {
		return fmt.Sprintf("error processing result for %s (%+v): %+v",
			o.output.Name, err, o.output)
	}

	return string(out)
}
