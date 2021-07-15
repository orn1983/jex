package main

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/rivo/tview"
	"github.com/gdamore/tcell"
)

func GetColor(i interface{}) tcell.Color {
	switch i.(type) {
	case []interface{}:
		return tcell.ColorGreen
	case map[string]interface{}:
		return tcell.ColorBlue
	case bool:
		return tcell.ColorOlive
	case string, json.Number:
		return tcell.ColorWhite
	case nil:
		return tcell.ColorPurple
	default:
		return tcell.ColorWhite
	}
}

func AsString(i interface{}) string {
	switch i.(type) {
	case []interface{}:
		return fmt.Sprintf("[ %d ]", len(i.([]interface{})))
	case map[string]interface{}:
		return fmt.Sprintf("{ %d }", len(i.(map[string]interface{})))
	case bool:
		return fmt.Sprintf("%t", i.(bool))
	case json.Number:
		return string(i.(json.Number))
	case string:
		return fmt.Sprintf("%q", i.(string))
	case nil:
		return "null"
	default:
		panic(fmt.Sprintf("unsupported data type %T in json!", i))
	}
}

func CreateNodeRecursive(i interface{}) (*tview.TreeNode, error) {
	var err error
	nodename := AsString(i)
	node := tview.NewTreeNode(nodename).SetColor(GetColor(i))
	switch i.(type) {
	// These cases cannot have children so let's just get them out of the way
	case string, json.Number, bool, nil:
		// Do nothing
	case []interface{}:
		node.SetReference(nodename)
		for _, entry := range i.([]interface{}) {
			child, err := CreateNodeRecursive(entry)
			if err != nil {
				return node, err
			}
			node.AddChild(child)
		}
	case map[string]interface{}:
		node.SetReference(nodename)
		var childNode *tview.TreeNode
		for k, v := range i.(map[string]interface{}) {
			switch v.(type) {
				case string, json.Number, bool, nil:
					// We really have two nodes, but the value cannot have children
					// so we flatten the nodes into one
					mergedNodeDescription  := fmt.Sprintf("%s: %s", k, AsString(v))
					childNode = tview.NewTreeNode(mergedNodeDescription).SetColor(GetColor(v))
				case []interface{}, map[string]interface{}:
					var err error
					childNode, err = CreateNodeRecursive(v)
					if err != nil {
						return node, err
					}
					childNode.SetText(fmt.Sprintf("%s: %s", k, childNode.GetText()))
				default:
					return node, fmt.Errorf("unsupported data type %T in json!", v)
			}
			node.AddChild(childNode)
		}
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
	if err != nil {
		panic(err)
	}
	rootNode.SetExpanded(true)

	app := tview.NewApplication()

	tree := tview.NewTreeView().SetRoot(rootNode).SetCurrentNode(rootNode)
	tree.SetSelectedFunc(selected).
		SetBorder(true).
		SetBorderAttributes(tcell.AttrBold).
		SetBorderColor(tcell.ColorYellow).
		SetTitle("[red:yellow]j[black:yellow]son[red:yellow] ex[black:yellow]plorer")

	searchBox := tview.NewInputField().
		SetLabel("Search (/) ").
		SetFieldBackgroundColor(tcell.ColorDefault)

	searchBox.SetDoneFunc(func(key tcell.Key) {
			switch key {
			case tcell.KeyEnter:
				searchBox.SetFieldBackgroundColor(tcell.ColorDefault)
				searchBox.SetFieldTextColor(tcell.ColorWhite)
				app.SetFocus(tree)
			case tcell.KeyEscape:
				searchBox.SetText("").
					SetFieldBackgroundColor(tcell.ColorDefault).
					SetPlaceholder("")
				app.SetFocus(tree)
			default:
				fmt.Printf("got key %s", key)
			}
		})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tree, 0, 1, true).
		AddItem(searchBox, 1, 0, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				app.Stop()
			case '/':
				if !searchBox.HasFocus() {
					searchBox.SetPlaceholder("tag or value").
						SetPlaceholderTextColor(tcell.ColorRed).
						SetFieldBackgroundColor(tcell.ColorYellow).
						SetFieldTextColor(tcell.ColorBlack)
					app.SetFocus(searchBox)
					return &tcell.EventKey{}
				}
			}
		case tcell.KeyEsc:
		  if searchBox.GetText() != "" {
			 searchBox.SetText("").
				SetPlaceholder("")
		  }
		}
		return event
	})
	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
