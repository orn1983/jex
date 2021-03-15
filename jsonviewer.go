package main

// TODO FIX string quoting

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/rivo/tview"
	"github.com/gdamore/tcell"
)

type Slicer []interface{}
func (s Slicer) String() string {
	return fmt.Sprintf("[ %d ]", len(s))
}

type StringMapper map[string]interface{}
func (m StringMapper) String() string {
	return fmt.Sprintf("{ %d }", len(m))
}

type Truther bool
func (t Truther) String() string {
	return fmt.Sprintf("%t", t)
}

type Nuller bool
func (n Nuller) String() string {
	return "null"
}

type Stringer string
func (s Stringer) String() string {
	return fmt.Sprintf("%s", string(s))
}

func CreateStringer(i interface{}) fmt.Stringer {
	switch i.(type) {
	case []interface{}:
		return Slicer(i.([]interface{}))
	case map[string]interface{}:
		return StringMapper(i.(map[string]interface{}))
	case bool:
		return Truther(i.(bool))
	case json.Number:
		return i.(json.Number)
	case string:
		return Stringer(i.(string))
	case nil:
		return Nuller(false)
	default:
		panic(fmt.Sprintf("unsupported data type %T in json!", i))
	}
	return nil
}

func CreateNodeRecursive(i interface{}) (*tview.TreeNode, error) {
	var node *tview.TreeNode
	var err error
	switch i.(type) {
	case []interface{}:
		s := CreateStringer(i)
		node = tview.NewTreeNode(s.String()).SetColor(tcell.ColorGreen)
		node.SetReference(s.String())
		for _, child := range i.([]interface{}) {
			nn, err := CreateNodeRecursive(child)
			if err != nil {
				panic(err)
			}
			node.AddChild(nn)
		}
	case map[string]interface{}:
		s := CreateStringer(i)
		node = tview.NewTreeNode(s.String()).SetColor(tcell.ColorBlue)
		node.SetReference(s.String())
		for childKey, childVal := range i.(map[string]interface{}) {
			switch childVal.(type) {
				case string:
					cvs := CreateStringer(childVal)
					description  := fmt.Sprintf("%s: \"%s\"", childKey, cvs.String())
					childNode := tview.NewTreeNode(description)
					if err != nil {
						panic(err)
					}
					node.AddChild(childNode)
				case json.Number:
					cvs := CreateStringer(childVal)
					description  := fmt.Sprintf("%s: %s", childKey, cvs.String())
					childNode := tview.NewTreeNode(description)
					node.AddChild(childNode)
				case bool:
					cvs := CreateStringer(childVal)
					description  := fmt.Sprintf("%s: %s", childKey, cvs.String())
					childNode := tview.NewTreeNode(description).SetColor(tcell.ColorOlive)
					node.AddChild(childNode)
				case nil:
					cvs := CreateStringer(childVal)
					description  := fmt.Sprintf("%s: %s", childKey, cvs.String())
					childNode := tview.NewTreeNode(description).SetColor(tcell.ColorPurple)
					node.AddChild(childNode)
				default:
					childNode, err := CreateNodeRecursive(childVal)
					if err != nil {
						panic(err)
					}
					childNode.SetText(fmt.Sprintf("%s: %s", childKey, childNode.GetText()))
					node.AddChild(childNode)
			}
		}
	case string:
		node = tview.NewTreeNode(i.(string))
		node.SetText(fmt.Sprintf("\"%s\"", i.(string)))
		node.SetReference(i.(string))
	case json.Number:
		s := CreateStringer(i)
		node = tview.NewTreeNode(s.String())
		node.SetReference(s.String())
	case bool:
		s := CreateStringer(i)
		node = tview.NewTreeNode(s.String()).SetColor(tcell.ColorOlive)
		node.SetReference(s.String())
	case nil:
		s := CreateStringer(i)
		node = tview.NewTreeNode(s.String()).SetColor(tcell.ColorPurple)
		node.SetReference(s.String())
	default:
		err = fmt.Errorf("unsupported type %T for node creation", i)
	}
	node.SetExpanded(false)
	return node, err
}

func selected(node *tview.TreeNode) {
	reference := node.GetReference()
	if reference == nil {
		return
	}
	children := node.GetChildren()
	if len(children) == 0 {
		return
	} else {
		node.SetExpanded(!node.IsExpanded())
	}
}

func main() {
	jsonFile, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	var m interface{}
	decoder := json.NewDecoder(jsonFile)
	decoder.UseNumber()
	err = decoder.Decode(&m)
	if err != nil {
		panic(err)
	}
	rootNode, err := CreateNodeRecursive(m)
	rootNode.SetExpanded(true)

	tree := tview.NewTreeView().SetRoot(rootNode).SetCurrentNode(rootNode)
	tree.SetSelectedFunc(selected)
	if err := tview.NewApplication().SetRoot(tree, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
