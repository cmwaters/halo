package consensus

import (
	"crypto"
	"errors"
	"fmt"

	"github.com/cmwaters/halo/pkg/group"
	"google.golang.org/protobuf/proto"
)

type verifier struct {
	namespace  string
	group      group.Group
	hasher     crypto.Hash
}

func NewVerifier(namespace string, group group.Group, hasher crypto.Hash) *verifier {
	return &verifier{
		namespace:  namespace,
		group:      group,
		hasher:     hasher,
	}
}

func (v *verifier) GetProposer(round uint32) group.Member {
	return v.group.Proposer(uint(round))
}

func (v *verifier) VerifyProposal(proposal *Proposal, height uint64, round uint32) error {
	if len(proposal.Signature) == 0 {
		return errors.New("proposal signature missing")
	}

	if proposal.Height != proposal.Height {
		return fmt.Errorf("proposal is from a different height (exp: %d, got: %d)", height, proposal.Height)
	}

	if proposal.Round > round {
		return fmt.Errorf("proposal is from a round in the future (exp: %d, got: %d)", round, proposal.Round)
	}

	proposer := v.group.Proposer(uint(proposal.Round))
	if proposer.Verify(v.ProposalMessageBytes(proposal), proposal.Signature) {
		// This could be caused by one of a few things:
		// - The proposal comes from a member who is not currently the proposer
		// - The proposer has incorrectly signed a different set of data
		// - There has been a breaking change to how a proposal message is serialized
		return errors.New("invalid proposal signature")
	}

	return nil
}

func (v *verifier) Hash(data []byte) []byte {
	hash := v.hasher.New()
	return hash.Sum(data)
}

func (v *verifier) ProposalMessageBytes(proposal *Proposal) []byte {
	sigMsg := &SignatureMessage{
		Type:       SignatureMessage_PROPOSE,
		Height:     int64(proposal.Height),
		Round:      int32(proposal.Round),
		Namespace:  v.namespace,
		DataDigest: v.Hash(proposal.Data),
	}
	bz, err := proto.Marshal(sigMsg)
	if err != nil {
		panic(err)
	}
	return bz
}

func (v *verifier) VerifyVote(vote *Vote, height uint64, dataDigest []byte) error {
	if err := vote.ValidateForm(); err != nil {
		return err
	}

	if vote.Height != height {
		return fmt.Errorf("vote is from a different height (exp: %d, got: %d)", height, vote.Height)
	}

	if int(vote.MemberIndex) >= len(v.group.members) {
		return fmt.Errorf("invalid member index exceeds total members (%d)", vote.MemberIndex)
	}

	member := v.group.GetMember(vote.MemberIndex)
	if !v.verifyFunc(member.PublicKey, v.VoteMessageBytes(vote, dataDigest), vote.Signature) {
		return errors.New("invalid vote signature")
	}

	return nil
}

func (v *verifier) VoteMessageBytes(vote *Vote, dataDigest []byte) []byte {
	sigMsg := &SignatureMessage{
		Type:       SignatureMessage_Type(vote.Type),
		Height:     int64(vote.Height),
		Round:      int32(vote.Round),
		Namespace:  v.namespace,
		DataDigest: dataDigest,
	}
	bz, err := proto.Marshal(sigMsg)
	if err != nil {
		panic(err)
	}
	return bz
}

func (p *Parameters) Validate() error {
	if p.RoundTimeout.AsDuration() == 0 {
		return errors.New("Round Timeout must be greater than zero")
	}
	return nil
}