package p2p

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	p2pFactory "github.com/ElrondNetwork/elrond-go/p2p/factory"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/stretchr/testify/assert"
)

const providedShard = "5"

func createMockArgInterceptedDirectConnectionInfo() ArgInterceptedDirectConnectionInfo {
	marshaller := &marshal.GogoProtoMarshalizer{}
	msg := &p2pFactory.DirectConnectionInfo{
		ShardId: providedShard,
	}
	msgBuff, _ := marshaller.Marshal(msg)

	return ArgInterceptedDirectConnectionInfo{
		Marshaller:  marshaller,
		DataBuff:    msgBuff,
		NumOfShards: 10,
	}
}
func TestNewInterceptedDirectConnectionInfo(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgInterceptedDirectConnectionInfo()
		args.Marshaller = nil

		idci, err := NewInterceptedDirectConnectionInfo(args)
		assert.Equal(t, process.ErrNilMarshalizer, err)
		assert.True(t, check.IfNil(idci))
	})
	t.Run("nil data buff should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgInterceptedDirectConnectionInfo()
		args.DataBuff = nil

		idci, err := NewInterceptedDirectConnectionInfo(args)
		assert.Equal(t, process.ErrNilBuffer, err)
		assert.True(t, check.IfNil(idci))
	})
	t.Run("invalid num of shards should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgInterceptedDirectConnectionInfo()
		args.NumOfShards = 0

		idci, err := NewInterceptedDirectConnectionInfo(args)
		assert.Equal(t, process.ErrInvalidValue, err)
		assert.True(t, check.IfNil(idci))
	})
	t.Run("unmarshal returns error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgInterceptedDirectConnectionInfo()
		args.DataBuff = []byte("invalid data")

		idci, err := NewInterceptedDirectConnectionInfo(args)
		assert.NotNil(t, err)
		assert.True(t, check.IfNil(idci))
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		idci, err := NewInterceptedDirectConnectionInfo(createMockArgInterceptedDirectConnectionInfo())
		assert.Nil(t, err)
		assert.False(t, check.IfNil(idci))
	})
}

func Test_interceptedDirectConnectionInfo_CheckValidity(t *testing.T) {
	t.Parallel()

	t.Run("invalid shard string should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgInterceptedDirectConnectionInfo()
		msg := &p2pFactory.DirectConnectionInfo{
			ShardId: "invalid shard",
		}
		msgBuff, _ := args.Marshaller.Marshal(msg)
		args.DataBuff = msgBuff
		idci, err := NewInterceptedDirectConnectionInfo(args)
		assert.Nil(t, err)
		assert.False(t, check.IfNil(idci))

		err = idci.CheckValidity()
		assert.NotNil(t, err)
	})
	t.Run("invalid shard should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgInterceptedDirectConnectionInfo()
		ps, _ := strconv.ParseInt(providedShard, 10, 32)
		args.NumOfShards = uint32(ps - 1)

		idci, err := NewInterceptedDirectConnectionInfo(args)
		assert.Nil(t, err)
		assert.False(t, check.IfNil(idci))

		err = idci.CheckValidity()
		assert.Equal(t, process.ErrInvalidValue, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		idci, err := NewInterceptedDirectConnectionInfo(createMockArgInterceptedDirectConnectionInfo())
		assert.Nil(t, err)
		assert.False(t, check.IfNil(idci))

		err = idci.CheckValidity()
		assert.Nil(t, err)
	})
}

func Test_interceptedDirectConnectionInfo_Getters(t *testing.T) {
	t.Parallel()

	idci, err := NewInterceptedDirectConnectionInfo(createMockArgInterceptedDirectConnectionInfo())
	assert.Nil(t, err)
	assert.False(t, check.IfNil(idci))

	assert.True(t, idci.IsForCurrentShard())
	assert.True(t, bytes.Equal([]byte(""), idci.Hash()))
	assert.Equal(t, interceptedDirectConnectionInfoType, idci.Type())
	identifiers := idci.Identifiers()
	assert.Equal(t, 1, len(identifiers))
	assert.True(t, bytes.Equal([]byte(""), identifiers[0]))
	assert.Equal(t, fmt.Sprintf("shard=%s", providedShard), idci.String())
	assert.Equal(t, providedShard, idci.ShardID())
}
