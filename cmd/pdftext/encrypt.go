package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rc4"
	"encoding/binary"
	"fmt"
)

var passwordPadding []byte = []byte("\x28\xBF\x4E\x5E\x4E\x75\x8A\x41\x64\x00\x4E\x56\xFF\xFA\x01\x08\x2E\x2E\x00\xB6\xD0\x68\x3E\x80\x2F\x0C\xA9\xFE\x64\x53\x69\x7A")

var BadPassword error = fmt.Errorf("bad password")

type pdfEncrypt struct {
	isEncrypted bool
	key         []byte
	id, O, U    []byte
	R, V        int
	n           int
	owner       bool
}

func (r *pdfReader) readEncrypt(password []byte) error {
	val, ok := r.trailer["Encrypt"]
	if !ok {
		return nil
	}
	encrypt, err := r.GetDict(val)
	if err != nil {
		return err
	}
	filter, _ := encrypt["Filter"].(pdfName)
	if filter != "Standard" {
		return fmt.Errorf("unsupported encryption")
	}

	val, ok = r.trailer["ID"]
	if !ok {
		return fmt.Errorf("missing document ID")
	}
	ids, _ := val.(pdfArray)
	if len(ids) != 2 {
		return fmt.Errorf("missing document ID")
	}
	id, _ := ids[0].([]byte)

	V, _ := encrypt["V"].(int)
	if V != 1 && V != 2 && V != 4 {
		return fmt.Errorf("bad encryption algorithm")
	} else if V == 4 {
		return fmt.Errorf("unsupported encryption algorithm")
	}
	length, ok := encrypt["Length"].(int)
	if !ok {
		length = 40
	} else if length%8 != 0 || length < 40 || 128 < length {
		return fmt.Errorf("bad encryption length")
	}

	R, ok := encrypt["R"].(int)
	if !ok {
		return fmt.Errorf("bad encryption dictionary")
	} else if V < 2 && R != 2 || (V == 2 || V == 3) && R != 3 || V == 4 && R != 4 {
		return fmt.Errorf("bad encryption revision")
	}
	O, ok := encrypt["O"].([]byte)
	if !ok || len(O) != 32 {
		return fmt.Errorf("bad encryption dictionary")
	}
	U, ok := encrypt["U"].([]byte)
	if !ok || len(U) != 32 {
		return fmt.Errorf("bad encryption dictionary")
	}
	P, ok := encrypt["P"].(int)
	if !ok {
		return fmt.Errorf("bad encryption dictionary")
	}
	encryptMetadata := true
	if val, ok := encrypt["EncryptMetadata"]; ok {
		encryptMetadata = val.(bool)
	}

	// pad or clip password to 32 bytes, password may be empty (default password)
	if 32 <= len(password) {
		password = password[:32]
	} else {
		password = append(password, passwordPadding[:32-len(password)]...)
	}

	n := 5
	if 3 <= R {
		n = length / 8
	}
	if md5.Size < n {
		return fmt.Errorf("bad encryption length")
	}

	// compute encryption key
	hash := md5.New()
	hash.Write([]byte(password))
	hash.Write(O)
	binary.Write(hash, binary.LittleEndian, uint32(P))
	hash.Write(id)
	if 4 <= R && !encryptMetadata {
		hash.Write([]byte("\xFF\xFF\xFF\xFF"))
	}

	if 3 <= R {
		for i := 0; i < 50; i++ {
			sum := hash.Sum(nil)
			hash.Reset()
			hash.Write(sum[:n])
		}
	}
	key := hash.Sum(nil)[:n]

	r.encrypt.isEncrypted = true
	r.encrypt.key = key
	r.encrypt.id = id
	r.encrypt.O = O
	r.encrypt.U = U
	r.encrypt.R = R
	r.encrypt.V = V
	r.encrypt.n = n

	// authenticate
	isOwner := r.encrypt.authenticateOwner(password)
	if !isOwner && !r.encrypt.authenticateUser(password) {
		return BadPassword
	}
	r.encrypt.owner = isOwner
	return nil
}

func (encrypt pdfEncrypt) authenticateUser(password []byte) bool {
	cipher, _ := rc4.NewCipher(encrypt.key)
	if encrypt.R == 2 {
		dst := make([]byte, 32)
		cipher.XORKeyStream(dst, passwordPadding)
		if !bytes.Equal(dst, encrypt.U) {
			return false
		}
	} else {
		// 3 <= encrypt.R
		hash := md5.New()
		hash.Write(passwordPadding)
		hash.Write(encrypt.id)

		dst := make([]byte, 16)
		cipher.XORKeyStream(dst, hash.Sum(nil))

		xorKey := make([]byte, len(encrypt.key))
		for i := 1; i < 20; i++ {
			for j := 0; j < len(encrypt.key); j++ {
				xorKey[j] = encrypt.key[j] ^ byte(i)
			}
			cipher, _ = rc4.NewCipher(xorKey)
			cipher.XORKeyStream(dst, dst)
		}
		if !bytes.Equal(dst, encrypt.U[:16]) {
			return false
		}
	}
	return true
}

func (encrypt pdfEncrypt) authenticateOwner(password []byte) bool {
	hash := md5.New()
	hash.Write(password)
	for i := 0; i < 50; i++ {
		sum := hash.Sum(nil)
		hash.Reset()
		hash.Write(sum)
	}
	key := hash.Sum(nil)[:encrypt.n]

	dst := make([]byte, 32)
	cipher, _ := rc4.NewCipher(key)
	if encrypt.R == 2 {
		cipher.XORKeyStream(dst, encrypt.O)
	} else {
		// 3 <= R
		xorKey := make([]byte, len(key))
		for i := 19; 0 <= i; i-- {
			for j := 0; j < len(key); j++ {
				xorKey[j] = key[j] ^ byte(i)
			}
			cipher, _ = rc4.NewCipher(xorKey)
			cipher.XORKeyStream(dst, dst)
		}
	}
	return encrypt.authenticateUser(dst)
}

func (encrypt pdfEncrypt) Encrypt(ref pdfRef, data []byte) []byte {
	key := append(encrypt.key[:encrypt.n], []byte("\x00\x00\x00\x00\x00")...)
	key[len(encrypt.key)+0] = byte(ref[0])
	key[len(encrypt.key)+1] = byte(ref[0] >> 8)
	key[len(encrypt.key)+2] = byte(ref[0] >> 16)
	key[len(encrypt.key)+3] = byte(ref[1])
	key[len(encrypt.key)+4] = byte(ref[1] >> 8)
	// TODO: add 'sAlT' (0x73416C54) to key when using AES algorithm

	n := encrypt.n + 5
	if 16 < n {
		n = 16
	}
	hash := md5.New()
	hash.Write(key)
	key = hash.Sum(nil)[:n]

	dst := make([]byte, len(data))
	cipher, _ := rc4.NewCipher(key)
	cipher.XORKeyStream(dst, data)
	return dst
}

func (encrypt pdfEncrypt) Decrypt(ref pdfRef, data []byte) []byte {
	return encrypt.Encrypt(ref, data)
}
