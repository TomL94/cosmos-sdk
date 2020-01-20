package keeper

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/02-client/types"
	connection "github.com/cosmos/cosmos-sdk/x/ibc/03-connection"
	"github.com/cosmos/cosmos-sdk/x/ibc/04-channel/exported"
	"github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	commitment "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment"
)

// TimeoutPacket is called by a module which originally attempted to send a
// packet to a counterparty module, where the timeout height has passed on the
// counterparty chain without the packet being committed, to prove that the
// packet can no longer be executed and to allow the calling module to safely
// perform appropriate state transitions.
func (k Keeper) TimeoutPacket(
	ctx sdk.Context,
	packet exported.PacketI,
	proof commitment.ProofI,
	proofHeight,
	nextSequenceRecv uint64,
) (exported.PacketI, error) {
	channel, found := k.GetChannel(ctx, packet.GetSourcePort(), packet.GetSourceChannel())
	if !found {
		return nil, sdkerrors.Wrapf(
			types.ErrChannelNotFound,
			packet.GetSourcePort(), packet.GetSourceChannel(),
		)
	}

	if channel.State != exported.OPEN {
		return nil, sdkerrors.Wrapf(
			types.ErrInvalidChannelState,
			"channel state is not OPEN (got %s)", channel.State.String(),
		)
	}

	// NOTE: TimeoutPacket is called by the AnteHandler which acts upon the packet.Route(),
	// so the capability authentication can be omitted here

	if packet.GetDestPort() != channel.Counterparty.PortID {
		return nil, sdkerrors.Wrapf(
			types.ErrInvalidPacket,
			"packet destination port doesn't match the counterparty's port (%s ≠ %s)", packet.GetDestPort(), channel.Counterparty.PortID,
		)
	}

	if packet.GetDestChannel() != channel.Counterparty.ChannelID {
		return nil, sdkerrors.Wrapf(
			types.ErrInvalidPacket,
			"packet destination channel doesn't match the counterparty's channel (%s ≠ %s)", packet.GetDestChannel(), channel.Counterparty.ChannelID,
		)
	}

	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return nil, sdkerrors.Wrap(
			connection.ErrConnectionNotFound,
			channel.ConnectionHops[0],
		)
	}

	// check that timeout height has passed on the other end
	if proofHeight < packet.GetTimeoutHeight() {
		return nil, types.ErrPacketTimeout
	}

	// check that packet has not been received
	if nextSequenceRecv >= packet.GetSequence() {
		return nil, sdkerrors.Wrap(types.ErrInvalidPacket, "packet already received")
	}

	commitment := k.GetPacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, types.CommitPacket(packet.GetData())) {
		return nil, sdkerrors.Wrap(types.ErrInvalidPacket, "packet hasn't been sent")
	}

	consensusState, found := k.clientKeeper.GetConsensusState(
		ctx, connectionEnd.GetClientID(), proofHeight,
	)
	if !found {
		return nil, clienttypes.ErrConsensusStateNotFound
	}

	var err error
	switch channel.Ordering {
	case exported.ORDERED:
		// check that the recv sequence is as claimed
		err = k.connectionKeeper.VerifyNextSequenceRecv(
			ctx, connectionEnd, proofHeight, proof,
			packet.GetDestPort(), packet.GetDestChannel(), nextSequenceRecv,
			consensusState,
		)
	case exported.UNORDERED:
		err = k.connectionKeeper.VerifyPacketAcknowledgementAbsence(
			ctx, connectionEnd, proofHeight, proof,
			packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
			consensusState,
		)
	default:
		panic(sdkerrors.Wrapf(types.ErrInvalidChannelOrdering, channel.Ordering.String()))
	}

	if err != nil {
		return nil, sdkerrors.Wrap(err, "packet verification failed")
	}

	// NOTE: the remaining code is located on the TimeoutExecuted function
	return packet, nil
}

// TimeoutExecuted deletes the commitment send from this chain after it verifies timeout
func (k Keeper) TimeoutExecuted(ctx sdk.Context, packet exported.PacketI) error {
	channel, found := k.GetChannel(ctx, packet.GetSourcePort(), packet.GetSourceChannel())
	if !found {
		return sdkerrors.Wrapf(types.ErrChannelNotFound, packet.GetSourcePort(), packet.GetSourceChannel())
	}

	// check if the packet is linked to a capability key
	// TODO: uncomment
	// _, found = k.GetChannelCapability(ctx, packet.GetSourcePort(), packet.GetSourceChannel())
	// if !found {
	// 	return types.ErrChannelCapabilityNotFound
	// }

	k.deletePacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	if channel.Ordering == exported.ORDERED {
		channel.State = exported.CLOSED
		k.SetChannel(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), channel)
	}

	return nil
}

// TimeoutOnClose is called by a module in order to prove that the channel to
// which an unreceived packet was addressed has been closed, so the packet will
// never be received (even if the timeoutHeight has not yet been reached).
func (k Keeper) TimeoutOnClose(
	ctx sdk.Context,
	packet types.Packet,
	proofNonMembership,
	proofClosed commitment.ProofI,
	proofHeight,
	nextSequenceRecv uint64,
) (exported.PacketI, error) {
	channel, found := k.GetChannel(ctx, packet.GetSourcePort(), packet.GetSourceChannel())
	if !found {
		return nil, sdkerrors.Wrapf(types.ErrChannelNotFound, packet.GetSourcePort(), packet.GetSourceChannel())
	}

	// TODO: uncomment
	// capKey, found := k.GetChannelCapability(ctx, packet.GetSourcePort(), packet.GetSourceChannel())
	// if !found {
	// 	return nil, types.ErrChannelCapabilityNotFound
	// }

	// portCapabilityKey := sdk.NewKVStoreKey(capKey)

	// if !k.portKeeper.Authenticate(portCapabilityKey, packet.GetSourcePort()) {
	// 	return nil, sdkerrors.Wrap(port.ErrInvalidPort, packet.GetSourcePort())
	// }

	if packet.GetDestPort() != channel.Counterparty.PortID {
		return nil, sdkerrors.Wrapf(
			types.ErrInvalidPacket,
			"packet destination port doesn't match the counterparty's port (%s ≠ %s)", packet.GetDestPort(), channel.Counterparty.PortID,
		)
	}

	if packet.GetDestChannel() != channel.Counterparty.ChannelID {
		return nil, sdkerrors.Wrapf(
			types.ErrInvalidPacket,
			"packet destination channel doesn't match the counterparty's channel (%s ≠ %s)", packet.GetDestChannel(), channel.Counterparty.ChannelID,
		)
	}

	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return nil, sdkerrors.Wrap(connection.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	commitment := k.GetPacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, types.CommitPacket(packet.GetData())) {
		return nil, sdkerrors.Wrap(types.ErrInvalidPacket, "packet hasn't been sent")
	}

	counterpartyHops, found := k.CounterpartyHops(ctx, channel)
	if !found {
		// Should not reach here, connectionEnd was able to be retrieved above
		panic("cannot find connection")
	}

	counterparty := types.NewCounterparty(packet.GetSourcePort(), packet.GetSourceChannel())
	expectedChannel := types.NewChannel(
		exported.CLOSED, channel.Ordering, counterparty, counterpartyHops, channel.Version,
	)

	consensusState, found := k.clientKeeper.GetConsensusState(
		ctx, connectionEnd.GetClientID(), proofHeight,
	)
	if !found {
		return nil, clienttypes.ErrConsensusStateNotFound
	}

	// check that the opposing channel end has closed
	if err := k.connectionKeeper.VerifyChannelState(
		ctx, connectionEnd, proofHeight, proofClosed,
		channel.Counterparty.PortID, channel.Counterparty.ChannelID,
		expectedChannel, consensusState,
	); err != nil {
		return nil, sdkerrors.Wrap(err, "channel membership verification failed")
	}

	var err error
	switch channel.Ordering {
	case exported.ORDERED:
		// check that the recv sequence is as claimed
		err = k.connectionKeeper.VerifyNextSequenceRecv(
			ctx, connectionEnd, proofHeight, proofClosed,
			packet.GetDestPort(), packet.GetDestChannel(), nextSequenceRecv,
			consensusState,
		)
	case exported.UNORDERED:
		err = k.connectionKeeper.VerifyPacketAcknowledgementAbsence(
			ctx, connectionEnd, proofHeight, proofClosed,
			packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence(),
			consensusState,
		)
	default:
		panic(sdkerrors.Wrapf(types.ErrInvalidChannelOrdering, channel.Ordering.String()))
	}

	if err != nil {
		return nil, sdkerrors.Wrap(err, "packet verification failed")
	}

	k.deletePacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	return packet, nil
}
