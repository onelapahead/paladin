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

package persistence

import (
	"context"

	// Import pq driver
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/kaleido-io/paladin/config/pkg/pldconf"
	"github.com/kaleido-io/paladin/core/internal/msgs"
	"gorm.io/gorm"
)

type Persistence interface {
	DB() *gorm.DB
	Close()
}

const (
	TypePostgres = "postgres"
	TypeSQLite   = "sqlite"
)

func NewPersistence(ctx context.Context, conf *pldconf.DBConfig) (Persistence, error) {
	switch conf.Type {
	case "", TypeSQLite: // default
		return newSQLiteProvider(ctx, conf)
	case TypePostgres:
		return newPostgresProvider(ctx, conf)
	default:
		return nil, i18n.NewError(ctx, msgs.MsgPersistenceInvalidType, conf.Type)
	}
}
