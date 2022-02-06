package world_mirror

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	wy_protocol "main.go/minecraft/protocol"
	"main.go/plugins/chunk_mirror"
)

func DeReflectLegacySetItemSlots(LegacySetItemSlots []protocol.LegacySetItemSlot) []wy_protocol.LegacySetItemSlot {
	dstLegacySetItemSlot := make([]wy_protocol.LegacySetItemSlot, 0)
	for _, slot := range LegacySetItemSlots {
		dstLegacySetItemSlot = append(dstLegacySetItemSlot, wy_protocol.LegacySetItemSlot{
			ContainerID: slot.ContainerID,
			Slots:       slot.Slots,
		})
	}
	return dstLegacySetItemSlot
}

func DeReflectItemType(ItemType protocol.ItemType) wy_protocol.ItemType {
	return wy_protocol.ItemType{
		// TODO is this right?
		NetworkID:     ItemType.NetworkID,
		MetadataValue: ItemType.MetadataValue,
	}
}

func DeReflectItemStack(ItemStack protocol.ItemStack) wy_protocol.ItemStack {
	return wy_protocol.ItemStack{
		ItemType: DeReflectItemType(ItemStack.ItemType),
		// TODO is this right?
		BlockRuntimeID: int32(chunk_mirror.BlockDeReflectMapping[uint32(ItemStack.BlockRuntimeID)]),
		Count:          ItemStack.Count,
		NBTData:        ItemStack.NBTData,
		CanBePlacedOn:  ItemStack.CanBePlacedOn,
		CanBreak:       ItemStack.CanBreak,
		HasNetworkID:   ItemStack.HasNetworkID,
	}
}

func DeReflectItemInstance(ItemInstance protocol.ItemInstance) wy_protocol.ItemInstance {
	return wy_protocol.ItemInstance{
		StackNetworkID: ItemInstance.StackNetworkID,
		Stack:          DeReflectItemStack(ItemInstance.Stack),
	}
}

func DeReflectInventoryAction(Action protocol.InventoryAction) wy_protocol.InventoryAction {
	return wy_protocol.InventoryAction{
		SourceType:    Action.SourceType,
		WindowID:      Action.WindowID,
		SourceFlags:   Action.SourceFlags,
		InventorySlot: Action.InventorySlot,
		OldItem:       DeReflectItemInstance(Action.OldItem),
		NewItem:       DeReflectItemInstance(Action.NewItem),
	}
}

func DeReflectInventoryActions(Actions []protocol.InventoryAction) []wy_protocol.InventoryAction {
	wy_Actions := make([]wy_protocol.InventoryAction, 0)
	for _, a := range Actions {
		wy_Actions = append(wy_Actions, DeReflectInventoryAction(a))
	}
	return wy_Actions
}
