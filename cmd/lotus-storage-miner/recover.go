package main

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-statestore"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
	sectorstorage "github.com/filecoin-project/lotus/extern/sector-storage"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/extern/sector-storage/stores"
	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/filecoin-project/lotus/storage"
	"github.com/ipfs/go-datastore/namespace"
	"golang.org/x/xerrors"
	"os"
	"strconv"
)

var recoverCmd = &cli.Command{
	Name:   "recover",
	Usage:  "恢复扇区 窗口编号（0~47）",
	Action: failureRecovery,
}

func failureRecovery(cctx *cli.Context) error {
	dlIdx, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
	if err != nil {
		return xerrors.Errorf("could not parse deadline index: %w", err)
	}
	//var closer func()

	//defer closer()

	//var nodeApi api.StorageMiner
	//for {
	//	nodeApi, closer, err = lcli.GetStorageMinerAPI(cctx)
	//	if err == nil {
	//		break
	//	}
	//	fmt.Printf("\r\x1b[0KConnecting to miner API... (%s)", err)
	//	time.Sleep(time.Second)
	//	continue
	//}
	repo.SetFsDatastore("datastore1")
	ctx := lcli.ReqContext(cctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	api,close,err := lcli.GetFullNodeAPIV1(cctx)
	if err != nil {
		log.Errorf("云构---------WorkerWindowPost-----------",err)
		return err
	}
	defer close()

	var addr address.Address
	actor := os.Getenv("actor")
	if actor==""{
		log.Errorf("云构 需要添加环境变量 export actor=[矿工ID]",err)
		return err
	}
	addr.Scan(actor)

	worker,err := api.WalletDefaultAddress(ctx)
	if err != nil {
		log.Errorf("云构---------WorkerWindowPost-----------",err)
		return err
	}

	fc := config.DefaultStorageMiner().Fees

	fmt.Println("--------WorkerWindowPost-------:",fc.MaxPreCommitGasFee,fc.MaxWindowPoStGasFee,fc.MaxCommitGasFee)
	fmt.Println("--------WorkerWindowPost-------:",addr,worker)

	repoPath := cctx.String(FlagMinerRepo)
	r,err := repo.NewFS(repoPath)
	if err != nil {
		return err
	}
	lr, err := r.NoLock(repo.StorageMiner)
	if err != nil {
		return err
	}
	mds, err := lr.Datastore(ctx,"/metadata")
	if err != nil {
		return err
	}

	sa, err := modules.StorageAuth(ctx, api)
	if err != nil {
		return xerrors.Errorf("yungo----------------------ActorAddress error-----------------------:", err)
	}

	wsts := statestore.New(namespace.Wrap(mds, modules.WorkerCallsPrefix))
	smsts := statestore.New(namespace.Wrap(mds, modules.ManagerWorkPrefix))

	smgr, err := sectorstorage.New(ctx, lr, stores.NewIndex(), sectorstorage.SealerConfig{
		ParallelFetchLimit: 10,
		AllowAddPiece:      true,
		AllowPreCommit1:    true,
		AllowPreCommit2:    true,
		AllowCommit:        true,
		AllowUnseal:        true,
	}, nil, sa,wsts,smsts)
	ac := new(storage.AddressSelector)
	j:=journal.NilJournal()

	fps, err := storage.NewWindowedPoStScheduler(api, fc,ac, smgr,ffiwrapper.ProofVerifier,smgr,j, addr)
	if err != nil {
		return err
	}
	fps.TestRun(ctx,dlIdx,storage.Recoveries)

	return nil
}

