//Package dag implements:
//- consensus functionality
//- finalizing process
//- block creation process

package dag

import (
	"fmt"
	"sync/atomic"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/consensus"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/creator"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/finalizer"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/headsync"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/slotticker"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/downloader"
	"gitlab.waterfall.network/waterfall/protocol/gwat/event"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
)

// Backend wraps all methods required for block creation.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	Downloader() *downloader.Downloader
	Etherbase() (eb common.Address, err error)
	SetEtherbase(etherbase common.Address)
	CreatorAuthorize(creator common.Address) error
	IsDevMode() bool
}

type Dag struct {
	chainConfig *params.ChainConfig

	// events
	mux *event.TypeMux

	consensusInfo     *types.ConsensusInfo
	consensusInfoFeed event.Feed

	eth Backend
	bc  *core.BlockChain

	//creator
	creator *creator.Creator
	//finalizer
	finalizer *finalizer.Finalizer
	//headsync
	headsync *headsync.Headsync

	busy int32

	exitChan chan struct{}
	errChan  chan error
}

// New creates new instance of Dag
func New(eth Backend, chainConfig *params.ChainConfig, mux *event.TypeMux, creatorConfig *creator.Config, engine consensus.Engine) *Dag {
	fin := finalizer.New(chainConfig, eth, mux)
	d := &Dag{
		chainConfig: chainConfig,
		eth:         eth,
		mux:         mux,
		bc:          eth.BlockChain(),
		creator:     creator.New(creatorConfig, chainConfig, engine, eth, mux),
		finalizer:   fin,
		headsync:    headsync.New(chainConfig, eth, mux, fin),
		exitChan:    make(chan struct{}),
		errChan:     make(chan error),
	}
	atomic.StoreInt32(&d.busy, 0)
	return d
}

// Creator get current creator
func (d *Dag) Creator() *creator.Creator {
	return d.creator
}

// HandleFinalize handles consensus data
// 1. blocks finalization
func (d *Dag) HandleFinalize(data *types.FinalizationParams) *types.FinalizationResult {
	//skip if synchronising
	if d.eth.Downloader().Synchronising() {
		errStr := creator.ErrSynchronization.Error()
		return &types.FinalizationResult{
			Error: &errStr,
		}
	}

	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()

	if data.BaseSpine != nil {
		log.Info("Handle Finalize: start", "baseSpine", (*data.BaseSpine).Hex(), "spines", data.Spines, "\u2692", params.BuildId)
	} else {
		log.Info("Handle Finalize: start", "baseSpine", nil, "spines", data.Spines, "\u2692", params.BuildId)
	}
	res := &types.FinalizationResult{
		Error: nil,
	}
	// finalization
	if len(data.Spines) > 0 {
		if err := d.finalizer.Finalize(&data.Spines, data.BaseSpine, false); err != nil {
			e := err.Error()
			res.Error = &e
		}
	}
	lfHeader := d.bc.GetLastFinalizedHeader()
	if lfHeader.Height != lfHeader.Nr() {
		err := fmt.Sprintf("☠ bad last finalized block: mismatch nr=%d and height=%d", lfHeader.Nr(), lfHeader.Height)
		if res.Error == nil {
			res.Error = &err
		} else {
			mrg := fmt.Sprintf("error[0]=%s\nerror[1]: %s", *res.Error, err)
			res.Error = &mrg
		}
	} else {
		d.bc.WriteLastCoordinatedHash(lfHeader.Hash())
	}
	lfHash := lfHeader.Hash()
	res.LFSpine = &lfHash

	log.Info("Handle Finalize: response", "result", res)
	return res
}

// HandleGetCandidates collect next finalization candidates
func (d *Dag) HandleGetCandidates(slot uint64) *types.CandidatesResult {
	//skip if synchronising
	if d.eth.Downloader().Synchronising() {
		errStr := creator.ErrSynchronization.Error()
		return &types.CandidatesResult{
			Error: &errStr,
		}
	}

	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()

	tstart := time.Now()

	// collect next finalization candidates
	candidates, err := d.finalizer.GetFinalizingCandidates(&slot)
	if len(candidates) == 0 {
		log.Info("No candidates for tips", "tips", d.bc.GetTips().Print())
	}
	log.Info("Handle GetCandidates: get finalizing candidates", "err", err, "candidates", candidates, "elapsed", common.PrettyDuration(time.Since(tstart)), "\u2692", params.BuildId)
	res := &types.CandidatesResult{
		Error:      nil,
		Candidates: candidates,
	}
	if err != nil {
		estr := err.Error()
		res.Error = &estr
	}
	log.Info("Handle GetCandidates: response", "result", res, "\u2692", params.BuildId)
	return res
}

// HandleHeadSyncReady set initial state to start head sync with coordinating network.
func (d *Dag) HandleHeadSyncReady(checkpoint *types.ConsensusInfo) (bool, error) {
	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()
	log.Info("Handle Head Sync Ready", "checkpoint", checkpoint)
	return d.headsync.SetReadyState(checkpoint)
}

// HandleSyncSlotInfo set initial state to start head sync with coordinating network.
func (d *Dag) HandleSyncSlotInfo(slotInfo types.SlotInfo) (bool, error) {
	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()
	log.Info("Handle Sync Slot info", "params", slotInfo)
	si := d.bc.GetSlotInfo()
	if si.GenesisTime == slotInfo.GenesisTime &&
		si.SecondsPerSlot == slotInfo.SecondsPerSlot &&
		si.SlotsPerEpoch == slotInfo.SlotsPerEpoch {
		return true, nil
	}
	if d.eth.IsDevMode() {
		err := d.bc.SetSlotInfo(&slotInfo)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// HandleHeadSync run head sync with coordinating network.
func (d *Dag) HandleHeadSync(data []types.ConsensusInfo) (bool, error) {
	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()
	log.Info("Handle Head Sync", "len(data)", len(data), "data", data)
	return d.headsync.Sync(data)
}

// HandleValidateSpines collect next finalization candidates
func (d *Dag) HandleValidateSpines(spines common.HashArray) (bool, error) {
	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()
	log.Info("Handle Validate Spines", "spines", spines, "\u2692", params.BuildId)
	return d.finalizer.IsValidSequenceOfSpines(spines)
}

// GetConsensusInfo returns the last info received from the consensus network
func (d *Dag) GetConsensusInfo() *types.ConsensusInfo {
	if d.consensusInfo == nil {
		return nil
	}
	return d.consensusInfo.Copy()
}

// SubscribeConsensusInfoEvent registers a subscription for consensusInfo updated event
func (d *Dag) SubscribeConsensusInfoEvent(ch chan<- types.Tips) event.Subscription {
	return d.consensusInfoFeed.Subscribe(ch)
}

func (d *Dag) StartWork(accounts []common.Address) {
	startTicker := time.NewTicker(500 * time.Millisecond)

	tickSec := 0
	for {
		currentTime := time.Now()
		genesisTime := time.Unix(int64(d.bc.GetSlotInfo().GenesisTime), 0)

		if currentTime.Before(genesisTime) {
			if tickSec != currentTime.Second() && currentTime.Second()%5 == 0 {
				timeRemaining := genesisTime.Sub(currentTime)
				log.Info("Time before start", "hour", timeRemaining.Truncate(time.Second))
			}
		} else {
			log.Info("Chain genesis time reached")
			startTicker.Stop()
			go d.workLoop(accounts)
		}
		tickSec = currentTime.Second()

		select {
		case <-d.exitChan:
			close(d.exitChan)
			close(d.errChan)
			startTicker.Stop()
			return
		case err := <-d.errChan:
			close(d.errChan)
			close(d.exitChan)
			startTicker.Stop()
			log.Error("dag worker has error", "error", err)
			return
		case <-startTicker.C:
		}
	}
}

func (d *Dag) workLoop(accounts []common.Address) {
	secPerSlot := d.bc.GetSlotInfo().SecondsPerSlot
	genesisTime := time.Unix(int64(d.bc.GetSlotInfo().GenesisTime), 0)
	slotTicker := slotticker.NewSlotTicker(genesisTime, secPerSlot)

	for {
		select {
		case <-d.exitChan:
			close(d.exitChan)
			close(d.errChan)
			slotTicker.Done()
			return
		case err := <-d.errChan:
			close(d.errChan)
			close(d.exitChan)
			slotTicker.Done()
			log.Error("dag worker has error", "error", err)
			return
		case slot := <-slotTicker.C():
			if slot == 0 {
				era := types.NewEra(0, d.bc.Config().EpochsPerEra)
				d.bc.SetNewEraInfo(0, era)
				continue
			}
			var (
				err      error
				creators []common.Address
			)

			log.Info("New slot", "slot", slot)

			d.handleEra(slot)

			// TODO: uncomment this code for subnetwork support, add subnet and get it to the creators getter (line 253)
			//if d.bc.Config().IsForkSlotSubNet1(currentSlot) {
			//	creators, err = d.bc.ValidatorStorage().GetCreatorsBySlot(d.bc, currentSlot,subnet)
			//	if err != nil {
			//		d.errChan <- err
			//	}
			//} else {}
			// TODO: move it to else condition

			creators, err = d.bc.ValidatorStorage().GetCreatorsBySlot(d.bc, slot)
			if err != nil {
				d.errChan <- err
			}

			d.work(slot, creators, accounts)
		}
	}
}

func (d *Dag) work(slot uint64, creators, accounts []common.Address) {
	//skip if synchronising
	if d.eth.Downloader().Synchronising() {
		d.errChan <- creator.ErrSynchronization
	}

	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()

	errs := map[string]string{}

	var (
		err error
	)
	// create block
	tips := d.bc.GetTips()
	dagSlots := d.countDagSlots(&tips)
	log.Info("Creator processing: create condition",
		"condition", d.creator.IsRunning() && len(errs) == 0 && dagSlots != -1 && dagSlots <= finalizer.CreateDagSlotsLimit,
		"IsRunning", d.creator.IsRunning(),
		"errs", errs,
		"dagSlots", dagSlots,
	)

	if d.creator.IsRunning() && len(errs) == 0 && dagSlots != -1 && dagSlots <= finalizer.CreateDagSlotsLimit {
		assigned := &creator.Assignment{
			Slot:     slot,
			Creators: creators,
		}

		crtStart := time.Now()
		crtInfo := map[string]string{}
		for _, creator := range assigned.Creators {
			// if received next slot

			// TODO: may be drop consensusInfo field from blockchain because we don`t write value to it
			if d.consensusInfo != nil && d.consensusInfo.Slot > assigned.Slot {
				break
			}

			//d.eth.BlockChain().DagMu.Lock()
			//defer d.eth.BlockChain().DagMu.Unlock()

			coinbase := common.Address{}
			for _, acc := range accounts {
				if creator == acc {
					coinbase = creator
					break
				}
			}
			if coinbase == (common.Address{}) {
				return
			}

			d.eth.SetEtherbase(coinbase)
			if err = d.eth.CreatorAuthorize(coinbase); err != nil {
				log.Error("Creator authorize err", "err", err, "creator", coinbase)
				return
			}
			log.Info("Creator assigned", "creator", coinbase)

			block, crtErr := d.creator.CreateBlock(assigned, &tips)
			if crtErr != nil {
				crtInfo["error"] = crtErr.Error()
			}

			if block != nil {
				crtInfo["newBlock"] = block.Hash().Hex()
			}
			log.Info("Creator processing: create block", "dagSlots", dagSlots, "IsRunning", d.creator.IsRunning(), "crtInfo", crtInfo, "elapsed", common.PrettyDuration(time.Since(crtStart)))
		}
	}
}

// countDagSlots count number of slots in dag chain
// if error returns  -1
func (d *Dag) countDagSlots(tips *types.Tips) int {
	candidates, err := d.finalizer.GetFinalizingCandidates(nil)
	if err != nil {
		return -1
	}
	return len(candidates)
}

// handleEra
func (d *Dag) handleEra(slot uint64) {
	d.bc.HandleEra(slot)
}
