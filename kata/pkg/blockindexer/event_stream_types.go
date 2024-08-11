// Copyright © 2024 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
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

package blockindexer

import (
	"github.com/google/uuid"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/kaleido-io/paladin/kata/pkg/types"
)

type EventStreamConfig struct {
	BatchSize    *int64  `json:"batchSize,omitempty"`
	BatchTimeout *string `json:"batchTimeout,omitempty"`
}

type EventStreamType string

func (est EventStreamType) Options() []string {
	return []string{
		"internal", // a core Paladin component, such as the state confirmation engine
	}
}
func (es EventStreamType) Default() string { return "internal" }

type EventStream struct {
	ID     uuid.UUID                      `json:"id"                     gorm:"primaryKey"`
	Name   string                         `json:"name"`
	Type   types.Enum[EventStreamType]    `json:"type"`
	Config types.JSONP[EventStreamConfig] `json:"config"`
	ABI    types.JSONP[abi.ABI]           `json:"abi,omitempty"` // immutable (event delivery behavior would be too undefined with mutability)
}

type EventStreamCheckpoint struct {
	ID          uuid.UUID `json:"id"                                     gorm:"primaryKey"`
	BlockNumber int64     `json:"blockNumber"                            gorm:"primaryKey"`
}

type EventStreamSignature struct {
	Stream    uuid.UUID    `json:"stream"                                gorm:"primaryKey"`
	Signature types.HashID `json:"signature"                             gorm:"primaryKey;embedded;embeddedPrefix:signature_;"`
}

type EventWithData struct {
	Stream uuid.UUID `json:"stream"`
	*IndexedEvent
	Address types.EthAddress `json:"address"`
	Data    types.RawJSON    `json:"data"`
}
