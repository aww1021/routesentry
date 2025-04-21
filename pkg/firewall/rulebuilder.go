package firewall

import (
	"net"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
)

type RuleBuilder struct {
	table *nftables.Table
	chain *nftables.Chain
	exprs []expr.Any
}

func newRuleBuilder(table *nftables.Table, chain *nftables.Chain) *RuleBuilder {
	return &RuleBuilder{
		table: table,
		chain: chain,
		exprs: []expr.Any{},
	}
}

func nullTerminate(s string) []byte {
	return append([]byte(s), 0)
}

func (r *RuleBuilder) MatchMetaOIFName(iface string) *RuleBuilder {
	r.exprs = append(
		r.exprs,
		&expr.Meta{ // Get the OIFName
			Key:      expr.MetaKeyOIFNAME,
			Register: 1,
		},
		&expr.Cmp{ // compare to iface name
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     nullTerminate(iface),
		},
	)
	return r
}

func (r *RuleBuilder) MatchL4Proto(proto L4Proto) *RuleBuilder {
	r.exprs = append(
		r.exprs,
		&expr.Meta{
			Key:      expr.MetaKeyL4PROTO,
			Register: 1,
		},
		&expr.Cmp{
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     proto.ToData(),
		},
	)
	return r
}

func (r *RuleBuilder) MatchDestinationIP(daddr net.IP) *RuleBuilder {
	r.exprs = append(
		r.exprs,
		&expr.Payload{
			OperationType: expr.PayloadLoad,
			DestRegister:  1,
			Base:          expr.PayloadBaseNetworkHeader,
			Offset:        16, // offset of destination address in IPv4 Header
			Len:           4,  // 4 bytes long
		},
		&expr.Cmp{
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     daddr.To4(),
		},
	)
	return r
}

func (r *RuleBuilder) MatchSourceIP(saddr net.IP) *RuleBuilder {
	r.exprs = append(
		r.exprs,
		&expr.Payload{
			OperationType: expr.PayloadLoad,
			DestRegister:  1,
			Base:          expr.PayloadBaseNetworkHeader,
			Offset:        12, // offset of source address in IPv4 Header
			Len:           4,  // 4 bytes long
		},
		&expr.Cmp{
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     saddr.To4(),
		},
	)
	return r
}

func (r *RuleBuilder) MatchUDPDestPort(dport uint16) *RuleBuilder {
	r.exprs = append(
		r.exprs,
		&expr.Payload{
			OperationType: expr.PayloadLoad,
			DestRegister:  1,
			Base:          expr.PayloadBaseTransportHeader,
			Offset:        2, // offset of destination port in UDP Header
			Len:           2, // 2 bytes long
		},
		&expr.Cmp{
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     htons(dport),
		},
	)
	return r
}

func (r *RuleBuilder) CtStateIn(ctStateMask uint8) *RuleBuilder {
	r.exprs = append(
		r.exprs,
		&expr.Ct{
			Register: 1,
			Key:      expr.CtKeySTATE,
		},
		&expr.Bitwise{
			SourceRegister: 1,
			DestRegister:   1,
			Len:            1,                   // 1 byte
			Mask:           []byte{ctStateMask}, // (ctState & ctStateMask)
			Xor:            []byte{0},           // do no xor
		},
		&expr.Cmp{
			Op:       expr.CmpOpNeq,
			Register: 1,
			Data:     []byte{0}, // cmp on result of the mask, if there are common bits: reg != 0
		},
	)
	return r
}

func (r *RuleBuilder) Verdict(verdict expr.VerdictKind) *RuleBuilder {
	r.exprs = append(r.exprs, &expr.Verdict{
		Kind: verdict,
	})
	return r
}

func (r *RuleBuilder) Build() *nftables.Rule {
	return &nftables.Rule{
		Table: r.table,
		Chain: r.chain,
		Exprs: r.exprs,
	}
}
