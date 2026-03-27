package main

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"proxy-gateway/api"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

type NftablesContext struct {
	conn             *nftables.Conn
	ttls             map[string]struct{}
	table            *nftables.Table
	prerouteNATChain *nftables.Chain
	inputFilterChain *nftables.Chain
	lookup4          *nftables.Set
	lookup6          *nftables.Set
}

func DialNftables() (*NftablesContext, error) {
	// dial nftables
	conn, err := nftables.New()
	if err != nil {
		return nil, fmt.Errorf("connect to nftables: %w", err)
	}
	ctx := &NftablesContext{
		conn: conn,
		ttls: make(map[string]struct{}),
	}

	// remove stale table first to avoid add/delete collisions in one batch
	existingTable, _ := conn.ListTableOfFamily("proxy_gateway", nftables.TableFamilyINet)
	if existingTable != nil {
		conn.DelTable(existingTable)
		if err := conn.Flush(); err != nil {
			return nil, fmt.Errorf("delete stale nftables table %q: %w", existingTable.Name, err)
		}
	}

	// define table
	ctx.table = conn.CreateTable(&nftables.Table{
		Name:   "proxy_gateway",
		Family: nftables.TableFamilyINet,
	})

	// define NAT chain
	ctx.prerouteNATChain = &nftables.Chain{
		Table:    ctx.table,
		Name:     "prerouting",
		Type:     nftables.ChainTypeNAT,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriorityNATDest,
	}
	conn.AddChain(ctx.prerouteNATChain)

	// define filter chain
	ctx.inputFilterChain = &nftables.Chain{
		Table:    ctx.table,
		Name:     "timeouts",
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookInput,
		Priority: nftables.ChainPriorityFilter,
	}
	conn.AddChain(ctx.inputFilterChain)

	// define lookup for ipv4
	ctx.lookup4 = &nftables.Set{
		Table:         ctx.table,
		Name:          "dnat4",
		IsMap:         true,
		Concatenation: true,
		KeyType: nftables.MustConcatSetType(
			nftables.TypeInetProto, nftables.TypeIPAddr, nftables.TypeInetService,
		),
		DataType: nftables.MustConcatSetType(
			nftables.TypeIPAddr, nftables.TypeInetService,
		),
	}
	if err := conn.AddSet(ctx.lookup4, nil); err != nil {
		return nil, fmt.Errorf("define nftables set %q: %w", ctx.lookup4.Name, err)
	}

	// define lookup for ipv6
	ctx.lookup6 = &nftables.Set{
		Table:         ctx.table,
		Name:          "dnat6",
		IsMap:         true,
		Concatenation: true,
		KeyType: nftables.MustConcatSetType(
			nftables.TypeInetProto, nftables.TypeIP6Addr, nftables.TypeInetService,
		),
		DataType: nftables.MustConcatSetType(
			nftables.TypeIP6Addr, nftables.TypeInetService,
		),
	}
	if err := conn.AddSet(ctx.lookup6, nil); err != nil {
		return nil, fmt.Errorf("define nftables set %q: %w", ctx.lookup6.Name, err)
	}

	// create table/chains/sets before adding rules that reference set IDs
	if err := conn.Flush(); err != nil {
		return nil, fmt.Errorf("create nftables table %q objects: %w", ctx.table.Name, err)
	}

	// define forwarding rules for ip4/ip6
	conn.AddRule(&nftables.Rule{
		Table: ctx.table,
		Chain: ctx.prerouteNATChain,
		Exprs: []expr.Any{
			// meta nfproto -> reg 1
			&expr.Meta{
				Key:      expr.MetaKeyNFPROTO,
				Register: 1,
			},
			// reg 1 == ipv4
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.NFPROTO_IPV4},
			},
			// meta l4proto -> reg 1
			&expr.Meta{
				Key:      expr.MetaKeyL4PROTO,
				Register: 1,
			},
			// load ip daddr -> reg 9
			&expr.Payload{
				DestRegister: 9,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       16,
				Len:          4,
			},
			// load th dport -> reg 10
			&expr.Payload{
				DestRegister: 10,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			// lookup key (l4proto . ip daddr . th dport) in @dnat4, write value to reg 1+
			&expr.Lookup{
				SourceRegister: 1,
				SetID:          ctx.lookup4.ID,
				DestRegister:   1,
				IsDestRegSet:   true,
				SetName:        ctx.lookup4.Name,
			},
			// dnat ip to <mapped addr> : <mapped port>
			&expr.NAT{
				Type:        expr.NATTypeDestNAT,
				Family:      unix.NFPROTO_IPV4,
				RegAddrMin:  1,
				RegProtoMin: 9,
				Specified:   true,
			},
		},
	})
	conn.AddRule(&nftables.Rule{
		Table: ctx.table,
		Chain: ctx.prerouteNATChain,
		Exprs: []expr.Any{
			// meta nfproto -> reg 1
			&expr.Meta{
				Key:      expr.MetaKeyNFPROTO,
				Register: 1,
			},
			// reg 1 == ipv6
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.NFPROTO_IPV6},
			},
			// meta l4proto -> reg 1
			&expr.Meta{
				Key:      expr.MetaKeyL4PROTO,
				Register: 1,
			},
			// load ip6 daddr -> reg 9
			&expr.Payload{
				DestRegister: 9,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       24,
				Len:          16,
			},
			// load th dport -> reg 13
			&expr.Payload{
				DestRegister: 13,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			// lookup key (l4proto . ip6 daddr . th dport) in @dnat6, write value to reg 1+
			&expr.Lookup{
				SourceRegister: 1,
				SetID:          ctx.lookup6.ID,
				DestRegister:   1,
				IsDestRegSet:   true,
				SetName:        ctx.lookup6.Name,
			},
			// dnat ip6 to <mapped addr> : <mapped port>
			&expr.NAT{
				Type:        expr.NATTypeDestNAT,
				Family:      unix.NFPROTO_IPV6,
				RegAddrMin:  1,
				RegProtoMin: 2,
				Specified:   true,
			},
		},
	})
	// submit changes
	if err := conn.Flush(); err != nil {
		return nil, fmt.Errorf("create nftables table %q rules: %w", ctx.table.Name, err)
	}
	return ctx, nil
}

func (ctx *NftablesContext) AddDNAT(protocol api.Protocol, matchAddress netip.AddrPort, targetAddress netip.AddrPort) error {
	if ctx == nil || ctx.conn == nil {
		return fmt.Errorf("netfilter context is not connected")
	}

	var lookup *nftables.Set
	if matchAddress.Addr().Is4() && targetAddress.Addr().Is4() {
		lookup = ctx.lookup4
	} else if matchAddress.Addr().Is6() && targetAddress.Addr().Is6() {
		lookup = ctx.lookup6
	} else {
		return fmt.Errorf("source and target must both be either IPv4 or IPv6")
	}

	element := nftables.SetElement{
		Key: encodeMapKey(protocol, matchAddress),
		Val: encodeMapValue(targetAddress),
	}
	if err := ctx.conn.SetAddElements(lookup, []nftables.SetElement{element}); err != nil {
		return fmt.Errorf("add map entry: %w", err)
	}
	_ = ctx.conn.Flush()
	return nil
}

func (ctx *NftablesContext) ClearDNAT(protocol api.Protocol, matchAddress netip.AddrPort) error {
	if ctx == nil || ctx.conn == nil {
		return fmt.Errorf("netfilter context is not connected")
	}

	var lookup *nftables.Set
	if matchAddress.Addr().Is4() {
		lookup = ctx.lookup4
	} else if matchAddress.Addr().Is6() {
		lookup = ctx.lookup6
	} else {
		return fmt.Errorf("address type must be either IPv4 or IPv6")
	}

	element := nftables.SetElement{
		Key: encodeMapKey(protocol, matchAddress),
	}

	if err := ctx.conn.SetDeleteElements(lookup, []nftables.SetElement{element}); err != nil {
		return fmt.Errorf("delete map entry: %w", err)
	}
	_ = ctx.conn.Flush()
	return nil
}

func (ctx *NftablesContext) SetTTL(name string, protocol api.Protocol, matchListen netip.AddrPort, ttl api.TTL) error {
	var matchAddressFamily uint8
	if matchListen.Addr().Is4() {
		matchAddressFamily = unix.NFPROTO_IPV4
	} else if matchListen.Addr().Is6() {
		matchAddressFamily = unix.NFPROTO_IPV6
	} else {
		return fmt.Errorf("address type must be IPv4 or IPv6")
	}

	ttlObj := &nftables.NamedObj{
		Table: ctx.table,
		Name:  name,
		Type:  nftables.ObjTypeCtTimeout,
		Obj: &expr.CtTimeout{
			L4Proto: uint8(protocol),
			Policy: expr.CtStatePolicyTimeout{
				expr.CtStateUDPUNREPLIED: ttl.Seconds(),
				expr.CtStateUDPREPLIED:   ttl.Seconds(),
			},
		},
	}

	var err error
	if _, present := ctx.ttls[ttlObj.Name]; present {
		// simply change the ttl value
		_, err = ctx.conn.ResetObject(ttlObj)
	} else {
		// match against the frontend and set timeout
		ctx.conn.AddObject(ttlObj)
		ruleExprs := []expr.Any{
			// meta l4proto -> reg 1
			&expr.Meta{
				Key:      expr.MetaKeyL4PROTO,
				Register: 1,
			},
			// reg 1 == configured protocol (tcp/udp)
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{uint8(protocol)},
			},
			// meta nfproto -> reg 1
			&expr.Meta{
				Key:      expr.MetaKeyNFPROTO,
				Register: 1,
			},
			// reg 1 == ipv4 or ipv6 family for this frontend
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{matchAddressFamily},
			},
		}
		if matchListen.Addr().Is4() {
			ruleExprs = append(ruleExprs,
				// load ip daddr -> reg 2
				&expr.Payload{
					DestRegister: 2,
					Base:         expr.PayloadBaseNetworkHeader,
					Offset:       16,
					Len:          4,
				},
				// reg 2 == frontend listen IPv4 address
				&expr.Cmp{
					Op:       expr.CmpOpEq,
					Register: 2,
					Data:     matchListen.Addr().AsSlice(),
				},
			)
		} else if matchListen.Addr().Is6() {
			ruleExprs = append(ruleExprs,
				// load ip6 daddr -> reg 2
				&expr.Payload{
					DestRegister: 2,
					Base:         expr.PayloadBaseNetworkHeader,
					Offset:       24,
					Len:          16,
				},
				// reg 2 == frontend listen IPv6 address
				&expr.Cmp{
					Op:       expr.CmpOpEq,
					Register: 2,
					Data:     matchListen.Addr().AsSlice(),
				},
			)
		}
		ruleExprs = append(ruleExprs,
			// load th dport -> reg 3
			&expr.Payload{
				DestRegister: 3,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			// reg 3 == frontend listen port
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 3,
				Data:     encodePort(matchListen.Port()),
			},
			// apply ct timeout policy object by name
			&expr.Objref{
				Type: int(nftables.ObjTypeCtTimeout),
				Name: ttlObj.Name,
			},
		)
		ctx.conn.AddRule(&nftables.Rule{
			Table: ctx.table,
			Chain: ctx.inputFilterChain,
			Exprs: ruleExprs,
		})
		ctx.ttls[ttlObj.Name] = struct{}{}
	}

	if err != nil {
		return fmt.Errorf("update ttl: %w", err)
	}
	if err = ctx.conn.Flush(); err != nil {
		return fmt.Errorf("update ttl: %w", err)
	}
	return nil
}

func (ctx *NftablesContext) Close() error {
	if ctx == nil || ctx.conn == nil {
		return nil
	}
	conn := ctx.conn
	ctx.conn = nil

	conn.DelTable(ctx.table)
	return conn.Flush()
}

func encodePort(port uint16) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, port)
	return buf
}

func encodeMapKey(protocol api.Protocol, address netip.AddrPort) []byte {
	addr := address.Addr().AsSlice()
	buf := make([]byte, 0, 1+len(addr)+2)
	buf = append(buf, uint8(protocol))
	buf = append(buf, addr...)
	buf = append(buf, encodePort(address.Port())...)
	return buf
}

func encodeMapValue(address netip.AddrPort) []byte {
	addr := address.Addr().AsSlice()
	buf := make([]byte, 0, len(addr)+2)
	buf = append(buf, addr...)
	buf = append(buf, encodePort(address.Port())...)
	return buf
}
