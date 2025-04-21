package firewall

import (
	"fmt"

	"github.com/google/nftables"
	"k8s.io/utils/ptr"
)

const (
	nfFilterTableName = "routesentry_filter"
	inputChainName    = "input"
	outputChainName   = "output"
	forwardChainName  = "forward"
)

type ChainType int

const (
	Input = iota
	Output
	Forward
)

type Firewall struct {
	conn         *nftables.Conn
	filterTable  *nftables.Table
	inChain      *nftables.Chain
	outChain     *nftables.Chain
	forwardChain *nftables.Chain
}

// New returns a new *Firewall with the given Option list
// It sets up a new table which by default drops all Input/Output/Forward traffic
// But does not actually commit this table until *Firewall.Flush() is called
func New(opts ...Option) (*Firewall, error) {

	cfg := &config{
		TableName:        nfFilterTableName,
		TableFamily:      nftables.TableFamilyINet,
		InputChainName:   inputChainName,
		OutputChainName:  outputChainName,
		ForwardChainName: forwardChainName,
		FlushRulesOnInit: true,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	conn, err := nftables.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create nftables connection: %w", err)
	}

	if cfg.FlushRulesOnInit {
		// remove all existing rulesets
		conn.FlushRuleset()
	}

	table := &nftables.Table{
		Name:   cfg.TableName,
		Family: cfg.TableFamily,
	}
	conn.AddTable(table)

	inChain := conn.AddChain(&nftables.Chain{
		Name:     cfg.InputChainName,
		Table:    table,
		Hooknum:  nftables.ChainHookInput,
		Priority: nftables.ChainPriorityFilter,
		Type:     nftables.ChainTypeFilter,
		Policy:   ptr.To(nftables.ChainPolicyDrop),
	})

	outChain := conn.AddChain(&nftables.Chain{
		Name:     cfg.OutputChainName,
		Table:    table,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityFilter,
		Type:     nftables.ChainTypeFilter,
		Policy:   ptr.To(nftables.ChainPolicyDrop),
	})

	forwardChain := conn.AddChain(&nftables.Chain{
		Name:     cfg.ForwardChainName,
		Table:    table,
		Hooknum:  nftables.ChainHookForward,
		Priority: nftables.ChainPriorityFilter,
		Type:     nftables.ChainTypeFilter,
		Policy:   ptr.To(nftables.ChainPolicyDrop),
	})

	return &Firewall{
		conn:         conn,
		filterTable:  table,
		inChain:      inChain,
		outChain:     outChain,
		forwardChain: forwardChain,
	}, nil
}

func (f *Firewall) AddRule(r *nftables.Rule) *nftables.Rule {
	return f.conn.AddRule(r)
}

func (f *Firewall) Flush() error {
	return f.conn.Flush()
}

func (f *Firewall) NewRuleBuilder(c ChainType) *RuleBuilder {
	switch c {
	case Input:
		return newRuleBuilder(f.filterTable, f.inChain)
	case Output:
		return newRuleBuilder(f.filterTable, f.outChain)
	case Forward:
		return newRuleBuilder(f.filterTable, f.forwardChain)
	default:
		return nil
	}
}
