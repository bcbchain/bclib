package hdwal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/bcbchain/bclib/algorithm"
	"github.com/bcbchain/bclib/bcdb"
	"github.com/bcbchain/bclib/tendermint/ed25519"
	"github.com/bcbchain/bclib/tendermint/go-amino"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/tyler-smith/go-bip39"
	"math"
	"strconv"
	"time"
)

var cdc = amino.NewCodec()
var db *bcdb.GILevelDB

type HdWal struct {
	Mnemonic string `json:"mnemonic"`
	Seed     []byte `json:"seed"`
	Hash     []byte `json:"hash"`
}

func init() {
	crypto.RegisterAmino(cdc)
}

func SetDB(bcdb *bcdb.GILevelDB) {
	db = bcdb
}

func entropy() []byte {
	randomBytes := make([]byte, 0)
	cpuPercent, _ := cpu.Percent(time.Second, false)
	memory, _ := mem.VirtualMemory()
	diskStatus, _ := disk.Usage("/")

	ioCounters, _ := net.IOCounters(true)
	netWork := strconv.Itoa(int(ioCounters[0].BytesSent + ioCounters[0].BytesRecv))

	randomBytes = append(randomBytes, crypto.CRandBytes(32)...)
	randomBytes = append(randomBytes, float64ToByte(cpuPercent[0])...)
	randomBytes = append(randomBytes, float64ToByte(memory.UsedPercent)...)
	randomBytes = append(randomBytes, float64ToByte(diskStatus.UsedPercent)...)
	randomBytes = append(randomBytes, []byte(netWork)...)

	entropy := crypto.Sha256(randomBytes)
	return entropy[:16]
}

func Mnemonic(possword string) (string, error) {
	bytes, err := db.Get(keyOfHdWal())
	if err != nil {
		return "", err
	}

	if len(bytes) != 0 {
		return "", errors.New("Mnemonic already exists!")
	}

	entropyBytes := entropy()
	mnemonic, err := bip39.NewMnemonic(entropyBytes)
	if err != nil {
		return "", err
	}

	seedBytes, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return "", err
	}

	var hdWal HdWal
	hdWal.Mnemonic = mnemonic
	hdWal.Seed = seedBytes

	hashByte := append([]byte(mnemonic), seedBytes...)
	hdWal.Hash = crypto.Sha256(hashByte)

	err = encryptAndSave(possword, hdWal)
	if err != nil {
		return "", err
	}

	return mnemonic, nil
}

func Export(possword string) (string, error) {
	hdWal, err := getHdWal(possword)
	if err != nil {
		return "", err
	}

	return hdWal.Mnemonic, nil
}

func Import(mnemonic string, possword string) error {
	seedBytes, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return err
	}

	var hdWal HdWal
	hdWal.Mnemonic = mnemonic
	hdWal.Seed = seedBytes

	err = encryptAndSave(possword, hdWal)
	if err != nil {
		return err
	}

	return nil
}

func NewPrivKey(path string, possword string) (crypto.PrivKey, error) {
	hdWal, err := getHdWal(possword)
	if err != nil {
		return nil, err
	}

	extendedKey, err := NewMaster(hdWal.Seed)
	if err != nil {
		return nil, err
	}

	derivationPath, err := ParseDerivationPath(path)
	if err != nil {
		return nil, err
	}

	for _, index := range derivationPath {
		childExtendedKey, err := extendedKey.Child(index)
		if err != nil {
			return nil, err
		}
		extendedKey = childExtendedKey
	}

	var privkey [64]byte
	copy(privkey[:32], extendedKey.key)
	ed25519.MakePublicKey(&privkey)

	return crypto.PrivKeyEd25519(privkey), nil
}

func encryptAndSave(possword string, hdWal HdWal) error {
	jsonBytes, err := cdc.MarshalJSON(hdWal)
	if err != nil {
		return err
	}

	hdWalBytes := algorithm.EncryptWithPassword(jsonBytes, []byte(possword), nil)
	err = db.Set(keyOfHdWal(), hdWalBytes)
	if err != nil {
		return err
	}
	return nil
}

func getHdWal(possword string) (*HdWal, error) {
	bytes, err := db.Get(keyOfHdWal())
	if err != nil {
		return nil, err
	}

	if len(bytes) == 0 {
		return nil, errors.New("Mnemonic does not exist!")
	}

	hdWalBytes, err := algorithm.DecryptWithPassword(bytes, []byte(possword), nil)
	if err != nil {
		return nil, fmt.Errorf("The password is wrong, err info : %s", err)
	}

	var hdWal HdWal
	err = cdc.UnmarshalJSON(hdWalBytes, &hdWal)
	if err != nil {
		return nil, err
	}
	return &hdWal, nil
}

func keyOfHdWal() []byte {
	return []byte("/hdWallet/hdwal")
}

func float64ToByte(float float64) []byte {
	bits := math.Float64bits(float)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)
	return bytes
}

//func LoadAccount(name string, possword string) crypto.PrivKey {
//
//}

//func Path(index int) string {
//	return fmt.Sprintf("m/44'/60'/0'/0/%v", index)
//}
