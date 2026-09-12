package data

import (
	"os"
	"testing"

	"github.com/attic-labs/testify/assert"
)

func TestOpenDataFile(t *testing.T) {
	t.Log("file path:", os.TempDir())
	df, err := OpenDataFile(os.TempDir(), 0)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, df)
	assert.Nil(t, err)
	df2, err := OpenDataFile(os.TempDir(), 111)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, df2)
	df3, err := OpenDataFile(os.TempDir(), 111)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, df3)
	df.Close()
	df2.Close()
}

func TestDataFileWrite(t *testing.T) {
	t.Log("file path:", os.TempDir())
	df, err := OpenDataFile(os.TempDir(), 0)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, df)
	assert.Nil(t, err)

	data := []byte("Hello, World!")
	n, err := df.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, len(data), n)

	data = []byte("\nHello, World!2")
	n, err = df.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, len(data), n)
	err = df.Close()
	assert.Nil(t, err)
}
