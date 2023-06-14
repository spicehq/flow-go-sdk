/*
 * Flow Go SDK
 *
 * Copyright 2019 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package flow

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
)

// A Transaction is a full transaction object containing a payload and signatures.
type Transaction struct {
	// Script is the UTF-8 encoded Cadence source code that defines the execution logic for this transaction.
	Script []byte

	// Arguments is a list of Cadence values passed into this transaction.
	//
	// Each argument is encoded as JSON-CDC bytes.
	Arguments [][]byte

	// ReferenceBlockID is a reference to the block used to calculate the expiry of this transaction.
	//
	// A transaction is considered expired if it is submitted to Flow after refBlock + N, where N
	// is a constant defined by the network.
	//
	// For example, if a transaction references a block with height of X and the network limit is 10,
	// a block with height X+10 is the last block that is allowed to include this transaction.
	ReferenceBlockID Identifier

	// GasLimit is the maximum number of computational units that can be used to execute this transaction.
	GasLimit uint64

	// ProposalKey is the account key used to propose this transaction.
	//
	// A proposal key references a specific key on an account, along with an up-to-date
	// sequence number for that key. This sequence number is used to prevent replay attacks.
	//
	// You can find more information about sequence numbers here: https://docs.onflow.org/concepts/transaction-signing/#sequence-numbers
	ProposalKey ProposalKey

	// Payer is the account that pays the fee for this transaction.
	//
	// You can find more information about the payer role here: https://docs.onflow.org/concepts/transaction-signing/#signer-roles
	Payer Address

	// Authorizers is a list of the accounts that are authorizing this transaction to
	// mutate to their on-chain account state.
	//
	// You can find more information about the authorizer role here: https://docs.onflow.org/concepts/transaction-signing/#signer-roles
	Authorizers []Address

	// PayloadSignatures is a list of signatures generated by the proposer and authorizer roles.
	//
	// A payload signature is generated over the inner portion of the transaction (TransactionDomainTag + payload).
	//
	// You can find more information about transaction signatures here: https://docs.onflow.org/concepts/transaction-signing/#anatomy-of-a-transaction
	PayloadSignatures []TransactionSignature

	// EnvelopeSignatures is a list of signatures generated by the payer role.
	//
	// An envelope signature is generated over the outer portion of the transaction (TransactionDomainTag + payload + payloadSignatures).
	//
	// You can find more information about transaction signatures here: https://docs.onflow.org/concepts/transaction-signing/#anatomy-of-a-transaction
	EnvelopeSignatures []TransactionSignature
}

type payloadCanonicalForm struct {
	Script                    []byte
	Arguments                 [][]byte
	ReferenceBlockID          []byte
	GasLimit                  uint64
	ProposalKeyAddress        []byte
	ProposalKeyIndex          uint64
	ProposalKeySequenceNumber uint64
	Payer                     []byte
	Authorizers               [][]byte
}

type envelopeCanonicalForm struct {
	Payload           payloadCanonicalForm
	PayloadSignatures []transactionSignatureCanonicalForm
}

type transactionCanonicalForm struct {
	Payload            payloadCanonicalForm
	PayloadSignatures  []transactionSignatureCanonicalForm
	EnvelopeSignatures []transactionSignatureCanonicalForm
}

// DefaultTransactionGasLimit should be high enough for small transactions
const DefaultTransactionGasLimit = 9999

// NewTransaction initializes and returns an empty transaction.
func NewTransaction() *Transaction {
	return &Transaction{
		GasLimit: DefaultTransactionGasLimit,
	}
}

// SetScript sets the Cadence script for this transaction.
//
// The script is the UTF-8 encoded Cadence source code.
func (t *Transaction) SetScript(script []byte) *Transaction {
	t.Script = script
	return t
}

// AddArgument adds a Cadence argument to this transaction.
func (t *Transaction) AddArgument(arg cadence.Value) error {
	encodedArg, err := jsoncdc.Encode(arg)
	if err != nil {
		return fmt.Errorf("failed to encode argument: %w", err)
	}

	t.Arguments = append(t.Arguments, encodedArg)
	return nil
}

// AddRawArgument adds a raw JSON-CDC encoded argument to this transaction.
func (t *Transaction) AddRawArgument(arg []byte) *Transaction {
	t.Arguments = append(t.Arguments, arg)
	return t
}

// Argument returns the decoded argument at the given index.
func (t *Transaction) Argument(i int, options ...jsoncdc.Option) (cadence.Value, error) {
	if i < 0 {
		return nil, fmt.Errorf("argument index must be positive")
	}

	if i >= len(t.Arguments) {
		return nil, fmt.Errorf("no argument at index %d", i)
	}

	encodedArg := t.Arguments[i]

	arg, err := jsoncdc.Decode(nil, encodedArg, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to decode argument at index %d: %w", i, err)
	}

	return arg, nil
}

// SetReferenceBlockID sets the reference block ID for this transaction.
//
// A transaction is considered expired if it is submitted to Flow after refBlock + N, where N
// is a constant defined by the network.
//
// For example, if a transaction references a block with height of X and the network limit is 10,
// a block with height X+10 is the last block that is allowed to include this transaction.
func (t *Transaction) SetReferenceBlockID(blockID Identifier) *Transaction {
	t.ReferenceBlockID = blockID
	return t
}

// SetGasLimit sets the gas limit for this transaction.
func (t *Transaction) SetGasLimit(limit uint64) *Transaction {
	t.GasLimit = limit
	return t
}

// SetProposalKey sets the proposal key and sequence number for this transaction.
//
// The first two arguments specify the account key to be used, and the last argument is the sequence
// number being declared.
func (t *Transaction) SetProposalKey(address Address, keyIndex int, sequenceNum uint64) *Transaction {
	proposalKey := ProposalKey{
		Address:        address,
		KeyIndex:       keyIndex,
		SequenceNumber: sequenceNum,
	}
	t.ProposalKey = proposalKey
	t.refreshSignerIndex()
	return t
}

// SetPayer sets the payer account for this transaction.
func (t *Transaction) SetPayer(address Address) *Transaction {
	t.Payer = address
	t.refreshSignerIndex()
	return t
}

// AddAuthorizer adds an authorizer account to this transaction.
func (t *Transaction) AddAuthorizer(address Address) *Transaction {
	t.Authorizers = append(t.Authorizers, address)
	t.refreshSignerIndex()
	return t
}

// signerList returns a list of unique accounts required to sign this transaction.
//
// The list is returned in the following order:
// 1. PROPOSER
// 2. PAYER
// 2. AUTHORIZERS (in insertion order)
//
// The only exception to the above ordering is for deduplication; if the same account
// is used in multiple signing roles, only the first occurrence is included in the list.
func (t *Transaction) signerList() []Address {
	signers := make([]Address, 0)
	seen := make(map[Address]struct{})

	var addSigner = func(address Address) {
		_, ok := seen[address]
		if ok {
			return
		}

		signers = append(signers, address)
		seen[address] = struct{}{}
	}

	if t.ProposalKey.Address != EmptyAddress {
		addSigner(t.ProposalKey.Address)
	}

	if t.Payer != EmptyAddress {
		addSigner(t.Payer)
	}

	for _, authorizer := range t.Authorizers {
		addSigner(authorizer)
	}

	return signers
}

// signerMap returns a mapping from address to signer index.
func (t *Transaction) signerMap() map[Address]int {
	signers := make(map[Address]int)

	for i, signer := range t.signerList() {
		signers[signer] = i
	}

	return signers
}

func (t *Transaction) refreshSignerIndex() {
	signerMap := t.signerMap()
	for i, sig := range t.PayloadSignatures {
		signerIndex, signerExists := signerMap[sig.Address]
		if !signerExists {
			signerIndex = -1
		}
		t.PayloadSignatures[i].SignerIndex = signerIndex
	}
	for i, sig := range t.EnvelopeSignatures {
		signerIndex, signerExists := signerMap[sig.Address]
		if !signerExists {
			signerIndex = -1
		}
		t.EnvelopeSignatures[i].SignerIndex = signerIndex
	}
}

// AddPayloadSignature adds a payload signature to the transaction for the given address and key index.
func (t *Transaction) AddPayloadSignature(address Address, keyIndex int, sig []byte) *Transaction {
	s := t.createSignature(address, keyIndex, sig)

	t.PayloadSignatures = append(t.PayloadSignatures, s)
	sort.Slice(t.PayloadSignatures, compareSignatures(t.PayloadSignatures))
	t.refreshSignerIndex()
	return t
}

// AddEnvelopeSignature adds an envelope signature to the transaction for the given address and key index.
func (t *Transaction) AddEnvelopeSignature(address Address, keyIndex int, sig []byte) *Transaction {
	s := t.createSignature(address, keyIndex, sig)

	t.EnvelopeSignatures = append(t.EnvelopeSignatures, s)
	sort.Slice(t.EnvelopeSignatures, compareSignatures(t.EnvelopeSignatures))
	t.refreshSignerIndex()
	return t
}

func (t *Transaction) createSignature(address Address, keyIndex int, sig []byte) TransactionSignature {
	signerIndex, signerExists := t.signerMap()[address]
	if !signerExists {
		signerIndex = -1
	}

	return TransactionSignature{
		Address:     address,
		SignerIndex: signerIndex,
		KeyIndex:    keyIndex,
		Signature:   sig,
	}
}

func (t *Transaction) PayloadMessage() []byte {
	temp := t.payloadCanonicalForm()
	return mustRLPEncode(&temp)
}

func (t *Transaction) payloadCanonicalForm() payloadCanonicalForm {
	authorizers := make([][]byte, len(t.Authorizers))
	for i, auth := range t.Authorizers {
		authorizers[i] = auth.Bytes()
	}

	// note(sideninja): This is a temporary workaround until cadence defines canonical format addressing the issue https://github.com/onflow/flow-go-sdk/issues/286
	for i, arg := range t.Arguments {
		if arg[len(arg)-1] == byte(10) { // extra new line character
			t.Arguments[i] = arg[:len(arg)-1]
		}
	}

	return payloadCanonicalForm{
		Script:                    t.Script,
		Arguments:                 t.Arguments,
		ReferenceBlockID:          t.ReferenceBlockID[:],
		GasLimit:                  t.GasLimit,
		ProposalKeyAddress:        t.ProposalKey.Address.Bytes(),
		ProposalKeyIndex:          uint64(t.ProposalKey.KeyIndex),
		ProposalKeySequenceNumber: t.ProposalKey.SequenceNumber,
		Payer:                     t.Payer.Bytes(),
		Authorizers:               authorizers,
	}
}

// EnvelopeMessage returns the signable message for the transaction envelope.
//
// This message is only signed by the payer account.
func (t *Transaction) EnvelopeMessage() []byte {
	temp := t.envelopeCanonicalForm()
	return mustRLPEncode(&temp)
}

func (t *Transaction) envelopeCanonicalForm() envelopeCanonicalForm {
	return envelopeCanonicalForm{
		Payload:           t.payloadCanonicalForm(),
		PayloadSignatures: signaturesList(t.PayloadSignatures).canonicalForm(),
	}
}

// Encode serializes the full transaction data including the payload and all signatures.
func (t *Transaction) Encode() []byte {
	temp := struct {
		Payload            payloadCanonicalForm
		PayloadSignatures  interface{}
		EnvelopeSignatures interface{}
	}{
		Payload:            t.payloadCanonicalForm(),
		PayloadSignatures:  signaturesList(t.PayloadSignatures).canonicalForm(),
		EnvelopeSignatures: signaturesList(t.EnvelopeSignatures).canonicalForm(),
	}

	return mustRLPEncode(&temp)
}

// DecodeTransaction decodes the input bytes into a Transaction struct
// able to decode outputs from PayloadMessage(), EnvelopeMessage() and Encode()
// functions
func DecodeTransaction(transactionMessage []byte) (*Transaction, error) {
	temp, err := decodeTransaction(transactionMessage)
	if err != nil {
		return nil, err
	}

	authorizers := make([]Address, len(temp.Payload.Authorizers))
	for i, auth := range temp.Payload.Authorizers {
		authorizers[i] = BytesToAddress(auth)
	}
	t := &Transaction{
		Script:           temp.Payload.Script,
		Arguments:        temp.Payload.Arguments,
		ReferenceBlockID: BytesToID(temp.Payload.ReferenceBlockID),
		GasLimit:         temp.Payload.GasLimit,
		ProposalKey: ProposalKey{
			Address:        BytesToAddress(temp.Payload.ProposalKeyAddress),
			KeyIndex:       int(temp.Payload.ProposalKeyIndex),
			SequenceNumber: temp.Payload.ProposalKeySequenceNumber,
		},
		Payer:       BytesToAddress(temp.Payload.Payer),
		Authorizers: authorizers,
	}
	signers := t.signerList()
	if len(temp.PayloadSignatures) > 0 {
		payloadSignatures := make([]TransactionSignature, len(temp.PayloadSignatures))
		for i, sig := range temp.PayloadSignatures {
			payloadSignatures[i] = transactionSignatureFromCanonicalForm(sig)
			payloadSignatures[i].Address = signers[payloadSignatures[i].SignerIndex]
		}
		t.PayloadSignatures = payloadSignatures
	}

	if len(temp.EnvelopeSignatures) > 0 {
		envelopeSignatures := make([]TransactionSignature, len(temp.EnvelopeSignatures))
		for i, sig := range temp.EnvelopeSignatures {
			envelopeSignatures[i] = transactionSignatureFromCanonicalForm(sig)
			envelopeSignatures[i].Address = signers[envelopeSignatures[i].SignerIndex]
		}
		t.EnvelopeSignatures = envelopeSignatures
	}

	if len(t.Arguments) == 0 {
		t.Arguments = nil
	}
	if len(t.Script) == 0 {
		t.Script = nil
	}
	return t, nil
}

func decodeTransaction(transactionMessage []byte) (*transactionCanonicalForm, error) {
	s := rlp.NewStream(bytes.NewReader(transactionMessage), 0)
	temp := &transactionCanonicalForm{}

	kind, _, err := s.Kind()
	if err != nil {
		return nil, err
	}

	// First kind should always be a list
	if kind != rlp.List {
		return nil, errors.New("unexpected rlp decoding type")
	}

	_, err = s.List()
	if err != nil {
		return nil, err
	}

	// Need to look at the type of the first element to determine if how we're going to be decoding
	kind, _, err = s.Kind()
	if err != nil {
		return nil, err
	}
	// If first kind is not list, safe to assume this is actually just encoded payload, and decrypt as such
	if kind != rlp.List {
		s.Reset(bytes.NewReader(transactionMessage), 0)
		txPayload := payloadCanonicalForm{}
		err := s.Decode(&txPayload)
		if err != nil {
			return nil, err
		}
		temp.Payload = txPayload
		return temp, nil
	}

	// If we're here, we will assume that we're decoding either a envelopeCanonicalForm
	// or a full transactionCanonicalForm

	// Decode the payload
	txPayload := payloadCanonicalForm{}
	err = s.Decode(&txPayload)
	if err != nil {
		return nil, err
	}
	temp.Payload = txPayload

	// Decode the payload sigs
	payloadSigs := []transactionSignatureCanonicalForm{}
	err = s.Decode(&payloadSigs)
	if err != nil {
		return nil, err
	}
	temp.PayloadSignatures = payloadSigs

	// It's possible for the envelope signature to not exist (e.g. envelopeCanonicalForm).
	kind, _, err = s.Kind()
	if errors.Is(err, rlp.EOL) {
		return temp, nil
	} else if err != nil {
		return nil, err
	}
	// If we're not at EOL, and no error, finish decoding
	envelopeSigs := []transactionSignatureCanonicalForm{}
	err = s.Decode(&envelopeSigs)
	if err != nil {
		return nil, err
	}
	temp.EnvelopeSignatures = envelopeSigs

	return temp, nil
}

// A ProposalKey is the key that specifies the proposal key and sequence number for a transaction.
type ProposalKey struct {
	Address        Address
	KeyIndex       int
	SequenceNumber uint64
}

// A TransactionSignature is a signature associated with a specific account key.
type TransactionSignature struct {
	Address     Address
	SignerIndex int
	KeyIndex    int
	Signature   []byte
}

type transactionSignatureCanonicalForm struct {
	SignerIndex uint
	KeyIndex    uint
	Signature   []byte
}

func (s TransactionSignature) canonicalForm() transactionSignatureCanonicalForm {
	return transactionSignatureCanonicalForm{
		SignerIndex: uint(s.SignerIndex), // int is not RLP-serializable
		KeyIndex:    uint(s.KeyIndex),    // int is not RLP-serializable
		Signature:   s.Signature,
	}
}

func transactionSignatureFromCanonicalForm(v transactionSignatureCanonicalForm) TransactionSignature {
	return TransactionSignature{
		SignerIndex: int(v.SignerIndex),
		KeyIndex:    int(v.KeyIndex),
		Signature:   v.Signature,
	}
}

func compareSignatures(signatures []TransactionSignature) func(i, j int) bool {
	return func(i, j int) bool {
		sigA := signatures[i]
		sigB := signatures[j]

		if sigA.SignerIndex == sigB.SignerIndex {
			return sigA.KeyIndex < sigB.KeyIndex
		}

		return sigA.SignerIndex < sigB.SignerIndex
	}
}

type signaturesList []TransactionSignature

func (s signaturesList) canonicalForm() []transactionSignatureCanonicalForm {
	signatures := make([]transactionSignatureCanonicalForm, len(s))

	for i, signature := range s {
		signatures[i] = signature.canonicalForm()
	}

	return signatures
}

type TransactionResult struct {
	Status        TransactionStatus
	Error         error
	Events        []Event
	BlockID       Identifier
	BlockHeight   uint64
	TransactionID Identifier
}

// TransactionStatus represents the status of a transaction.
type TransactionStatus int

const (
	// TransactionStatusUnknown indicates that the transaction status is not known.
	TransactionStatusUnknown TransactionStatus = iota
	// TransactionStatusPending is the status of a pending transaction.
	TransactionStatusPending
	// TransactionStatusFinalized is the status of a finalized transaction.
	TransactionStatusFinalized
	// TransactionStatusExecuted is the status of an executed transaction.
	TransactionStatusExecuted
	// TransactionStatusSealed is the status of a sealed transaction.
	TransactionStatusSealed
	// TransactionStatusExpired is the status of an expired transaction.
	TransactionStatusExpired
)

// String returns the string representation of a transaction status.
func (s TransactionStatus) String() string {
	return [...]string{"UNKNOWN", "PENDING", "FINALIZED", "EXECUTED", "SEALED", "EXPIRED"}[s]
}
