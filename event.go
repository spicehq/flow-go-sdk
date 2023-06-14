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
	"time"

	"github.com/onflow/cadence"
)

// List of built-in account event types.
const (
	EventAccountCreated         string = "flow.AccountCreated"
	EventAccountKeyAdded        string = "flow.AccountKeyAdded"
	EventAccountKeyRemoved      string = "flow.AccountKeyRemoved"
	EventAccountContractAdded   string = "flow.AccountContractAdded"
	EventAccountContractUpdated string = "flow.AccountContractUpdated"
	EventAccountContractRemoved string = "flow.AccountContractRemoved"
)

type Event struct {
	// Type is the qualified event type.
	Type string
	// TransactionID is the ID of the transaction this event was emitted from.
	TransactionID Identifier
	// TransactionIndex is the index of the transaction this event was emitted from, within its containing block.
	TransactionIndex int
	// EventIndex is the index of the event within the transaction it was emitted from.
	EventIndex int
	// Value contains the event data.
	Value cadence.Event
	// Bytes representing event data.
	Payload []byte
}

// Encode returns the canonical RLP byte representation of this event.
func (e Event) Encode() []byte {
	temp := struct {
		TransactionID []byte
		EventIndex    uint
	}{
		TransactionID: e.TransactionID[:],
		EventIndex:    uint(e.EventIndex),
	}
	return mustRLPEncode(&temp)
}

// Fingerprint calculates a fingerprint of an event.
func (e *Event) Fingerprint() []byte {

	return mustRLPEncode(struct {
		TxID             []byte
		Index            uint32
		Type             string
		TransactionIndex uint32
		Payload          []byte
	}{
		TxID:             e.TransactionID[:],
		Index:            uint32(e.EventIndex),
		Type:             string(e.Type),
		TransactionIndex: uint32(e.TransactionIndex),
		Payload:          e.Payload[:],
	})
}

// BlockEvents are the events that occurred in a specific block.
type BlockEvents struct {
	BlockID        Identifier
	Height         uint64
	BlockTimestamp time.Time
	Events         []Event
}

// An AccountCreatedEvent is emitted when a transaction creates a new Flow account.
//
// This event contains the following fields:
// - Address: Address
type AccountCreatedEvent Event

// Address returns the address of the newly-created account.
func (evt AccountCreatedEvent) Address() Address {
	return BytesToAddress(evt.Value.Fields[0].(cadence.Address).Bytes())
}
