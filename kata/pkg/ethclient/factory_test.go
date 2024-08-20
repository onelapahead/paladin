/*
 * Copyright © 2024 Kaleido, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
 * an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package ethclient

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hyperledger/firefly-signer/pkg/ethsigner"
	"github.com/hyperledger/firefly-signer/pkg/ethtypes"
	"github.com/hyperledger/firefly-signer/pkg/rpcbackend"
	"github.com/kaleido-io/paladin/kata/internal/httpserver"
	"github.com/kaleido-io/paladin/kata/internal/rpcclient"
	"github.com/kaleido-io/paladin/kata/internal/rpcserver"
	"github.com/kaleido-io/paladin/kata/pkg/signer/api"
	"github.com/kaleido-io/paladin/toolkit/pkg/confutil"
	"github.com/stretchr/testify/assert"
)

type mockEth struct {
	eth_chainId             func(context.Context) (ethtypes.HexUint64, error)
	eth_getTransactionCount func(context.Context, ethtypes.Address0xHex, string) (ethtypes.HexUint64, error)
	eth_estimateGas         func(context.Context, ethsigner.Transaction) (ethtypes.HexInteger, error)
	eth_sendRawTransaction  func(context.Context, ethtypes.HexBytes0xPrefix) (ethtypes.HexBytes0xPrefix, error)
	eth_call                func(context.Context, ethsigner.Transaction, string) (ethtypes.HexBytes0xPrefix, error)
}

func newTestServer(t *testing.T, ctx context.Context, isWS bool, mEth *mockEth) (rpcServer rpcserver.Server, done func()) {
	var rpcServerConf *rpcserver.Config
	if isWS {
		rpcServerConf = &rpcserver.Config{
			HTTP: rpcserver.HTTPEndpointConfig{
				Disabled: true,
			},
			WS: rpcserver.WSEndpointConfig{
				Config: httpserver.Config{
					Port: confutil.P(0),
				},
			},
		}
	} else {
		rpcServerConf = &rpcserver.Config{
			HTTP: rpcserver.HTTPEndpointConfig{
				Config: httpserver.Config{
					Port: confutil.P(0),
				},
			},
			WS: rpcserver.WSEndpointConfig{
				Disabled: true,
			},
		}
	}

	rpcServer, err := rpcserver.NewServer(ctx, rpcServerConf)
	assert.NoError(t, err)

	if mEth.eth_chainId == nil {
		mEth.eth_chainId = func(ctx context.Context) (ethtypes.HexUint64, error) {
			return 12345, nil
		}
	}

	rpcServer.Register(rpcserver.NewRPCModule("eth").
		Add("eth_chainId", checkNil(mEth.eth_chainId, rpcserver.RPCMethod0)).
		Add("eth_getTransactionCount", checkNil(mEth.eth_getTransactionCount, rpcserver.RPCMethod2)).
		Add("eth_estimateGas", checkNil(mEth.eth_estimateGas, rpcserver.RPCMethod1)).
		Add("eth_sendRawTransaction", checkNil(mEth.eth_sendRawTransaction, rpcserver.RPCMethod1)).
		Add("eth_call", checkNil(mEth.eth_call, rpcserver.RPCMethod2)),
	)

	err = rpcServer.Start()
	assert.NoError(t, err)

	return rpcServer, func() {
		rpcServer.Stop()
	}
}

func checkNil[T any](v T, fn func(T) rpcserver.RPCHandler) rpcserver.RPCHandler {
	if !reflect.ValueOf(v).IsNil() {
		return fn(v)
	}
	return func(ctx context.Context, req *rpcbackend.RPCRequest) *rpcbackend.RPCResponse {
		return &rpcbackend.RPCResponse{
			JSONRpc: "2.0",
			ID:      req.ID,
			Error: &rpcbackend.RPCError{
				Code:    int64(rpcbackend.RPCCodeInvalidRequest),
				Message: "not implemented by test",
			},
		}
	}
}

func newTestClientAndServer(t *testing.T, mEth *mockEth) (ctx context.Context, _ *ethClientFactory, done func()) {
	ctx = context.Background()

	httpRPCServer, httpServerDone := newTestServer(t, ctx, false, mEth)
	wsRPCServer, wsServerDone := newTestServer(t, ctx, true, mEth)

	kmgr := newTestHDWalletKeyManager(t)

	conf := &Config{
		HTTP: rpcclient.HTTPConfig{
			URL: fmt.Sprintf("http://%s", httpRPCServer.HTTPAddr().String()),
		},
		WS: rpcclient.WSConfig{
			HTTPConfig: rpcclient.HTTPConfig{
				URL: fmt.Sprintf("ws://%s", wsRPCServer.WSAddr().String()),
			},
		},
	}

	ecf, err := NewEthClientFactory(ctx, kmgr, conf)
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), ecf.ChainID())

	return ctx, ecf.(*ethClientFactory), func() {
		httpServerDone()
		wsServerDone()
		ecf.Close()
	}

}

func TestNewEthClientFactoryBadConfig(t *testing.T) {
	kmgr, err := NewSimpleTestKeyManager(context.Background(), &api.Config{
		KeyStore: api.StoreConfig{Type: api.KeyStoreTypeStatic},
	})
	assert.NoError(t, err)
	_, err = NewEthClientFactory(context.Background(), kmgr, &Config{
		HTTP: rpcclient.HTTPConfig{
			URL: "http://ok.example.com",
		},
		WS: rpcclient.WSConfig{
			HTTPConfig: rpcclient.HTTPConfig{
				URL: "wrong://bad.example.com",
			},
		},
	})
	assert.Regexp(t, "PD011513", err)
}

func TestNewEthClientFactoryMissingURL(t *testing.T) {
	kmgr := newTestHDWalletKeyManager(t)
	_, err := NewEthClientFactory(context.Background(), kmgr, &Config{})
	assert.Regexp(t, "PD011511", err)
}

func TestNewEthClientFactoryBadURL(t *testing.T) {
	kmgr := newTestHDWalletKeyManager(t)
	_, err := NewEthClientFactory(context.Background(), kmgr, &Config{
		HTTP: rpcclient.HTTPConfig{
			URL: "wrong://type",
		},
	})
	assert.Regexp(t, "PD011514", err)
}

func TestNewEthClientFactoryChainIDFail(t *testing.T) {
	ctx := context.Background()
	rpcServer, done := newTestServer(t, ctx, false, &mockEth{
		eth_chainId: func(ctx context.Context) (ethtypes.HexUint64, error) { return 0, fmt.Errorf("pop") },
	})
	defer done()

	kmgr := newTestHDWalletKeyManager(t)
	_, err := NewEthClientFactory(context.Background(), kmgr, &Config{
		HTTP: rpcclient.HTTPConfig{
			URL: fmt.Sprintf("http://%s", rpcServer.HTTPAddr().String()),
		},
	})
	assert.Regexp(t, "PD011508.*pop", err)

}

func TestMismatchedChainID(t *testing.T) {
	ctx := context.Background()
	mEthHTTP := &mockEth{
		eth_chainId: func(ctx context.Context) (ethtypes.HexUint64, error) { return 22222, nil },
	}
	mEthWS := &mockEth{
		eth_chainId: func(ctx context.Context) (ethtypes.HexUint64, error) { return 11111, nil },
	}

	httpRPCServer, httpServerDone := newTestServer(t, ctx, false, mEthHTTP)
	defer httpServerDone()
	wsRPCServer, wsServerDone := newTestServer(t, ctx, true, mEthWS)
	defer wsServerDone()

	kmgr := newTestHDWalletKeyManager(t)

	conf := &Config{
		HTTP: rpcclient.HTTPConfig{
			URL: fmt.Sprintf("http://%s", httpRPCServer.HTTPAddr().String()),
		},
		WS: rpcclient.WSConfig{
			HTTPConfig: rpcclient.HTTPConfig{
				URL: fmt.Sprintf("ws://%s", wsRPCServer.WSAddr().String()),
			},
		},
	}

	_, err := NewEthClientFactory(ctx, kmgr, conf)
	assert.Regexp(t, "PD011512", err)

}
