package bdump

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/andybalholm/brotli"
	"main.go/plugins/builder/define"
	"main.go/plugins/builder/ir"
	"main.go/plugins/fastbuilder/bdump"
	"main.go/plugins/fastbuilder/i18n"
	"main.go/plugins/fastbuilder/types"
	"main.go/plugins/fastbuilder/world_provider"
	"os"
)

func ReadBrString(br *bytes.Buffer) (string, error) {
	str := ""
	c := make([]byte, 1)
	for {
		_, err := br.Read(c)
		if err != nil {
			return "", err
		}
		if c[0] == 0 {
			break
		}
		str += string(c)
	}
	return str, nil
}

func LoadBDX(filePath string, needSign bool, sendNotice func(string), ir *ir.IR) error {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return I18n.ProcessSystemFileError(err)
	}
	defer file.Close()
	{
		header3bytes := make([]byte, 3)
		_, err := file.Read(header3bytes)
		if err != nil {
			return fmt.Errorf(I18n.T(I18n.BDump_EarlyEOFRightWhenOpening))
		}
		if string(header3bytes) != "BD@" {
			return fmt.Errorf(I18n.T(I18n.BDump_NotBDX_Invheader))
		}
	}
	bro := brotli.NewReader(file)
	br := &bytes.Buffer{}
	filelen, _ := br.ReadFrom(bro)
	if filelen == 0 {
		return fmt.Errorf(I18n.T(I18n.InvalidFileError))
	}
	{
		bts := br.Bytes()
		if bts[filelen-1] == 90 {
			sendNotice(fmt.Sprintf(I18n.T(I18n.BDump_SignedVerifying)))
			lent := int64(bts[filelen-2])
			sign := bts[filelen-lent-2 : filelen-2]
			cor, un, err := bdump.VerifyBDX(bts[:filelen-lent-3], sign)
			if cor {
				return fmt.Errorf(I18n.T(I18n.FileCorruptedError))
			}
			if err != nil {
				e := fmt.Errorf(I18n.T(I18n.BDump_VerificationFailedFor), err)
				if needSign {
					return e
				} else {
					sendNotice(fmt.Sprintf("%s(%s): %v", I18n.T(I18n.ERRORStr), I18n.T(I18n.IgnoredStr), e))
				}
			} else {
				sendNotice(fmt.Sprintf(I18n.T(I18n.BDump_FileSigned), un))
			}
		} else if needSign {
			return fmt.Errorf("%s.", I18n.T(I18n.BDump_FileNotSigned))
		} else {
			sendNotice(fmt.Sprintf("%s!", I18n.T(I18n.BDump_FileNotSigned)))
		}
	}
	{
		tempbuf := make([]byte, 4)
		_, err := br.Read(tempbuf)
		if err != nil {
			return fmt.Errorf(I18n.T(I18n.InvalidFileError))
		}
		if string(tempbuf) != "BDX\x00" {
			return fmt.Errorf(I18n.T(I18n.BDump_NotBDX_Invinnerheader))
		}
	}
	ReadBrString(br) // Ignores author field
	curcmdbuf := make([]byte, 1)
	brushPosition := []int{0, 0, 0}
	var blocksStrPool []string
	var runtimeIdPoolUsing []*types.ConstBlock
	prevCmd := 0
	for {
		prevCmd = int(curcmdbuf[0])
		_, err := br.Read(curcmdbuf)
		if err != nil {
			return fmt.Errorf("Failed to get construction command, file may be corrupted")
		}
		cmd := curcmdbuf[0]
		if cmd == 88 {
			break
		}
		if cmd == 1 {
			bstr, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos:0], file may be corrupted!")
			}
			blocksStrPool = append(blocksStrPool, bstr)
			continue
		} else if cmd == 2 {
			rdst := make([]byte, 2)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos1], file may be corrupted")
			}
			jumpval := binary.BigEndian.Uint16(rdst)
			brushPosition[0] += int(jumpval)
			brushPosition[1] = 0
			brushPosition[2] = 0
		} else if cmd == 3 {
			brushPosition[0]++
			brushPosition[1] = 0
			brushPosition[2] = 0
		} else if cmd == 4 {
			rdst := make([]byte, 2)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos2], file may be corrupted")
			}
			jumpval := binary.BigEndian.Uint16(rdst)
			brushPosition[1] += int(jumpval)
			brushPosition[2] = 0
		} else if cmd == 5 {
			brushPosition[1]++
			brushPosition[2] = 0
		} else if cmd == 6 {
			rdst := make([]byte, 2)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos3], file may be corrupted")
			}
			jumpval := binary.BigEndian.Uint16(rdst)
			brushPosition[2] += int(jumpval)
		} else if cmd == 7 {
			rdst := make([]byte, 2)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos4], file may be corrupted")
			}
			blockId := binary.BigEndian.Uint16(rdst)
			_, err = br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos5], file may be corrupted")
			}
			if int(blockId) >= len(blocksStrPool) {
				fmt.Printf("WARNING: Invalid command")
				continue
			}
			blockData := binary.BigEndian.Uint16(rdst)
			blockName := &blocksStrPool[int(blockId)]
			ir.SetBlock(define.PE(brushPosition[0]), define.PE(brushPosition[1]), define.PE(brushPosition[2]), define.BlockDescribe{
				Name: *blockName,
				Meta: blockData,
			})
		} else if cmd == 8 {
			brushPosition[2]++
		} else if cmd == 9 {
			// Command: NOP
		} else if cmd == 10 {
			rdst := make([]byte, 4)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos6], file may be corrupted")
			}
			jumpval := binary.BigEndian.Uint32(rdst)
			brushPosition[0] += int(jumpval)
			brushPosition[1] = 0
			brushPosition[2] = 0
		} else if cmd == 11 {
			rdst := make([]byte, 4)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos7], file may be corrupted")
			}
			jumpval := binary.BigEndian.Uint32(rdst)
			brushPosition[1] += int(jumpval)
			brushPosition[2] = 0
		} else if cmd == 12 {
			rdst := make([]byte, 4)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos8], file may be corrupted")
			}
			jumpval := binary.BigEndian.Uint32(rdst)
			brushPosition[2] += int(jumpval)
		} else if cmd == 13 {
			fmt.Printf("WARNING: BDump/Import: Use of reserved command\n")
		} else if cmd == 14 {
			brushPosition[0]++
		} else if cmd == 15 {
			brushPosition[0]--
		} else if cmd == 16 {
			brushPosition[1]++
		} else if cmd == 17 {
			brushPosition[1]--
		} else if cmd == 18 {
			brushPosition[2]++
		} else if cmd == 19 {
			brushPosition[2]--
		} else if cmd == 20 {
			rdst := make([]byte, 2)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos9], file may be corrupted")
			}
			jumpval := binary.BigEndian.Uint16(rdst)
			brushPosition[0] += int(int16(jumpval))
		} else if cmd == 21 {
			rdst := make([]byte, 4)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos10], file may be corrupted")
			}
			jumpval := binary.BigEndian.Uint32(rdst)
			brushPosition[0] += int(int32(jumpval))
		} else if cmd == 22 {
			rdst := make([]byte, 2)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos11], file may be corrupted")
			}
			jumpval := binary.BigEndian.Uint16(rdst)
			brushPosition[1] += int(int16(jumpval))
		} else if cmd == 23 {
			rdst := make([]byte, 4)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos12], file may be corrupted")
			}
			jumpval := binary.BigEndian.Uint32(rdst)
			brushPosition[1] += int(int32(jumpval))
		} else if cmd == 24 {
			rdst := make([]byte, 2)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos13], file may be corrupted")
			}
			jumpval := binary.BigEndian.Uint16(rdst)
			brushPosition[2] += int(int16(jumpval))
		} else if cmd == 25 {
			rdst := make([]byte, 4)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos14], file may be corrupted")
			}
			jumpval := binary.BigEndian.Uint32(rdst)
			brushPosition[2] += int(int32(jumpval))
		} else if cmd == 26 {
			fbuf := make([]byte, 4)
			_, err := br.Read(fbuf)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos15], file may be corrupted")
			}
			cbmode := binary.BigEndian.Uint32(fbuf)
			command, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos16], file may be corrupted")
			}
			cusname, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos17], file may be corrupted")
			}
			lasout, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos18], file may be corrupted")
			}
			_, err = br.Read(fbuf)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos19], file may be corrupted")
			}
			tickdelay := int32(binary.BigEndian.Uint32(fbuf))
			_, err = br.Read(fbuf)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos20], file may be corrupted")
			}
			fbools := []bool{false, false, false, false}
			if fbuf[0] == 1 {
				fbools[0] = true
			}
			if fbuf[1] == 1 {
				fbools[1] = true
			}
			if fbuf[2] == 1 {
				fbools[2] = true
			}
			if fbuf[3] == 1 {
				fbools[3] = true
			}
			ir.SetNbtBlock(define.PE(brushPosition[0]), define.PE(brushPosition[1]), define.PE(brushPosition[2]), define.BlockDescribe{
				Name: "set_cmd_block",
				Meta: 0,
			}, define.Nbt{
				"Mode":               cbmode,
				"Command":            command,
				"CustomName":         cusname,
				"LastOutput":         lasout,
				"TickDelay":          tickdelay,
				"ExecuteOnFirstTick": fbools[0],
				"TrackOutput":        fbools[1],
				"Conditional":        fbools[2],
				"NeedRedstone":       fbools[3],
			})
		} else if cmd == 27 {
			rdst := make([]byte, 2)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos4], file may be corrupted")
			}
			blockId := binary.BigEndian.Uint16(rdst)
			_, err = br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos5], file may be corrupted")
			}
			if int(blockId) >= len(blocksStrPool) {
				fmt.Printf("WARNING: Invalid command")
				continue
			}
			blockData := binary.BigEndian.Uint16(rdst)
			blockName := &blocksStrPool[int(blockId)]
			fbuf := make([]byte, 4)
			_, err = br.Read(fbuf)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos15], file may be corrupted")
			}
			cbmode := binary.BigEndian.Uint32(fbuf)
			command, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos16], file may be corrupted")
			}
			cusname, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos17], file may be corrupted")
			}
			lasout, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos18], file may be corrupted")
			}
			_, err = br.Read(fbuf)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos19], file may be corrupted")
			}
			tickdelay := int32(binary.BigEndian.Uint32(fbuf))
			_, err = br.Read(fbuf)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos20], file may be corrupted")
			}
			fbools := []bool{false, false, false, false}
			if fbuf[0] == 1 {
				fbools[0] = true
			}
			if fbuf[1] == 1 {
				fbools[1] = true
			}
			if fbuf[2] == 1 {
				fbools[2] = true
			}
			if fbuf[3] == 1 {
				fbools[3] = true
			}
			ir.SetNbtBlock(define.PE(brushPosition[0]), define.PE(brushPosition[1]), define.PE(brushPosition[2]), define.BlockDescribe{
				Name: *blockName,
				Meta: uint16(blockData),
			}, define.Nbt{
				"Mode":               cbmode,
				"Command":            command,
				"CustomName":         cusname,
				"LastOutput":         lasout,
				"TickDelay":          tickdelay,
				"ExecuteOnFirstTick": fbools[0],
				"TrackOutput":        fbools[1],
				"Conditional":        fbools[2],
				"NeedRedstone":       fbools[3],
			})
		} else if cmd == 36 {
			rdst := make([]byte, 2)
			_, err = br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos5], file may be corrupted")
			}
			fbuf := make([]byte, 4)
			_, err = br.Read(fbuf)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos15], file may be corrupted")
			}
			cbmode := binary.BigEndian.Uint32(fbuf)
			command, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos16], file may be corrupted")
			}
			cusname, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos17], file may be corrupted")
			}
			lasout, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos18], file may be corrupted")
			}
			_, err = br.Read(fbuf)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos19], file may be corrupted")
			}
			tickdelay := int32(binary.BigEndian.Uint32(fbuf))
			_, err = br.Read(fbuf)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos20], file may be corrupted")
			}
			fbools := []bool{false, false, false, false}
			if fbuf[0] == 1 {
				fbools[0] = true
			}
			if fbuf[1] == 1 {
				fbools[1] = true
			}
			if fbuf[2] == 1 {
				fbools[2] = true
			}
			if fbuf[3] == 1 {
				fbools[3] = true
			}
			ir.SetNbtBlock(define.PE(brushPosition[0]), define.PE(brushPosition[1]), define.PE(brushPosition[2]), define.BlockDescribe{
				Name: "command_block",
				Meta: binary.BigEndian.Uint16(rdst),
			}, define.Nbt{
				"Mode":               cbmode,
				"Command":            command,
				"CustomName":         cusname,
				"LastOutput":         lasout,
				"TickDelay":          tickdelay,
				"ExecuteOnFirstTick": fbools[0],
				"TrackOutput":        fbools[1],
				"Conditional":        fbools[2],
				"NeedRedstone":       fbools[3],
			})
		} else if cmd == 28 {
			rdst := make([]byte, 1)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos21], file may be corrupted")
			}
			brushPosition[0] += int(int8(rdst[0]))
		} else if cmd == 29 {
			rdst := make([]byte, 1)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos22], file may be corrupted")
			}
			brushPosition[1] += int(int8(rdst[0]))
		} else if cmd == 30 {
			rdst := make([]byte, 1)
			_, err := br.Read(rdst)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos23], file may be corrupted")
			}
			brushPosition[2] += int(int8(rdst[0]))
		} else if cmd == 31 {
			poolId := make([]byte, 1)
			_, err := br.Read(poolId)
			if err != nil {
				return fmt.Errorf("Failed to get pool id, file may be corrupted.")
			}
			if poolId[0] == 117 {
				runtimeIdPoolUsing = world_provider.RuntimeIdArray_117
			} else {
				return fmt.Errorf("This file is using an unknown runtime id pool, we're unable to resolve it.")
			}
		} else if cmd == 32 {
			runtimeIdMem := make([]byte, 2)
			_, err = br.Read(runtimeIdMem)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos24], file may be corrupted")
			}
			runtimeId := uint32(binary.BigEndian.Uint16(runtimeIdMem))
			Block := runtimeIdPoolUsing[runtimeId].Take()
			ir.SetBlock(define.PE(brushPosition[0]), define.PE(brushPosition[1]), define.PE(brushPosition[2]), define.BlockDescribe{
				Name: *Block.Name,
				Meta: Block.Data,
			})
		} else if cmd == 33 {
			runtimeIdMem := make([]byte, 4)
			_, err = br.Read(runtimeIdMem)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos24], file may be corrupted")
			}
			runtimeId := binary.BigEndian.Uint32(runtimeIdMem)
			Block := runtimeIdPoolUsing[runtimeId].Take()
			ir.SetBlock(define.PE(brushPosition[0]), define.PE(brushPosition[1]), define.PE(brushPosition[2]), define.BlockDescribe{
				Name: *Block.Name,
				Meta: Block.Data,
			})
		} else if cmd == 34 || cmd == 35 {
			var dataval uint32
			if cmd == 34 {
				datavbuf := make([]byte, 2)
				_, err = br.Read(datavbuf)
				dataval = uint32(binary.BigEndian.Uint16(datavbuf))
			} else {
				datavbuf := make([]byte, 4)
				_, err = br.Read(datavbuf)
				dataval = binary.BigEndian.Uint32(datavbuf)
			}
			fbuf := make([]byte, 4)
			_, err = br.Read(fbuf)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos a15], file may be corrupted")
			}
			cbmode := binary.BigEndian.Uint32(fbuf)
			command, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos a16], file may be corrupted")
			}
			cusname, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos a 17], file may be corrupted")
			}
			lasout, err := ReadBrString(br)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos a18], file may be corrupted")
			}
			_, err = br.Read(fbuf)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos a19], file may be corrupted")
			}
			tickdelay := int32(binary.BigEndian.Uint32(fbuf))
			_, err = br.Read(fbuf)
			if err != nil {
				return fmt.Errorf("Failed to get argument for cmd[pos a20], file may be corrupted")
			}
			fbools := []bool{false, false, false, false}
			if fbuf[0] == 1 {
				fbools[0] = true
			}
			if fbuf[1] == 1 {
				fbools[1] = true
			}
			if fbuf[2] == 1 {
				fbools[2] = true
			}
			if fbuf[3] == 1 {
				fbools[3] = true
			}
			Block := runtimeIdPoolUsing[dataval].Take()
			ir.SetNbtBlock(define.PE(brushPosition[0]), define.PE(brushPosition[1]), define.PE(brushPosition[2]), define.BlockDescribe{
				Name: *Block.Name,
				Meta: Block.Data,
			}, define.Nbt{
				"Mode":               cbmode,
				"Command":            command,
				"CustomName":         cusname,
				"LastOutput":         lasout,
				"TickDelay":          tickdelay,
				"ExecuteOnFirstTick": fbools[0],
				"TrackOutput":        fbools[1],
				"Conditional":        fbools[2],
				"NeedRedstone":       fbools[3],
			})
		} else if cmd == 37 || cmd == 38 {
			fmt.Printf("Chest is Ignored\n")
		} else {
			fmt.Printf("WARNING: BDump/Import: Unimplemented method found : %d\n", cmd)
			fmt.Printf("WARNING: BDump/Import: Previous command is: %d\n", prevCmd)
			fmt.Printf("WARNING: BDump/Import: Trying to ignore, it will probably cause an error!\n")
		}
	}
	return nil
}
