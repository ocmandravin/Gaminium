package crypto

import (
	"encoding/hex"

	"lukechampine.com/blake3"
)

const HashSize = 64 // BLAKE3-512 (64 bytes)

type Hash [HashSize]byte

// HashBytes returns the BLAKE3-512 hash of data.
func HashBytes(data []byte) Hash {
	h := blake3.New(HashSize, nil)
	h.Write(data)
	var out Hash
	h.Sum(out[:0])
	return out
}

// HashMany hashes multiple byte slices concatenated together.
func HashMany(parts ...[]byte) Hash {
	h := blake3.New(HashSize, nil)
	for _, p := range parts {
		h.Write(p)
	}
	var out Hash
	h.Sum(out[:0])
	return out
}

// MerkleRoot computes the BLAKE3-512 merkle root of a list of hashes.
func MerkleRoot(hashes []Hash) Hash {
	if len(hashes) == 0 {
		return Hash{}
	}
	current := make([]Hash, len(hashes))
	copy(current, hashes)
	for len(current) > 1 {
		if len(current)%2 != 0 {
			current = append(current, current[len(current)-1])
		}
		next := make([]Hash, len(current)/2)
		for i := 0; i < len(current); i += 2 {
			next[i/2] = HashMany(current[i][:], current[i+1][:])
		}
		current = next
	}
	return current[0]
}

func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

func (h Hash) Bytes() []byte {
	b := make([]byte, HashSize)
	copy(b, h[:])
	return b
}

func HashFromBytes(b []byte) Hash {
	var h Hash
	copy(h[:], b)
	return h
}

func HashFromHex(s string) (Hash, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return Hash{}, err
	}
	return HashFromBytes(b), nil
}
