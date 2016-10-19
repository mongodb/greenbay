package check

import (
	"testing"

	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/greenbay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fileExistsFactoryFactory(name string, require *require.Assertions) func() *fileExistance {
	factory, err := registry.GetJobFactory(name)
	require.NoError(err)
	return func() *fileExistance {
		check, ok := factory().(*fileExistance)
		require.True(ok)
		return check
	}
}

func TestFileExistsCheckImplementation(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	checkFactory := fileExistsFactoryFactory("file-exists", require)

	var check *fileExistance
	var output greenbay.CheckOutput

	// make sure it can find files that do exist
	check = checkFactory()
	check.FileName = "../makefile"
	check.Run()
	output = check.Output()

	assert.True(output.Completed)
	assert.True(output.Passed, output.Message)
	assert.NoError(check.Error())
	assert.Equal("", output.Error)

	// make sure it doesn't find files that don't exist
	check = checkFactory()
	check.FileName = "../makefile.DOES-NOT-EXIST"
	check.Run()
	output = check.Output()

	assert.True(output.Completed)
	assert.False(output.Passed, output.Message)
	assert.Error(check.Error())
	assert.NotEqual("", output.Error)
}

func TestFileDoesNotExistCheckImplementation(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	checkFactory := fileExistsFactoryFactory("file-does-not-exist", require)

	var check *fileExistance
	var output greenbay.CheckOutput

	// make sure that files that don't exist pass
	check = checkFactory()
	check.FileName = "../makefile.DOES-NOT-EXIST"
	check.Run()
	output = check.Output()

	assert.True(output.Completed)
	assert.True(output.Passed, output.Message)
	assert.NoError(check.Error())
	assert.Equal("", output.Error)

	// make sure files that exist fail
	check = checkFactory()
	check.FileName = "../makefile"
	check.Run()
	output = check.Output()

	assert.True(output.Completed)
	assert.False(output.Passed, output.Message)
	assert.Error(check.Error())
	assert.NotEqual("", output.Error)
}
