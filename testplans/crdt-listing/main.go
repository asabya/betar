package main

import (
	"github.com/asabya/betar/testplans/crdt-listing/testcase"
	"github.com/testground/sdk-go/run"
)

var testcases = map[string]interface{}{
	"list-10-agents": run.InitializedTestCaseFn(testcase.List10Agents),
}

func main() {
	run.InvokeMap(testcases)
}
