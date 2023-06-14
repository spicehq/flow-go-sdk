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

package test

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"

	"github.com/onflow/flow-go-sdk"
)

type Accounts struct {
	addresses   *Addresses
	accountKeys *AccountKeys
}

func AccountGenerator() *Accounts {
	return &Accounts{
		addresses:   AddressGenerator(),
		accountKeys: AccountKeyGenerator(),
	}
}

type AccountKeys struct {
	count int
	ids   *Identifiers
}

func AccountKeyGenerator() *AccountKeys {
	return &AccountKeys{
		count: 1,
		ids:   IdentifierGenerator(),
	}
}

type Addresses struct {
	generator *flow.AddressGenerator
}

func AddressGenerator() *Addresses {
	return &Addresses{
		generator: flow.NewAddressGenerator(flow.Emulator),
	}
}

func (g *Addresses) New() flow.Address {
	return g.generator.NextAddress()
}

type Blocks struct {
	headers    *BlockHeaders
	guarantees *CollectionGuarantees
	seals      *BlockSeals
	signatures *Signatures
}

func BlockGenerator() *Blocks {
	return &Blocks{
		headers:    BlockHeaderGenerator(),
		guarantees: CollectionGuaranteeGenerator(),
		seals:      BlockSealGenerator(),
		signatures: SignaturesGenerator(),
	}
}

func (g *Blocks) New() *flow.Block {
	header := g.headers.New()

	guarantees := []*flow.CollectionGuarantee{
		g.guarantees.New(),
		g.guarantees.New(),
		g.guarantees.New(),
	}

	seals := []*flow.BlockSeal{
		g.seals.New(),
	}

	payload := flow.BlockPayload{
		CollectionGuarantees: guarantees,
		Seals:                seals,
	}

	return &flow.Block{
		BlockHeader:  header,
		BlockPayload: payload,
	}
}

type BlockHeaders struct {
	count     int
	ids       *Identifiers
	startTime time.Time
}

func BlockHeaderGenerator() *BlockHeaders {
	startTime, _ := time.Parse(time.RFC3339, "2020-06-04T15:43:21+00:00")

	return &BlockHeaders{
		count:     1,
		ids:       IdentifierGenerator(),
		startTime: startTime.UTC(),
	}
}

func (g *BlockHeaders) New() flow.BlockHeader {
	defer func() { g.count++ }()

	return flow.BlockHeader{
		ID:        g.ids.New(),
		ParentID:  g.ids.New(),
		Height:    uint64(g.count),
		Timestamp: g.startTime.Add(time.Hour * time.Duration(g.count)),
	}
}

type Collections struct {
	ids *Identifiers
}

func CollectionGenerator() *Collections {
	return &Collections{
		ids: IdentifierGenerator(),
	}
}

func (g *Collections) New() *flow.Collection {
	return &flow.Collection{
		TransactionIDs: []flow.Identifier{
			g.ids.New(),
			g.ids.New(),
		},
	}
}

type CollectionGuarantees struct {
	ids *Identifiers
}

type BlockSeals struct {
	ids *Identifiers
}

func CollectionGuaranteeGenerator() *CollectionGuarantees {
	return &CollectionGuarantees{
		ids: IdentifierGenerator(),
	}
}

func (g *CollectionGuarantees) New() *flow.CollectionGuarantee {
	return &flow.CollectionGuarantee{
		CollectionID: g.ids.New(),
	}
}

func BlockSealGenerator() *BlockSeals {
	return &BlockSeals{
		ids: IdentifierGenerator(),
	}
}

func (g *BlockSeals) New() *flow.BlockSeal {
	return &flow.BlockSeal{
		BlockID:            g.ids.New(),
		ExecutionReceiptID: g.ids.New(),
	}
}

type Events struct {
	count int
	ids   *Identifiers
}

func EventGenerator() *Events {
	return &Events{
		count: 1,
		ids:   IdentifierGenerator(),
	}
}

func (g *Events) New() flow.Event {
	defer func() { g.count++ }()

	identifier := fmt.Sprintf("FooEvent%d", g.count)

	location := common.StringLocation("test")

	testEventType := &cadence.EventType{
		Location:            location,
		QualifiedIdentifier: identifier,
		Fields: []cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.IntType{},
			},
			{
				Identifier: "b",
				Type:       cadence.StringType{},
			},
		},
	}

	testEvent := cadence.NewEvent(
		[]cadence.Value{
			cadence.NewInt(g.count),
			cadence.String("foo"),
		}).WithType(testEventType)

	typeID := location.TypeID(nil, identifier)

	payload, err := jsoncdc.Encode(testEvent)
	if err != nil {
		panic(fmt.Errorf("cannot encode test event: %w", err))
	}

	event := flow.Event{
		Type:             string(typeID),
		TransactionID:    g.ids.New(),
		TransactionIndex: g.count,
		EventIndex:       g.count,
		Value:            testEvent,
		Payload:          payload,
	}

	return event
}

type Signatures struct {
	count int
}

func SignaturesGenerator() *Signatures {
	return &Signatures{1}
}

func (g *Signatures) New() [][]byte {
	defer func() { g.count++ }()
	return [][]byte{
		[]byte(strconv.Itoa(g.count + 1)),
	}
}

func newSignatures(count int) Signatures {
	return Signatures{
		count: count,
	}
}

type Identifiers struct {
	count int
}

func IdentifierGenerator() *Identifiers {
	return &Identifiers{1}
}

func (g *Identifiers) New() flow.Identifier {
	defer func() { g.count++ }()
	return newIdentifier(g.count + 1)
}

func newIdentifier(count int) flow.Identifier {
	var id flow.Identifier
	for i := range id {
		id[i] = uint8(count)
	}

	return id
}

type Transactions struct {
	count     int
	greetings *Greetings
}

func TransactionGenerator() *Transactions {
	return &Transactions{
		count:     1,
		greetings: GreetingGenerator(),
	}
}

type TransactionResults struct {
	events *Events
}

func TransactionResultGenerator() *TransactionResults {
	return &TransactionResults{
		events: EventGenerator(),
	}
}

func (g *TransactionResults) New() flow.TransactionResult {
	eventA := g.events.New()
	eventB := g.events.New()
	blockID := newIdentifier(1)
	blockHeight := uint64(42)

	return flow.TransactionResult{
		Status: flow.TransactionStatusSealed,
		Error:  errors.New("transaction execution error"),
		Events: []flow.Event{
			eventA,
			eventB,
		},
		BlockID:     blockID,
		BlockHeight: blockHeight,
	}
}
