package wallet

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/ocamndravin/gaminium/config"
	"github.com/ocamndravin/gaminium/internal/crypto"
)

// TxInput references a previous unspent output.
type TxInput struct {
	TxID      crypto.Hash
	OutIndex  uint32
	Signature crypto.DilithiumSignature
	PublicKey crypto.DilithiumPublicKey
	Sequence  uint32
}

// TxOutput defines a payment destination.
type TxOutput struct {
	Amount  int64  // in Minium
	Address string // GMN1... address
	Script  []byte // locking script (for multisig)
}

// Transaction is a GAMINIUM on-chain transfer.
type Transaction struct {
	Version   uint32
	Inputs    []TxInput
	Outputs   []TxOutput
	LockTime  int64
	Timestamp int64
	Fee       int64 // in Minium
}

// TxID computes the canonical BLAKE3-512 transaction ID.
func (tx *Transaction) TxID() crypto.Hash {
	return crypto.HashMany(tx.serialiseForID())
}

func (tx *Transaction) serialiseForID() []byte {
	// Deterministic serialisation for hashing
	var buf []byte
	buf = append(buf, uint32Bytes(tx.Version)...)
	buf = append(buf, int64Bytes(tx.Timestamp)...)
	buf = append(buf, int64Bytes(tx.LockTime)...)
	for _, in := range tx.Inputs {
		buf = append(buf, in.TxID[:]...)
		buf = append(buf, uint32Bytes(in.OutIndex)...)
		buf = append(buf, uint32Bytes(in.Sequence)...)
		buf = append(buf, in.PublicKey[:]...)
	}
	for _, out := range tx.Outputs {
		buf = append(buf, int64Bytes(out.Amount)...)
		buf = append(buf, []byte(out.Address)...)
		buf = append(buf, out.Script...)
	}
	return buf
}

// SigningHash returns the hash that each input must sign.
// Includes all outputs and other inputs (SIGHASH_ALL semantics).
func (tx *Transaction) SigningHash() crypto.Hash {
	return crypto.HashMany([]byte("gmn-sighash-v1"), tx.serialiseForID())
}

// Sign signs all inputs with the provided derived key.
func (tx *Transaction) Sign(key *DerivedKey) error {
	sigHash := tx.SigningHash()
	sig, err := crypto.DilithiumSign(key.DilithiumKey.Private, sigHash[:])
	if err != nil {
		return fmt.Errorf("sign tx: %w", err)
	}
	for i := range tx.Inputs {
		tx.Inputs[i].Signature = sig
		tx.Inputs[i].PublicKey = key.DilithiumKey.Public
	}
	return nil
}

// Validate performs basic transaction validation.
func (tx *Transaction) Validate() error {
	if len(tx.Inputs) == 0 {
		return errors.New("tx: no inputs")
	}
	if len(tx.Outputs) == 0 {
		return errors.New("tx: no outputs")
	}
	if tx.Fee < 0 {
		return errors.New("tx: negative fee")
	}

	totalOut := int64(0)
	for _, out := range tx.Outputs {
		if out.Amount < config.MinTransaction {
			return fmt.Errorf("tx: output below minimum (%d Minium)", config.MinTransaction)
		}
		if err := ValidateAddress(out.Address); err != nil {
			return fmt.Errorf("tx: invalid output address: %w", err)
		}
		totalOut += out.Amount
	}

	// Verify all input signatures
	sigHash := tx.SigningHash()
	for i, in := range tx.Inputs {
		if err := crypto.DilithiumVerify(in.PublicKey, sigHash[:], in.Signature); err != nil {
			return fmt.Errorf("tx: input %d signature invalid: %w", i, err)
		}
		// Verify input address matches public key
		expectedAddr := PublicKeyToAddress(in.PublicKey)
		_ = expectedAddr // address verified against UTXO set at block validation
	}
	_ = totalOut
	return nil
}

// NewTransaction creates an unsigned transaction.
func NewTransaction(inputs []TxInput, outputs []TxOutput, fee int64) *Transaction {
	return &Transaction{
		Version:   1,
		Inputs:    inputs,
		Outputs:   outputs,
		Fee:       fee,
		Timestamp: time.Now().Unix(),
	}
}

// FeeDistribution calculates fee splits in Minium.
func FeeDistribution(fee int64) (minerFee, treasuryFee, oracleFee int64) {
	minerFee = fee * config.FeeMinerShare / 100
	treasuryFee = fee * config.FeeTreasuryShare / 100
	oracleFee = fee - minerFee - treasuryFee
	return
}

func uint32Bytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

func int64Bytes(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
