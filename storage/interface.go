package storage

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/ethclient"
)

var extrainfoDatabase map[common.Hash]*extrainfo

func init() {
	extrainfoDatabase = make(map[common.Hash]*extrainfo)
	extrainfoDatabase[common.HexToHash("5e31ddf9f93f71ee232ca25885f39fa8b669f51013c9c01db5e7a2772ef7d69f")] = &extrainfo{
		RegistryContractAddress: common.HexToAddress("0x6D7030A63DEA0C9BbF32D76Cf6B03f93E818CD8F"),
		AliasContractAddress:    common.HexToAddress("0x5a76921D67b863E7d6c77ACfe1307D2ACEdd9007"),
	}
}

type StateInformation struct {
	CurrentBlock int64
	CurrentTime  int64
}
type TransactionInfo struct {
	Successful  bool
	BlockNumber int64
	Pending     bool
}
type extrainfo struct {
	RegistryContractAddress common.Address
	AliasContractAddress    common.Address
}
type Storage interface {
	GetStateInformation(ctx context.Context) (*StateInformation, error)
	RetrieveEntity(ctx context.Context, VK []byte) ([]byte, *StateInformation, error)
	RetrieveEntityField(ctx context.Context, VK []byte, index int) ([]byte, *StateInformation, error)
	RetrieveDOTByVKIndex(ctx context.Context, DstVK []byte, index int) (*DOTRegistration, *StateInformation, error)
	RetrieveDOTByHash(ctx context.Context, Hash []byte, Location int) (*DOTRegistration, *StateInformation, error)
	ResolvePartialAlias(ctx context.Context, domain [32]byte, tld [32]byte) (*AliasRegistration, *StateInformation, error)
	ResolveAlias(ctx context.Context, subdomain [32]byte, domain [32]byte, tld [32]byte) (*AliasRegistration, *StateInformation, error)
	InsertEntity(ctx context.Context, controller common.Address, VK []byte, revokable bool, data []byte, signFn SignerFn) (*Transaction, error)
	InsertDOTOnChain(ctx context.Context, account common.Address, DstVK []byte, data []byte, signFn SignerFn) (*Transaction, error)
	InsertDOTOffChain(ctx context.Context, account common.Address, DstVK []byte, hash []byte, location uint64, signFn SignerFn) (*Transaction, error)
	CreateAlias(ctx context.Context, controller common.Address, subdomain [32]byte, domain [32]byte, tld [32]byte, value []byte, signFn SignerFn) (*Transaction, error)
	CreateTLD(ctx context.Context, controller common.Address, tld [32]byte, signFn SignerFn) (*Transaction, error)
	TransactionInfo(ctx context.Context, hash []byte) (*TransactionInfo, error)
}

type EthereumStorage struct {
	cl          *ethclient.Client
	ei          *extrainfo
	currentHead *big.Int
	currentTime *big.Int
	mu          sync.Mutex
	regTrans    *RegistryAPITransactor
	aliasTrans  *AliasAPITransactor
}
type EntityRegistration struct {
	VK   []byte
	Addr common.Address
	Data []byte
}
type DOTRegistration struct {
	Hash     []byte
	DstVK    []byte
	MaxIndex int
	Index    int
	Data     []byte
}
type AliasRegistration struct {
	//What is the latest subdomain for this domain
	Head [32]byte
	//What subdomain is actually resolved
	Subdomain [32]byte
	Domain    [32]byte
	TLD       [32]byte
	//The value of the subdomain
	Value []byte
}

func NewEthereumStorage(ctx context.Context, ipcaddr string) (*EthereumStorage, error) {
	client, err := ethclient.Dial(ipcaddr)
	if err != nil {
		return nil, err
	}
	genesis, err := client.BlockByNumber(ctx, big.NewInt(0))
	if err != nil {
		return nil, err
	}
	extrainfo, ok := extrainfoDatabase[genesis.Hash()]
	if !ok {
		fmt.Printf("our hash was %x\n", common.HexToHash("5e31ddf9f93f71ee232ca25885f39fa8b669f51013c9c01db5e7a2772ef7d69f"))
		return nil, fmt.Errorf("Block chain with genesis %x is not in the database, cannot use as storage backend", genesis.Hash())

	}
	rv := &EthereumStorage{
		cl: client,
		ei: extrainfo,
	}
	rv.currentHead = big.NewInt(0)
	go rv.updateHead()

	// NewRegistryAPITransactor creates a new write-only instance of RegistryAPI, bound to a specific deployed contract.
	regTrans, err := NewRegistryAPITransactor(extrainfo.RegistryContractAddress, client)
	if err != nil {
		return nil, err
	}
	rv.regTrans = regTrans

	aliasTrans, err := NewAliasAPITransactor(extrainfo.AliasContractAddress, client)
	if err != nil {
		return nil, err
	}
	rv.aliasTrans = aliasTrans
	return rv, nil
}
func (es *EthereumStorage) updateHead() error {
	rch := make(chan *types.Header)
	_, err := es.cl.SubscribeNewHead(context.Background(), rch)
	if err != nil {
		return err
	}
	for h := range rch {
		es.mu.Lock()
		es.currentHead = h.Number
		es.currentTime = h.Time
		es.mu.Unlock()
	}
	return fmt.Errorf("UpdateHead channel ended")
}

func (es *EthereumStorage) getHead() (*big.Int, *big.Int) {
	es.mu.Lock()
	n := es.currentHead
	t := es.currentTime
	es.mu.Unlock()
	return n, t
}

func (es *EthereumStorage) GetStateInformation(ctx context.Context) (*StateInformation, error) {
	rv := &StateInformation{}
	n, t := es.getHead()
	if n == nil || t == nil {
		rv.CurrentBlock = 0
		rv.CurrentTime = 0
	} else {
		rv.CurrentBlock = n.Int64()
		rv.CurrentTime = t.Int64()
	}
	return rv, nil
}

func (es *EthereumStorage) RetrieveEntity(ctx context.Context, VK []byte) (*EntityRegistration, *StateInformation, error) {
	blockN, blockT := es.getHead()
	if blockN == nil || blockT == nil {
		return nil, nil, fmt.Errorf("Still synchronizing to the chain")
	}
	hsh := sha3.NewKeccak256()
	//The key to the map
	hsh.Write(VK)
	//It's the first slot
	hsh.Write(make([]byte, 32))
	sum := hsh.Sum(nil)
	addressblob, err := es.cl.StorageAt(ctx, es.ei.RegistryContractAddress, common.BytesToHash(sum), blockN)
	if err != nil {
		panic(err)
	}
	rv := EntityRegistration{}
	copy(rv.Addr[:], addressblob[12:32])
	bigval := big.NewInt(0)
	bigval.SetBytes(sum)
	bigval.Add(bigval, big.NewInt(1))
	headerAddr := common.BigToHash(bigval)
	contentsHeader, err := es.cl.StorageAt(ctx, es.ei.RegistryContractAddress, headerAddr, blockN)
	chint := bigval.SetBytes(contentsHeader)
	arrlen := chint.Int64() / 2
	slots := arrlen / 32
	hsh = sha3.NewKeccak256()
	hsh.Write(headerAddr[:])
	bigval.SetBytes(hsh.Sum(nil))
	rvbytes := []byte{}
	//include last half slot
	for i := 0; i <= int(slots); i++ {
		slotcontents, err := es.cl.StorageAt(ctx, es.ei.RegistryContractAddress, common.BigToHash(bigval), blockN)
		if err != nil {
			panic(err)
		}
		bigval.Add(bigval, big.NewInt(1))
		rvbytes = append(rvbytes, slotcontents...)
	}
	rvbytes = rvbytes[:arrlen]
	rv.VK = make([]byte, 32)
	copy(rv.VK[:], VK[:])
	rv.Data = rvbytes
	si := &StateInformation{
		CurrentBlock: blockN.Int64(),
		CurrentTime:  blockT.Int64(),
	}
	return &rv, si, nil
}
func (es *EthereumStorage) RetrieveEntityField(ctx context.Context, VK []byte, index int) ([]byte, *StateInformation, error) {
	panic("not implemented")
	return nil, nil, nil
}
func (es *EthereumStorage) hashDot(dat []byte) []byte {
	hsh := sha3.NewKeccak256()
	//The key to the map
	hsh.Write(dat)
	return hsh.Sum(nil)
}
func (es *EthereumStorage) RetrieveDOTByVKIndex(ctx context.Context, DstVK []byte, index int) (*DOTRegistration, *StateInformation, error) {
	blockN, blockT := es.getHead()
	if blockN == nil || blockT == nil {
		return nil, nil, fmt.Errorf("Still synchronizing to the chain")
	}
	return es.retrieveDOTByVKIndex(ctx, DstVK, index, blockN, blockT)
}

func (es *EthereumStorage) retrieveDOTByVKIndex(ctx context.Context, DstVK []byte, index int, blockN *big.Int, blockT *big.Int) (*DOTRegistration, *StateInformation, error) {
	si := &StateInformation{
		CurrentBlock: blockN.Int64(),
		CurrentTime:  blockT.Int64(),
	}
	hsh := sha3.NewKeccak256()
	//The key to the map
	hsh.Write(DstVK)
	slot := make([]byte, 32)
	slot[31] = 4
	hsh.Write(slot)
	//The start of the array
	mainSlot := hsh.Sum(nil)
	//The length of the array
	arrayLenBytes, err := es.cl.StorageAt(ctx, es.ei.RegistryContractAddress, common.BytesToHash(mainSlot), blockN)
	if err != nil {
		panic(err)
	}
	arrayLen := new(big.Int).SetBytes(arrayLenBytes).Int64()
	fmt.Printf("total dot array length is %d\n", arrayLen)
	if index >= int(arrayLen) {
		return nil, si, nil
	}
	rv := DOTRegistration{}
	rv.DstVK = make([]byte, 32)
	copy(rv.DstVK, DstVK)
	hsh = sha3.NewKeccak256()
	//The key to the map
	hsh.Write(mainSlot)
	baseAddrHash := hsh.Sum(nil)
	fmt.Printf("array base slot: %x\n", baseAddrHash)
	thisEntryBase := new(big.Int).SetBytes(baseAddrHash)
	thisEntryBase.Add(thisEntryBase, big.NewInt(int64(index)))
	entryBaseHash := common.BigToHash(thisEntryBase)
	thisEntryLengthBytes, err := es.cl.StorageAt(ctx, es.ei.RegistryContractAddress, entryBaseHash, blockN)
	if err != nil {
		panic(err)
	}
	thisEntryLen := new(big.Int).SetBytes(thisEntryLengthBytes).Int64() / 2
	if thisEntryLen == 0 {
		fmt.Printf("unexpected zero length DOT\n")
		return nil, si, nil
	}
	hsh = sha3.NewKeccak256()
	hsh.Write(entryBaseHash[:])
	entryContentBase := hsh.Sum(nil)
	entryContentInt := new(big.Int).SetBytes(entryContentBase)
	entryContent := []byte{}
	for i := 0; i <= int(thisEntryLen); i++ {
		slotcontents, err := es.cl.StorageAt(ctx, es.ei.RegistryContractAddress, common.BigToHash(entryContentInt), blockN)
		if err != nil {
			panic(err)
		}
		entryContentInt.Add(entryContentInt, big.NewInt(1))
		entryContent = append(entryContent, slotcontents...)
	}
	rv.Data = entryContent[:thisEntryLen]
	rv.Hash = es.hashDot(rv.Data)
	rv.Index = index
	rv.MaxIndex = int(arrayLen)

	return &rv, si, nil
}

func (es *EthereumStorage) RetrieveDOTByHash(ctx context.Context, Hash []byte) (*DOTRegistration, *StateInformation, error) {
	blockN, blockT := es.getHead()
	if blockN == nil || blockT == nil {
		return nil, nil, fmt.Errorf("Still synchronizing to the chain")
	}
	si := &StateInformation{
		CurrentBlock: blockN.Int64(),
		CurrentTime:  blockT.Int64(),
	}
	hsh := sha3.NewKeccak256()
	//The key to the map
	hsh.Write(Hash)
	slot := make([]byte, 32)
	slot[31] = 5
	hsh.Write(slot)
	//The start of the struct
	baseAddr := hsh.Sum(nil)
	DstVKBytes, err := es.cl.StorageAt(ctx, es.ei.RegistryContractAddress, common.BytesToHash(baseAddr), blockN)
	if err != nil {
		panic(err)
	}
	indexAddr := new(big.Int).SetBytes(baseAddr)
	indexAddr.Add(indexAddr, big.NewInt(1))
	IndexBytes, err := es.cl.StorageAt(ctx, es.ei.RegistryContractAddress, common.BigToHash(indexAddr), blockN)
	if err != nil {
		panic(err)
	}
	index := new(big.Int).SetBytes(IndexBytes).Int64()
	//DOT doesn't exist
	if bytes.Equal(DstVKBytes, make([]byte, 32)) {
		return nil, si, nil
	}
	//=== same procedure
	return es.retrieveDOTByVKIndex(ctx, DstVKBytes, int(index), blockN, blockT)
}

type SignerFn = bind.SignerFn
type Transaction = types.Transaction

func (es *EthereumStorage) InsertEntity(ctx context.Context, controller common.Address, VK []byte, revokable bool, data []byte, signFn SignerFn) (*Transaction, error) {
	topts := bind.TransactOpts{
		From:    controller,
		Signer:  signFn,
		Context: ctx,
	}

	tx, err := es.regTrans.RegisterEntity(&topts, common.BytesToHash(VK), revokable, data)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (es *EthereumStorage) InsertDOT(ctx context.Context, account common.Address, DstVK []byte, data []byte, signFn SignerFn) (*Transaction, error) {
	topts := bind.TransactOpts{
		From:    account,
		Signer:  signFn,
		Context: ctx,
	}

	tx, err := es.regTrans.RegisterDot(&topts, common.BytesToHash(DstVK), data)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (es *EthereumStorage) CreateAlias(ctx context.Context, account common.Address, tld [32]byte, domain [32]byte, subdomain [32]byte, value []byte, signFn SignerFn) (*Transaction, error) {
	topts := bind.TransactOpts{
		From:    account,
		Signer:  signFn,
		Context: ctx,
	}
	tx, err := es.aliasTrans.CreateSubdomain(&topts, tld, domain, subdomain, value)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (es *EthereumStorage) TransactionInfo(ctx context.Context, hash []byte) (*TransactionInfo, error) {
	_, pending, err := es.cl.TransactionByHash(ctx, common.BytesToHash(hash))
	if err != nil {
		return nil, err
	}
	rv := &TransactionInfo{}
	rv.Pending = pending
	if !pending {
		receipt, err := es.cl.TransactionReceipt(ctx, common.BytesToHash(hash))
		if err != nil {
			return nil, err
		}
		if receipt == nil {
			rv.Successful = false
		} else {
			rv.Successful = true
		}
	}
	return rv, nil
}

func (es *EthereumStorage) ResolvePartialAlias(ctx context.Context, domain [32]byte, tld [32]byte) (*AliasRegistration, *StateInformation, error) {
	blockN, blockT := es.getHead()
	if blockN == nil || blockT == nil {
		return nil, nil, fmt.Errorf("Still synchronizing to the chain")
	}
	//Looks up the head
	hsh := sha3.NewKeccak256()
	hsh.Write(tld[:])
	hsh.Write(make([]byte, 32))
	tldobject := hsh.Sum(nil)
	domainMap := new(big.Int).SetBytes(tldobject)
	domainMap.Add(domainMap, big.NewInt(1))
	domainMapHash := common.BigToHash(domainMap)
	hsh = sha3.NewKeccak256()
	hsh.Write(domain[:])
	hsh.Write(domainMapHash[:])
	headBytes, err := es.cl.StorageAt(ctx, es.ei.AliasContractAddress, common.BytesToHash(hsh.Sum(nil)), blockN)
	if err != nil {
		return nil, nil, err
	}
	return es.resolveAlias(ctx, common.BytesToHash(headBytes), domain, tld, blockN, blockT)
}
func (es *EthereumStorage) resolveAlias(ctx context.Context,
	subdomain [32]byte, domain [32]byte, tld [32]byte,
	blockN *big.Int, blockT *big.Int) (*AliasRegistration, *StateInformation, error) {
	si := &StateInformation{
		CurrentBlock: blockN.Int64(),
		CurrentTime:  blockT.Int64(),
	}
	hsh := sha3.NewKeccak256()
	hsh.Write(tld[:])
	hsh.Write(make([]byte, 32))
	tldobject := hsh.Sum(nil)
	domainMap := new(big.Int).SetBytes(tldobject)
	domainMap.Add(domainMap, big.NewInt(1))
	hsh = sha3.NewKeccak256()
	hsh.Write(domain[:])
	domainMapHash := common.BigToHash(domainMap)
	hsh.Write(domainMapHash[:])
	domainObject := hsh.Sum(nil)
	subdomainMap := new(big.Int).SetBytes(domainObject)
	subdomainMap.Add(subdomainMap, big.NewInt(1))
	subdomainMapHash := common.BigToHash(subdomainMap)
	hsh = sha3.NewKeccak256()
	hsh.Write(subdomain[:])
	hsh.Write(subdomainMapHash[:])
	subdomainBytesObject := hsh.Sum(nil)
	headBytes, err := es.cl.StorageAt(ctx, es.ei.AliasContractAddress, common.BytesToHash(domainObject), blockN)
	if err != nil {
		return nil, nil, err
	}

	//TODO proper bytes depacking
	rvBytes, err := es.cl.StorageAt(ctx, es.ei.AliasContractAddress, common.BytesToHash(subdomainBytesObject), blockN)
	if err != nil {
		return nil, nil, err
	}
	ar := AliasRegistration{
		Head: common.BytesToHash(headBytes),
		//What subdomain is actually resolved
		Subdomain: subdomain,
		Domain:    domain,
		TLD:       tld,
		//The value of the subdomain
		Value: rvBytes,
	}
	return &ar, si, nil
}
func (es *EthereumStorage) ResolveAlias(ctx context.Context, subdomain [32]byte, domain [32]byte, tld [32]byte) (*AliasRegistration, *StateInformation, error) {
	blockN, blockT := es.getHead()
	if blockN == nil || blockT == nil {
		return nil, nil, fmt.Errorf("Still synchronizing to the chain")
	}
	return es.resolveAlias(ctx, subdomain, domain, tld, blockN, blockT)
}
