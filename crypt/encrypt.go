// Copyright © 2017 carlos derich <carlosderich@gmail.com>
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

package crypt

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"log"
	"path/filepath"

	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/scrypt"
)

// on Linux, Reader uses getrandom(2) if available, /dev/urandom otherwise.
// on OpenBSD, Reader uses getentropy(2).
// on other Unix-like systems, Reader reads from /dev/urandom.
// on Windows systems, Reader uses the CryptGenRandom API.
func random(size int) []byte {
	r := make([]byte, size)
	_, err := rand.Read(r)
	if err != nil {
		log.Fatal("error: ", err)
		return nil
	}

	return r
}

// reads the target file
func readFile(path string) ([]byte, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// creates the output encrypted file without extension
// save encrypted data in hex
// appends hex salt to output file
// appends hex file ext to output file
func createEncryptedFile(file string, salt, content []byte) (string, error) {

	extension := filepath.Ext(file)
	name := file[0 : len(file) - len(extension)]

	hexExt := hex.EncodeToString([]byte(extension))

	final := [][]byte{[]byte(hex.EncodeToString(content)), []byte(hex.EncodeToString(salt)), []byte(hexExt)}

	err := ioutil.WriteFile(name, bytes.Join(final, []byte("\n")), 0644)
	if err != nil {
		return "", err
	}

	return name, nil
}

func handleError(e error) (string, string, error) {
	log.Fatal(e)
	return "", "", e
}

// scrypt derives a 64 bytes key based from the passphrase if its provided
// or randomly generates a passphrase if its not provided.
// uses nacl box to encrypt the data using derived scrypt key
func Encrypt(path string, passphrase []byte) (string, string, error) {

	if len(passphrase) == 0 {
		log.Println("generating random passphrase ...")
		passphrase = []byte(hex.EncodeToString(random(16)))
		log.Println("file passphrase: ", string(passphrase))
	} else {
		log.Println("using user defined passphrase")
	}

	// generates a 32 bytes salt
	salt := random(32)

	var key [32]byte
	keyBytes, err := scrypt.Key(passphrase, salt, 16384, 8, 1, 32)
	if err != nil {
		return handleError(err)
	}

	// trick to set a fixed slice size for nacl
	copy(key[:], keyBytes)

	// must use a different nonce for each message you encrypt with the
	// same key. Since the nonce here is 192 bits long, a random value
	// provides a sufficiently small probability of repeats.
	var nonce [24]byte
	nonceBytes := random(24)
	copy(nonce[:], nonceBytes)

	data, err := readFile(path)
	if err != nil {
		return handleError(err)
	}

	// saves the nonce at the first 24 bytes of the encrypted output
	encrypted := secretbox.Seal(nonce[:], data, &nonce, &key)

	outputFilename, err := createEncryptedFile(path, salt, encrypted)
	if err != nil {
		return handleError(err)
	}

	return string(passphrase), outputFilename, nil
}
