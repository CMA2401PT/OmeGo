package function

import (
	"fmt"
	command "main.go/plugins/fastbuilder/bridge"
	"main.go/plugins/fastbuilder/i18n"
	"strconv"
	"strings"
)

type Function struct {
	Name          string
	OwnedKeywords []string

	FunctionType    byte
	SFMinSliceLen   uint16
	SFArgumentTypes []byte
	FunctionContent interface{} // Regular/Simple: func(*minecraft.Conn,interface{})
	// Continue: map[string]*FunctionChainItem
}

type FunctionChainItem struct {
	FunctionType  byte
	ArgumentTypes []byte
	Content       interface{}
}

const (
	FunctionTypeSimple   = 0 // End of simple chain
	FunctionTypeContinue = 1 // Simple chain
	FunctionTypeRegular  = 2
)

const (
	SimpleFunctionArgumentString  = 0
	SimpleFunctionArgumentDecider = 1
	SimpleFunctionArgumentInt     = 2
	//SimpleFunctionArgumentEnum  = ---->
)

var FunctionMap = make(map[string]*Function)

func RegisterFunction(function *Function) {
	for _, nm := range function.OwnedKeywords {
		if _, ok := FunctionMap[nm]; !ok {
			FunctionMap[nm] = function
		}
	}
}

type EnumInfo struct {
	WantedValuesDescription string // "discrete, continuous, none"
	Parser                  func(string) byte
	InvalidValue            byte
}

var SimpleFunctionEnums []*EnumInfo

func RegisterEnum(desc string, parser func(string) byte, inv byte) int {
	SimpleFunctionEnums = append(SimpleFunctionEnums, &EnumInfo{WantedValuesDescription: desc, InvalidValue: inv, Parser: parser})
	return len(SimpleFunctionEnums) - 1 + 3
}

func Process(msg string) {
	slc := strings.Split(msg, " ")
	fun, ok := FunctionMap[slc[0]]
	if !ok {
		return
	}
	if fun.FunctionType == FunctionTypeRegular {
		cont, _ := fun.FunctionContent.(func(string))
		cont(msg)
		return
	}
	if len(slc) < int(fun.SFMinSliceLen) {
		command.Tellraw(fmt.Sprintf("Parser: Simple function %s required at least %d arguments, but got %d.", fun.Name, fun.SFMinSliceLen, len(slc)))
		return
	}
	var arguments []interface{}
	ic := 1
	cc := &FunctionChainItem{
		FunctionType:  fun.FunctionType,
		ArgumentTypes: fun.SFArgumentTypes,
		Content:       fun.FunctionContent,
	}
	for {
		if cc.FunctionType == FunctionTypeContinue {
			if len(slc) <= ic {
				rf, _ := cc.Content.(map[string]*FunctionChainItem)
				itm, got := rf[""]
				if !got {
					command.Tellraw(I18n.T(I18n.SimpleParser_Too_few_args))
					return
				}
				cc = itm
				continue
			}
			rfc, _ := cc.Content.(map[string]*FunctionChainItem)
			chainitem, got := rfc[slc[ic]]
			if !got {
				command.Tellraw(I18n.T(I18n.SimpleParser_Invalid_decider))
				return
			}
			cc = chainitem
			ic++
			continue
		}
		if len(cc.ArgumentTypes) > len(slc)-ic {
			command.Tellraw(I18n.T(I18n.SimpleParser_Too_few_args))
			return
		}
		for _, tp := range cc.ArgumentTypes {
			if tp == SimpleFunctionArgumentString {
				arguments = append(arguments, slc[ic])
			} else if tp == SimpleFunctionArgumentDecider {
				command.Tellraw("Parser: Internal error - argument type [decider] is preserved.")
				fmt.Println("Parser: Internal error - DO NOT REGISTER Decider ARGUMENT!")
				return
			} else if tp == SimpleFunctionArgumentInt {
				parsedInt, err := strconv.Atoi(slc[ic])
				if err != nil {
					command.Tellraw(fmt.Sprintf("%s: %v", I18n.T(I18n.SimpleParser_Int_ParsingFailed), err))
					return
				}
				arguments = append(arguments, parsedInt)
			} else {
				eindex := int(tp - 3)
				if eindex >= len(SimpleFunctionEnums) {
					command.Tellraw("Parser: Internal error, unregistered enum")
					fmt.Printf("Internal error, unregistered enum %d\n", int(tp))
					return
				}
				ei := SimpleFunctionEnums[eindex]
				itm := ei.Parser(slc[ic])
				if itm == ei.InvalidValue {
					command.Tellraw(fmt.Sprintf(I18n.T(I18n.SimpleParser_InvEnum), ei.WantedValuesDescription))
					return
				}
				arguments = append(arguments, itm)
			}
			ic++
		}
		cont, _ := cc.Content.(func([]interface{}))
		if cont == nil {
			cont, _ := cc.Content.(func([]interface{}))
			cont(arguments)
			return
		}
		cont(arguments)
		return
	}
}
