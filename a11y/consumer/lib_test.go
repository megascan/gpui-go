package consumer

import (
	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

func nid(id a11y.NodeID) NodeID {
	return newNodeID(id, treeIndex(0))
}

const (
	rootID                    a11y.NodeID = 0
	paragraph0ID              a11y.NodeID = 1
	label00IgnoredID          a11y.NodeID = 2
	paragraph1IgnoredID       a11y.NodeID = 3
	button10HiddenID          a11y.NodeID = 4
	container100HiddenID      a11y.NodeID = 5
	label11ID                 a11y.NodeID = 6
	button12HiddenID          a11y.NodeID = 7
	container120HiddenID      a11y.NodeID = 8
	paragraph2ID              a11y.NodeID = 9
	label20ID                 a11y.NodeID = 10
	paragraph3IgnoredID       a11y.NodeID = 11
	emptyContainer30IgnoredID a11y.NodeID = 12
	link31IgnoredID           a11y.NodeID = 13
	label310ID                a11y.NodeID = 14
	button32ID                a11y.NodeID = 15
	emptyContainer33IgnoredID a11y.NodeID = 16
)

func testTree() *Tree {
	root := a11y.NewNode(a11y.RoleRootWebArea)
	root.SetChildren([]a11y.NodeID{paragraph0ID, paragraph1IgnoredID, paragraph2ID, paragraph3IgnoredID})
	paragraph0 := a11y.NewNode(a11y.RoleParagraph)
	paragraph0.SetChildren([]a11y.NodeID{label00IgnoredID})
	label00Ignored := a11y.NewNode(a11y.RoleLabel)
	label00Ignored.SetValue("label_0_0_ignored")
	paragraph1Ignored := a11y.NewNode(a11y.RoleParagraph)
	paragraph1Ignored.SetTransform(a11y.AffineTranslate(a11y.Vec2{X: 10.0, Y: 40.0}))
	paragraph1Ignored.SetBounds(a11y.Rect{X0: 0.0, Y0: 0.0, X1: 800.0, Y1: 40.0})
	paragraph1Ignored.SetChildren([]a11y.NodeID{button10HiddenID, label11ID, button12HiddenID})
	button10Hidden := a11y.NewNode(a11y.RoleButton)
	button10Hidden.SetLabel("button_1_0_hidden")
	button10Hidden.SetHidden()
	button10Hidden.SetChildren([]a11y.NodeID{container100HiddenID})
	container100Hidden := a11y.NewNode(a11y.RoleGenericContainer)
	container100Hidden.SetHidden()
	label11 := a11y.NewNode(a11y.RoleLabel)
	label11.SetBounds(a11y.Rect{X0: 10.0, Y0: 10.0, X1: 90.0, Y1: 30.0})
	label11.SetValue("label_1_1")
	button12Hidden := a11y.NewNode(a11y.RoleButton)
	button12Hidden.SetLabel("button_1_2_hidden")
	button12Hidden.SetHidden()
	button12Hidden.SetChildren([]a11y.NodeID{container120HiddenID})
	container120Hidden := a11y.NewNode(a11y.RoleGenericContainer)
	container120Hidden.SetHidden()
	paragraph2 := a11y.NewNode(a11y.RoleParagraph)
	paragraph2.SetChildren([]a11y.NodeID{label20ID})
	label20 := a11y.NewNode(a11y.RoleLabel)
	label20.SetLabel("label_2_0")
	paragraph3Ignored := a11y.NewNode(a11y.RoleParagraph)
	paragraph3Ignored.SetChildren([]a11y.NodeID{emptyContainer30IgnoredID, link31IgnoredID, button32ID, emptyContainer33IgnoredID})
	emptyContainer30Ignored := a11y.NewNode(a11y.RoleGenericContainer)
	link31Ignored := a11y.NewNode(a11y.RoleLink)
	link31Ignored.SetChildren([]a11y.NodeID{label310ID})
	label310 := a11y.NewNode(a11y.RoleLabel)
	label310.SetValue("label_3_1_0")
	button32 := a11y.NewNode(a11y.RoleButton)
	button32.SetLabel("button_3_2")
	emptyContainer33Ignored := a11y.NewNode(a11y.RoleGenericContainer)
	initialUpdate := a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: rootID, Node: root},
			{ID: paragraph0ID, Node: paragraph0},
			{ID: label00IgnoredID, Node: label00Ignored},
			{ID: paragraph1IgnoredID, Node: paragraph1Ignored},
			{ID: button10HiddenID, Node: button10Hidden},
			{ID: container100HiddenID, Node: container100Hidden},
			{ID: label11ID, Node: label11},
			{ID: button12HiddenID, Node: button12Hidden},
			{ID: container120HiddenID, Node: container120Hidden},
			{ID: paragraph2ID, Node: paragraph2},
			{ID: label20ID, Node: label20},
			{ID: paragraph3IgnoredID, Node: paragraph3Ignored},
			{ID: emptyContainer30IgnoredID, Node: emptyContainer30Ignored},
			{ID: link31IgnoredID, Node: link31Ignored},
			{ID: label310ID, Node: label310},
			{ID: button32ID, Node: button32},
			{ID: emptyContainer33IgnoredID, Node: emptyContainer33Ignored},
		},
		Tree:   gpuiutil.Some(a11y.NewTree(rootID)),
		TreeID: a11y.TreeIDRoot(),
		Focus:  rootID,
	}
	return NewTree(initialUpdate, false)
}
