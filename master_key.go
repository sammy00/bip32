package bip32

import (
	"crypto/hmac"
	"crypto/sha512"
	"io"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
)

func GenerateMasterKey(rand io.Reader, keyID Magic,
	strength ...int) (*ExtendedKey, error) {
	seedLen := RecommendedSeedLen
	if len(strength) > 0 && strength[0] >= MinSeedBytes &&
		strength[0] <= MaxSeedBytes {
		seedLen = strength[0]
	}

	seed := make([]byte, seedLen)
	//if n, err := rand.Read(seed); nil != err || n != seedLen {
	//	return nil, ErrNoEnoughEntropy
	//}
	if _, err := io.ReadFull(rand, seed); nil != err {
		return nil, ErrNoEnoughEntropy
	}

	// I = HMAC-SHA512(Key = "Bitcoin seed", Data = S)
	hmac512 := hmac.New(sha512.New, masterKey)
	hmac512.Write(seed)
	I := hmac512.Sum(nil)

	secretKey, chainCode := I[:len(I)/2], I[len(I)/2:]
	// Ensure the key in usable.
	if x := new(big.Int).SetBytes(secretKey); 0 == x.Sign() ||
		x.Cmp(btcec.S256().N) >= 0 {
		return nil, ErrUnusableSeed
	}

	// fingerprint of parent
	parentFP := []byte{0x00, 0x00, 0x00, 0x00}

	return NewExtendedKey(keyID[:], secretKey, chainCode,
		parentFP, 0, 0, true), nil
}

// NewMaster creates a new master node for use in creating a hierarchical
// deterministic key chain.  The seed must be between 128 and 512 bits and
// should be generated by a cryptographically secure random generation source.
//
// NOTE: There is an extremely small chance (< 1 in 2^127) the provided seed
// will derive to an unusable secret key.  The ErrUnusable error will be
// returned if this should occur, so the caller must check for it and generate a
// new seed accordingly.
func NewMaster(seed []byte, net *chaincfg.Params) (*ExtendedKey, error) {
	// Per [BIP32], the seed must be in range [MinSeedBytes, MaxSeedBytes].
	if len(seed) < MinSeedBytes || len(seed) > MaxSeedBytes {
		return nil, ErrInvalidSeedLen
	}

	// First take the HMAC-SHA512 of the master key and the seed data:
	//   I = HMAC-SHA512(Key = "Bitcoin seed", Data = S)
	hmac512 := hmac.New(sha512.New, masterKey)
	hmac512.Write(seed)
	lr := hmac512.Sum(nil)

	// Split "I" into two 32-byte sequences Il and Ir where:
	//   Il = master secret key
	//   Ir = master chain code
	secretKey := lr[:len(lr)/2]
	chainCode := lr[len(lr)/2:]

	// Ensure the key in usable.
	secretKeyNum := new(big.Int).SetBytes(secretKey)
	if secretKeyNum.Cmp(btcec.S256().N) >= 0 || secretKeyNum.Sign() == 0 {
		return nil, ErrUnusableSeed
	}

	parentFP := []byte{0x00, 0x00, 0x00, 0x00}
	return NewExtendedKey(net.HDPrivateKeyID[:], secretKey, chainCode,
		parentFP, 0, 0, true), nil
}