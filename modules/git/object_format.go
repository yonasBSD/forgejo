// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

type ObjectHash struct {
	Hash       []byte
	FormatType ObjectFormatType
}

// String returns a string representation of the object hash.
func (oh ObjectHash) String() string {
	return hex.EncodeToString(oh.Hash)
}

const (
	sha1FullLength = 40
	// Sha1EmptySHA defines empty git object format for SHA1.
	Sha1EmptySHA = "0000000000000000000000000000000000000000"
	// Sha1EmptyTree is the object format for SHA1 of an empty tree
	Sha1EmptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

	sha256FullLength = 64
	// Sha256EmptySHA defines empty git object format for SHA256.
	Sha256EmptySHA = "0000000000000000000000000000000000000000000000000000000000000000"
	// Sha256EmptyTree is the object format for SHA256 of an empty tree
	Sha256EmptyTree = "6ef19b41225c5369f1c104d45d8d85efa9b057b53b14b4b9b939dd74decc5321"
)

var (
	sha1Pattern = regexp.MustCompile(`^[0-9a-f]{4,40}$`)

	sha256Pattern = regexp.MustCompile(`^[0-9a-f]{4,64}$`)
)

// ObjectFormatType defines the types of object format for this repository.
type ObjectFormatType uint8

// Kinds of ObjectFormat
const (
	invalidObjectFormat ObjectFormatType = iota
	SHA1ObjectFormat
	SHA256ObjectFormat
)

// ToObjectFormat converts a string to a ObjectFormatType
func ToObjectFormat(objectFormat string) (ObjectFormatType, error) {
	switch strings.ToLower(strings.TrimSpace(objectFormat)) {
	case "sha1":
		return SHA1ObjectFormat, nil
	case "sha256":
		return SHA256ObjectFormat, nil
	}

	return 0, fmt.Errorf("unknown object format: %q", objectFormat)
}

// String converts a ObjectFormatType to a string
func (t ObjectFormatType) String() string {
	switch t {
	case SHA1ObjectFormat:
		return "sha1"
	case SHA256ObjectFormat:
		return "sha256"
	}

	return "Unknown object format"
}

// HashLen returns the length of the hash algorithm that the object format uses.
func (t ObjectFormatType) HashLen() int {
	switch t {
	case SHA1ObjectFormat:
		return 20
	case SHA256ObjectFormat:
		return 32
	}

	return 0
}

// HashHexLen returns the hexified length of the hash algorithm that the object format uses.
func (t ObjectFormatType) HashHexLen() int {
	switch t {
	case SHA1ObjectFormat:
		return sha1FullLength
	case SHA256ObjectFormat:
		return sha256FullLength
	}

	return 0
}

// IsValidCommitHash returns if the commitID is valid.
func (t ObjectFormatType) IsValidCommitHash(commitID string) bool {
	switch t {
	case SHA1ObjectFormat:
		return len(commitID) == sha1FullLength && sha1Pattern.MatchString(commitID)
	case SHA256ObjectFormat:
		return len(commitID) == sha256FullLength && sha256Pattern.MatchString(commitID)
	}

	return false
}

// NewObjectHashFromString creates a new object hash from an ID string.
func (t ObjectFormatType) NewObjectHashFromString(commitID string) (ObjectHash, error) {
	commitID = strings.TrimSpace(commitID)

	if len(commitID) != t.HashHexLen() {
		return ObjectHash{}, fmt.Errorf("Length must be %d: %s", t.HashHexLen(), commitID)
	}

	b, err := hex.DecodeString(commitID)
	if err != nil {
		return ObjectHash{}, err
	}

	return ObjectHash{
		Hash:       b,
		FormatType: t,
	}, nil
}
