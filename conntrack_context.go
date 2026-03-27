package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ti-mo/conntrack"
	"golang.org/x/sync/errgroup"
)

const conntrackWatchdogPeriod = 60 * time.Second

type ConntrackContext struct {
	conn     *conntrack.Conn
	lastSeen map[*TargetContext]*time.Time
}

func DialConntrack(rt *RuntimeContext) (*ConntrackContext, error) {
	conn, err := conntrack.Dial(nil)
	if err != nil {
		return nil, err
	}
	ctx := &ConntrackContext{
		conn:     conn,
		lastSeen: make(map[*TargetContext]*time.Time),
	}
	now := time.Now()
	for _, t := range rt.Targets {
		lastSeen := new(time.Time)
		*lastSeen = now
		ctx.lastSeen[t] = lastSeen
	}
	return ctx, nil
}

func (ctx *ConntrackContext) Start(group *errgroup.Group, groupCtx context.Context) {
	groupCtx, cancel := context.WithCancel(groupCtx)
	ticker := time.NewTicker(conntrackWatchdogPeriod)

	group.Go(func() error {
		defer cancel()
		defer ticker.Stop()
		defer func() {
			log.Printf("[conntrack] watchdog stopped")
		}()
		log.Printf("[conntrack] watchdog started period=%s", conntrackWatchdogPeriod)

		filter := conntrack.NewFilter().Status(conntrack.StatusDstNATDone)
		var parallelGroup sync.WaitGroup

		for {
			select {
			case <-groupCtx.Done():
				return nil
			case tickStarted := <-ticker.C:
				log.Printf("[conntrack] filter start")
				flows, err := ctx.conn.DumpFilter(filter, nil)
				if err != nil {
					log.Printf("[conntrack] filter error: %v", err)
					continue
				}

				// scan flows in parallel
				for target, lastSeen := range ctx.lastSeen {
					parallelGroup.Go(func() {
						// mark target as recently active
						for i := range flows {
							flow := &flows[i]
							if targetMatchesFlow(target, flow) {
								*lastSeen = tickStarted
								return
							}
						}
						// if we didn't see a flow, check if we timed out
						if tickStarted.Sub(*lastSeen) >= target.idleTimeout.ToDuration() {
							_ = target.Deactivate()
						}
					})

				}
				parallelGroup.Wait()
				log.Printf("[conntrack] filtered %d flows in %s", len(flows), time.Since(tickStarted))
			}
		}
	})
}

func targetMatchesFlow(target *TargetContext, flow *conntrack.Flow) bool {
	for _, service := range target.services {
		if serviceMatchesFlow(service, flow) {
			return true
		}
	}
	return false
}

func serviceMatchesFlow(service *TargetServiceContext, flow *conntrack.Flow) bool {
	return tupleMatchesService(flow.TupleOrig, service) || tupleMatchesService(flow.TupleReply, service)
}

func tupleMatchesService(tuple conntrack.Tuple, service *TargetServiceContext) bool {
	if tuple.Proto.Protocol != uint8(service.protocol) {
		return false
	}
	dst := service.dst
	// Match either side so reply direction still counts as activity for the service.
	if tuple.IP.DestinationAddress == dst.Addr() && tuple.Proto.DestinationPort == dst.Port() {
		return true
	}
	if tuple.IP.SourceAddress == dst.Addr() && tuple.Proto.SourcePort == dst.Port() {
		return true
	}
	return false
}
