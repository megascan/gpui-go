package taffy

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
)

// PrintTreeStdout prints a debug representation of the computed layout of a tree, starting at root (upstream print_tree).
func PrintTreeStdout(tree PrintTree, root NodeID) {
	var buffer bytes.Buffer
	if err := WriteTree(&buffer, tree, root); err != nil {
		panic(fmt.Sprintf("Wrote tree debug representation to Stdout: %v", err))
	}
	fmt.Println(buffer.String())
}

// WriteTree writes a debug representation of the computed layout of a tree to writer, starting at root.
func WriteTree(writer io.Writer, tree PrintTree, root NodeID) error {
	if _, err := fmt.Fprintln(writer, "TREE"); err != nil {
		return err
	}
	return writeNode(writer, tree, root, false, "")
}

func formatF32(v float32) string {
	switch {
	case math.IsNaN(float64(v)):
		return "NaN"
	case math.IsInf(float64(v), 1):
		return "inf"
	case math.IsInf(float64(v), -1):
		return "-inf"
	}
	return strconv.FormatFloat(float64(v), 'f', -1, 32)
}

func writeNode(writer io.Writer, tree PrintTree, nodeID NodeID, hasSibling bool, linesString string) error {
	layout := tree.GetFinalLayout(nodeID)
	display := tree.GetDebugLabel(nodeID)
	numChildren := tree.ChildCount(nodeID)

	forkString := "└── "
	if hasSibling {
		forkString = "├── "
	}
	_, err := fmt.Fprintf(writer,
		"%s%s %s [x: %-4s y: %-4s w: %-4s h: %-4s overflow: l:%s r:%s t:%s b:%s, border: l:%s r:%s t:%s b:%s, padding: l:%s r:%s t:%s b:%s] (%v)\n",
		linesString,
		forkString,
		display,
		formatF32(layout.Location.X),
		formatF32(layout.Location.Y),
		formatF32(layout.Size.Width),
		formatF32(layout.Size.Height),
		formatF32(layout.ScrollableOverflowRect.Left),
		formatF32(layout.ScrollableOverflowRect.Right),
		formatF32(layout.ScrollableOverflowRect.Top),
		formatF32(layout.ScrollableOverflowRect.Bottom),
		formatF32(layout.Border.Left),
		formatF32(layout.Border.Right),
		formatF32(layout.Border.Top),
		formatF32(layout.Border.Bottom),
		formatF32(layout.Padding.Left),
		formatF32(layout.Padding.Right),
		formatF32(layout.Padding.Top),
		formatF32(layout.Padding.Bottom),
		nodeID,
	)
	if err != nil {
		return err
	}
	bar := "    "
	if hasSibling {
		bar = "│   "
	}
	newString := linesString + bar

	index := 0
	for child := range tree.ChildIds(nodeID) {
		hasSibling := index < numChildren-1
		if err := writeNode(writer, tree, child, hasSibling, newString); err != nil {
			return err
		}
		index++
	}
	return nil
}
