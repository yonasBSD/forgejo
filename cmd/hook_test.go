// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cmd

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPktLine(t *testing.T) {
	// test read
	s := strings.NewReader("0000")
	r := bufio.NewReader(s)
	result, err := readPktLine(r)
	assert.NoError(t, err)
	assert.Equal(t, pktLineTypeFlush, result.Type)

	s = strings.NewReader("0006a\n")
	r = bufio.NewReader(s)
	result, err = readPktLine(r)
	assert.NoError(t, err)
	assert.Equal(t, pktLineTypeData, result.Type)
	assert.Equal(t, "a\n", result.Data)

	// test write
	w := bytes.NewBuffer([]byte{})
	err = writePktLine(w, pktLineTypeFlush, nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("0000"), w.Bytes())

	w.Reset()
	err = writePktLine(w, pktLineTypeData, []byte("a\nb"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("0007a\nb"), w.Bytes())
}
