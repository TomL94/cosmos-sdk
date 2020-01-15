package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	clienterrors "github.com/cosmos/cosmos-sdk/x/ibc/02-client/types/errors"
	"github.com/cosmos/cosmos-sdk/x/ibc/03-connection/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	commitment "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment"
)

// VerifyClientConsensusState verifies a proof of the consensus state of the
// specified client stored on the target machine.
func (k Keeper) VerifyClientConsensusState(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height uint64,
	proof commitment.ProofI,
	consensusState clientexported.ConsensusState,
) error {
	clientState, found := k.clientKeeper.GetClientState(ctx, connection.ClientID)
	if !found {
		return clienterrors.ErrClientNotFound
	}

	return clientState.VerifyClientConsensusState(
		k.cdc, height, connection.Counterparty.Prefix, proof, clientState.GetID(), consensusState,
	)
}

// VerifyConnectionState verifies a proof of the connection state of the
// specified connection end stored on the target machine.
func (k Keeper) VerifyConnectionState(
	ctx sdk.Context,
	height uint64,
	proof commitment.ProofI,
	connectionID string,
	connection types.ConnectionEnd,
	consensusState clientexported.ConsensusState,
) error {
	clientState, found := k.clientKeeper.GetClientState(ctx, connection.ClientID)
	if !found {
		return clienterrors.ErrClientNotFound
	}

	return clientState.VerifyConnectionState(
		k.cdc, height, connection.Counterparty.Prefix, proof, connectionID, connection, consensusState,
	)
}

// VerifyChannelState verifies a proof of the channel state of the specified
// channel end, under the specified port, stored on the target machine.
func (k Keeper) VerifyChannelState(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height uint64,
	prefix commitment.PrefixI,
	proof commitment.ProofI,
	portID,
	channelID string,
	channel channeltypes.Channel,
	consensusState clientexported.ConsensusState,
) error {
	clientState, found := k.clientKeeper.GetClientState(ctx, connection.ClientID)
	if !found {
		return clienterrors.ErrClientNotFound
	}

	return clientState.VerifyChannelState(
		k.cdc, height, prefix, proof, portID, channelID, channel, consensusState,
	)
}

// VerifyPacketCommitment verifies a proof of an outgoing packet commitment at
// the specified port, specified channel, and specified sequence.
func (k Keeper) VerifyPacketCommitment(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height uint64,
	prefix commitment.PrefixI,
	proof commitment.ProofI,
	portID,
	channelID string,
	sequence uint64,
	commitmentBytes []byte,
	consensusState clientexported.ConsensusState,
) error {
	clientState, found := k.clientKeeper.GetClientState(ctx, connection.ClientID)
	if !found {
		return clienterrors.ErrClientNotFound
	}

	return clientState.VerifyPacketCommitment(
	 height, prefix, proof, portID, channelID, sequence, commitmentBytes, consensusState,
	)
}

// VerifyPacketAcknowledgement verifies a proof of an incoming packet
// acknowledgement at the specified port, specified channel, and specified sequence.
func (k Keeper) VerifyPacketAcknowledgement(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height uint64,
	prefix commitment.PrefixI,
	proof commitment.ProofI,
	portID,
	channelID string,
	sequence uint64,
	acknowledgement []byte,
	consensusState clientexported.ConsensusState,
) error {
	clientState, found := k.clientKeeper.GetClientState(ctx, connection.ClientID)
	if !found {
		return clienterrors.ErrClientNotFound
	}

	return clientState.VerifyPacketAcknowledgement(
		height, prefix, proof, portID, channelID, sequence, acknowledgement, consensusState,
	)
}

// VerifyPacketAcknowledgementAbsence verifies a proof of the absence of an
// incoming packet acknowledgement at the specified port, specified channel, and
// specified sequence.
func (k Keeper) VerifyPacketAcknowledgementAbsence(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height uint64,
	prefix commitment.PrefixI,
	proof commitment.ProofI,
	portID,
	channelID string,
	sequence uint64,
	consensusState clientexported.ConsensusState,
) error {
	clientState, found := k.clientKeeper.GetClientState(ctx, connection.ClientID)
	if !found {
		return clienterrors.ErrClientNotFound
	}

	return clientState.VerifyPacketAcknowledgementAbsence(
		height, prefix, proof, portID, channelID, sequence, consensusState,
	)
}

// VerifyNextSequenceRecv verifies a proof of the next sequence number to be
// received of the specified channel at the specified port.
func (k Keeper) VerifyNextSequenceRecv(
	ctx sdk.Context,
	connection types.ConnectionEnd,
	height uint64,
	prefix commitment.PrefixI,
	proof commitment.ProofI,
	portID,
	channelID string,
	nextSequenceRecv uint64,
	consensusState clientexported.ConsensusState,
) error {
	clientState, found := k.clientKeeper.GetClientState(ctx, connection.ClientID)
	if !found {
		return clienterrors.ErrClientNotFound
	}

	return clientState.VerifyNextSequenceRecv(
		height, prefix, proof, portID, channelID, nextSequenceRecv, consensusState,
	)
}
