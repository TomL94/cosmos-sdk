package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/tendermint/abci/types"

	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	"github.com/cosmos/cosmos-sdk/x/ibc/03-connection/exported"
	"github.com/cosmos/cosmos-sdk/x/ibc/03-connection/types"
	tendermint "github.com/cosmos/cosmos-sdk/x/ibc/07-tendermint"
	commitment "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment"
	ibctypes "github.com/cosmos/cosmos-sdk/x/ibc/types"
)

const (
	clientType = clientexported.Tendermint
	storeKey   = ibctypes.StoreKey
	chainID    = "test"
	testHeight = 10

	testClientID1     = "testclientid1"
	testConnectionID1 = "connectionid1"

	testClientID2     = "testclientid2"
	testConnectionID2 = "connectionid2"

	testClientID3     = "testclientid3"
	testConnectionID3 = "connectionid3"
)

type KeeperTestSuite struct {
	suite.Suite

	cdc            *codec.Codec
	ctx            sdk.Context
	app            *simapp.SimApp
	valSet         *tmtypes.ValidatorSet
	consensusState clientexported.ConsensusState
	header         tendermint.Header
}

func (suite *KeeperTestSuite) SetupTest() {
	isCheckTx := false
	app := simapp.Setup(isCheckTx)

	suite.cdc = app.Codec()
	suite.ctx = app.BaseApp.NewContext(isCheckTx, abci.Header{ChainID: chainID, Height: testHeight})
	suite.app = app

	privVal := tmtypes.NewMockPV()

	validator := tmtypes.NewValidator(privVal.GetPubKey(), 1)
	suite.valSet = tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})
	suite.header = tendermint.CreateTestHeader(chainID, testHeight, suite.valSet, suite.valSet, []tmtypes.PrivValidator{privVal})
	suite.consensusState = tendermint.ConsensusState{
		Root:             commitment.NewRoot(suite.header.AppHash),
		ValidatorSetHash: suite.valSet.Hash(),
	}
}

func (suite *KeeperTestSuite) queryProof(key []byte) (commitment.Proof, int64) {
	res := suite.app.Query(abci.RequestQuery{
		Path:  fmt.Sprintf("store/%s/key", storeKey),
		Data:  key,
		Prove: true,
	})

	height := res.Height
	proof := commitment.Proof{
		Proof: res.Proof,
	}

	return proof, height
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestSetAndGetConnection() {
	_, existed := suite.app.IBCKeeper.ConnectionKeeper.GetConnection(suite.ctx, testConnectionID1)
	suite.False(existed)

	counterparty := types.NewCounterparty(testClientID1, testConnectionID1, suite.app.IBCKeeper.ConnectionKeeper.GetCommitmentPrefix())
	expConn := types.NewConnectionEnd(exported.INIT, testClientID1, counterparty, types.GetCompatibleVersions())
	suite.app.IBCKeeper.ConnectionKeeper.SetConnection(suite.ctx, testConnectionID1, expConn)
	conn, existed := suite.app.IBCKeeper.ConnectionKeeper.GetConnection(suite.ctx, testConnectionID1)
	suite.True(existed)
	suite.EqualValues(expConn, conn)
}

func (suite *KeeperTestSuite) TestSetAndGetClientConnectionPaths() {
	_, existed := suite.app.IBCKeeper.ConnectionKeeper.GetClientConnectionPaths(suite.ctx, testClientID1)
	suite.False(existed)

	suite.app.IBCKeeper.ConnectionKeeper.SetClientConnectionPaths(suite.ctx, testClientID1, types.GetCompatibleVersions())
	paths, existed := suite.app.IBCKeeper.ConnectionKeeper.GetClientConnectionPaths(suite.ctx, testClientID1)
	suite.True(existed)
	suite.EqualValues(types.GetCompatibleVersions(), paths)
}

func (suite KeeperTestSuite) TestGetAllConnections() {
	// Connection (Counterparty): A(C) -> C(B) -> B(A)
	counterparty1 := types.NewCounterparty(testClientID1, testConnectionID1, suite.app.IBCKeeper.ConnectionKeeper.GetCommitmentPrefix())
	counterparty2 := types.NewCounterparty(testClientID2, testConnectionID2, suite.app.IBCKeeper.ConnectionKeeper.GetCommitmentPrefix())
	counterparty3 := types.NewCounterparty(testClientID3, testConnectionID3, suite.app.IBCKeeper.ConnectionKeeper.GetCommitmentPrefix())

	conn1 := types.NewConnectionEnd(exported.INIT, testClientID1, counterparty3, types.GetCompatibleVersions())
	conn2 := types.NewConnectionEnd(exported.INIT, testClientID2, counterparty1, types.GetCompatibleVersions())
	conn3 := types.NewConnectionEnd(exported.UNINITIALIZED, testClientID3, counterparty2, types.GetCompatibleVersions())

	expConnections := []types.ConnectionEnd{conn1, conn2, conn3}

	suite.app.IBCKeeper.ConnectionKeeper.SetConnection(suite.ctx, testConnectionID1, expConnections[0])
	suite.app.IBCKeeper.ConnectionKeeper.SetConnection(suite.ctx, testConnectionID2, expConnections[1])
	suite.app.IBCKeeper.ConnectionKeeper.SetConnection(suite.ctx, testConnectionID3, expConnections[2])

	connections := suite.app.IBCKeeper.ConnectionKeeper.GetAllConnections(suite.ctx)
	suite.Require().Len(connections, len(expConnections))
	suite.Require().Equal(expConnections, connections)
}
