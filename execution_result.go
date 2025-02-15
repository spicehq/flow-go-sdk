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

type ExecutionResult struct {
	PreviousResultID Identifier // commit of the previous ER
	BlockID          Identifier // commit of the current block
	Chunks           []*Chunk
	ServiceEvents    []*ServiceEvent
}

type Chunk struct {
	CollectionIndex      uint
	StartState           StateCommitment // start state when starting executing this chunk
	BlockID              Identifier      // Block id of the execution result this chunk belongs to
	TotalComputationUsed uint64          // total amount of computation used by running all txs in this chunk
	NumberOfTransactions uint16          // number of transactions inside the collection
	Index                uint64          // chunk index inside the ER (starts from zero)
	EndState             StateCommitment // EndState inferred from next chunk or from the ER
}

type ServiceEvent struct {
	Type    string
	Payload []byte
}
