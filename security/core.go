/*
 * @file            security/core.go
 * @description
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2024-03-02 19:14:18
 * @lastModified    2024-06-27 17:24:18
 * Copyright ©Théo Mougnibas All rights reserved
 */

package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"
)

func CheckRequestIntegrity(req []byte) bool {

	sent_hash := req[len(req)-1-32:]

	gen_hash := sha256.Sum256(req[0 : len(req)-32])

	return ([32]byte)(sent_hash) == gen_hash
}

func DecryptRequest(req []byte, device_key string) ([]byte, error) {
	// Create a new AES cipher block from the key
	block, err := aes.NewCipher([]byte(device_key))
	if err != nil {
		return nil, err
	}

	// Create a new GCM instance
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Extract the nonce from the ciphertext
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := req[:nonceSize], req[nonceSize:]

	// Decrypt the ciphertext using AES-GCM
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	// Return the plaintext
	return plaintext, nil
}

func EncryptRequest(req []byte, device_key string) ([]byte, error) {
	// Create a new AES cipher block from the key
	block, err := aes.NewCipher([]byte(device_key))
	if err != nil {
		return nil, err
	}

	// Create a new GCM instance
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate a random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt the plaintext using AES-GCM
	ciphertext := gcm.Seal(nil, nonce, req, nil)

	// Return the ciphertext and nonce
	return append(nonce, ciphertext...), nil
}

/*
func main() {
    // key should be 16, 24 or 32 bytes to select AES-128, AES-192 or AES-256
    key := []byte("0123456789012345")
    plaintext := []byte("This is a secret message.")
    fmt.Println("plaintext : ", plaintext)
    ciphertext, err := encrypt(plaintext, key)
    if err != nil {
        fmt.Println(err)
        return
    }
    fmt.Println("ciphertext : ", ciphertext)
    decrypted, err := decrypt(ciphertext, key)
    if err != nil {
        fmt.Println(err)
        return
    }
    fmt.Println("decrypted : ", decrypted)
}
*/
