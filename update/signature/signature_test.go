// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package signature

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/flatcar-linux/mantle/update/metadata"
)

const (
	// must match the developer key in signature.go
	developerKeyBits  = 2048
	developerKeyBytes = developerKeyBits / 8
	// protobuf encoding and Version field take up 8 bytes
	developerSigBytes = developerKeyBytes + 8

	// base64 encoded sha256 hash of nothing
	testHashStr = `47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=`

	// signature of that hash using the developer key,
	// generated by openssl and base64 encoded.
	testSigStr = `
ouJu+zJ3fHr0m4+O7GxNIa3vuFS1c453OSoA9IMjkZSrAw7yZfnpROQjwn3tQHZB
X5uBcmv1y+UEgyWDA5pnpL/gpw1NSB04H/J7atzO4s42ilkmhH6smLLV9OsQQ5/W
tmxkVDxFH7GgGyueuxqlTFL4BsBA89/PXZ1gbcn/4fvQc9quq8aAynBSJcAoZca2
Eby/N3mSO3sPZLAOzbfwZ23Ph6gJ9QJ6VnFh2xe6NGSwXhgjyHGKiYcajdzF2Iqf
7ajqxLe8xVnC2KmRKcY25qs1Atq6e66Cs5PdN7uzNhhLrqBeCoTQjAUnjOA90wt8
1rgCGKxZBVYqZPQsuBdSaw==`
)

var (
	testHash []byte
	testSig  []byte
)

func init() {
	var err error
	testHash, err = base64.StdEncoding.DecodeString(testHashStr)
	if err != nil {
		panic(err)
	}
	if len(testHash) != signatureHash.Size() {
		panic("invalid test hash")
	}
	testSig, err = base64.StdEncoding.DecodeString(testSigStr)
	if err != nil {
		panic(err)
	}
	if len(testSig) != developerKeyBytes {
		panic("invalid test sig")
	}
}

func TestKeySize(t *testing.T) {
	n, err := keySize()
	if err != nil {
		t.Fatal(err)
	}

	if n != developerKeyBytes {
		t.Errorf("key size is %d not %d", n, developerKeyBytes)
	}
}

func TestSignaturesSize(t *testing.T) {
	n, err := SignaturesSize()
	if err != nil {
		t.Fatal(err)
	}

	if n != developerSigBytes {
		t.Errorf("sig size is %d not %d", n, developerSigBytes)
	}
}

func TestSign(t *testing.T) {
	sigs, err := Sign(testHash)
	if err != nil {
		t.Fatal(err)
	}

	if len(sigs.Signatures) != 1 {
		t.Fatalf("Unexpected: %s", sigs)
	}

	if *sigs.Signatures[0].Version != signatureVersion {
		t.Errorf("Unexpected version %d", *sigs.Signatures[0].Version)
	}

	if !bytes.Equal(sigs.Signatures[0].Data, testSig) {
		t.Errorf("Unexpected signature %q", sigs.Signatures[0].Data)
	}
}

func TestVerifySignature(t *testing.T) {
	sigs := &metadata.Signatures{
		Signatures: []*metadata.Signatures_Signature{
			&metadata.Signatures_Signature{
				Version: proto.Uint32(signatureVersion),
				Data:    testSig,
			},
		},
	}

	if err := VerifySignature(testHash, sigs); err != nil {
		t.Error(err)
	}
}
