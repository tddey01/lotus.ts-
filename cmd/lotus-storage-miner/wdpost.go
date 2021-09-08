package main

import (
	"context"
	"fmt"
	"github.com/urfave/cli/v2"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-statestore"
	lcli "github.com/filecoin-project/lotus/cli"
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
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
)

var windowpostCmd = &cli.Command{
	Name:      "wdpost",
	Usage:     "windowpost测试 窗口编号（0~47）",
	ArgsUsage: "<deadlineIdx>",
	Action:    testWindowPost,
}

func testWindowPost(cctx *cli.Context) error {
	dlIdx, err := strconv.ParseUint(cctx.Args().Get(0), 10, 64)
	if err != nil {
		return xerrors.Errorf("could not parse deadline index: %w", err)
	}
	//var closer func()
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
	//defer closer()
	wdpath := os.Getenv("wdpath")
	if wdpath==""{
		wdpath = "datastore1"
	}
	repo.SetFsDatastore(wdpath)

	ctx := lcli.ReqContext(cctx)
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	api,close,err := lcli.GetFullNodeAPIV1(cctx)
	if err != nil {
		log.Errorf("云构---------WorkerWindowPost-----------",err)
		return err
	}
	//api,close,err := lcli.GetStorageMinerAPI(cctx)
	//if err != nil {
	//	log.Errorf("云构---------WorkerWindowPost-----------",err)
	//	return err
	//}
	defer close()

	var addr address.Address
	actor := os.Getenv("actor")
	if actor==""{
		log.Errorf("云构 需要添加环境变量 export actor=[矿工ID]",err)
		return err
	}
	addr.Scan(actor)

	fc := config.DefaultStorageMiner().Fees

	fmt.Println("--------WorkerWindowPost-------:",fc.MaxPreCommitGasFee,fc.MaxWindowPoStGasFee,fc.MaxCommitGasFee)

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
	wsts := statestore.New(namespace.Wrap(mds, modules.WorkerCallsPrefix))
	smsts := statestore.New(namespace.Wrap(mds, modules.ManagerWorkPrefix))

	sa, err := modules.StorageAuth(ctx, api)
	if err != nil {
		return xerrors.Errorf("yungo----------------------ActorAddress error-----------------------:", err)
	}

	smgr, err := sectorstorage.New(ctx, lr, stores.NewIndex(),sectorstorage.SealerConfig{
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
	sign := make(chan os.Signal)
	signal.Notify(sign, syscall.SIGINT, syscall.SIGTERM,syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		<-sign
		runtime.Goexit()
	}()
	fps.TestRun(ctx,dlIdx,storage.Wpost)
	return nil
}

