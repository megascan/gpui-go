# Debt: M3-a11y-consumer-tree

- `a11y/consumer/stubs.go` carries working ports of the node.rs items tree.rs uses (NodeID, nodeState, parentAndIndex, Node.Data/IsFocused/IsGraft/ParentID/Parent/ChildIDs/Children/ID/Role/IsDialog/Label/ActiveDescendant). M3-a11y-consumer-node moves them to node.go, settles the iterator shape of ChildIDs/Children (slices here; upstream is DoubleEnded+ExactSize, likely iter.Seq plus reverse/len helpers), replaces `labelFromLabelledBy`'s panic with the labelled_by port, and updates tree_test.go if the shape changes.
- `testTreeFilter` (lib.rs tests::test_tree_filter) needs FilterResult and Node.IsHidden; row is blocked:M3-a11y-consumer-node.
