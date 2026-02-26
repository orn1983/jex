package main

import (
	"encoding/json"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os"
	"strings"
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
				mergedNodeDescription := fmt.Sprintf("%s: %s", k, AsString(v))
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

func walkTree(node *tview.TreeNode, fn func(*tview.TreeNode)) {
	if node == nil {
		return
	}
	fn(node)
	for _, child := range node.GetChildren() {
		walkTree(child, fn)
	}
}

func setExpandedRecursive(node *tview.TreeNode, expanded bool) {
	if node == nil {
		return
	}
	node.SetExpanded(expanded)
	for _, child := range node.GetChildren() {
		setExpandedRecursive(child, expanded)
	}
}

func setChildrenExpanded(node *tview.TreeNode, expanded bool) {
	if node == nil {
		return
	}
	for _, child := range node.GetChildren() {
		setExpandedRecursive(child, expanded)
	}
}

func findPath(root *tview.TreeNode, target *tview.TreeNode, path *[]*tview.TreeNode) bool {
	if root == nil {
		return false
	}

	*path = append(*path, root)
	if root == target {
		return true
	}

	for _, child := range root.GetChildren() {
		if findPath(child, target, path) {
			return true
		}
	}

	*path = (*path)[:len(*path)-1]
	return false
}

func revealNode(root *tview.TreeNode, node *tview.TreeNode) {
	var path []*tview.TreeNode
	if !findPath(root, node, &path) {
		return
	}
	for _, n := range path {
		n.SetExpanded(true)
	}
}

func collectMatches(root *tview.TreeNode, query string) []*tview.TreeNode {
	var matches []*tview.TreeNode
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return matches
	}
	walkTree(root, func(node *tview.TreeNode) {
		if strings.Contains(strings.ToLower(node.GetText()), query) {
			matches = append(matches, node)
		}
	})
	return matches
}

type searchState struct {
	query   string
	matches []*tview.TreeNode
	index   int
}

func (s *searchState) position() (int, int) {
	total := len(s.matches)
	if total == 0 {
		return 0, 0
	}
	return s.index + 1, total
}

func updateSearchLabel(searchBox *tview.InputField, s *searchState) {
	current, total := s.position()
	if strings.TrimSpace(s.query) == "" {
		searchBox.SetLabel("Search (/) ")
		return
	}
	searchBox.SetLabel(fmt.Sprintf("Search (/) [%d/%d] ", current, total))
}

func (s *searchState) refresh(root *tview.TreeNode, query string) {
	s.query = query
	s.matches = collectMatches(root, query)
	s.index = 0
}

func (s *searchState) clear() {
	s.query = ""
	s.matches = nil
	s.index = 0
}

func (s *searchState) current() *tview.TreeNode {
	if len(s.matches) == 0 {
		return nil
	}
	return s.matches[s.index]
}

func (s *searchState) next() *tview.TreeNode {
	if len(s.matches) == 0 {
		return nil
	}
	s.index = (s.index + 1) % len(s.matches)
	return s.matches[s.index]
}

func (s *searchState) prev() *tview.TreeNode {
	if len(s.matches) == 0 {
		return nil
	}
	s.index--
	if s.index < 0 {
		s.index = len(s.matches) - 1
	}
	return s.matches[s.index]
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <json-file>\n", os.Args[0])
		os.Exit(1)
	}

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
	search := &searchState{}

	tree := tview.NewTreeView().SetRoot(rootNode).SetCurrentNode(rootNode)
	tree.SetSelectedFunc(selected).
		SetBorder(true).
		SetBorderAttributes(tcell.AttrBold).
		SetBorderColor(tcell.ColorYellow).
		SetTitle("[red:yellow]j[black:yellow]son[red:yellow] ex[black:yellow]plorer")

	searchBox := tview.NewInputField().
		SetLabel("Search (/) ").
		SetFieldBackgroundColor(tcell.ColorDefault)

	helpBar := tview.NewTextView().
		SetDynamicColors(true).
		SetText("Keys: [yellow]/[white] search  [yellow]Enter[white] toggle  [yellow]n/p[white] next/prev  [yellow]e/c[white] expand/collapse children  [yellow]E/C[white] expand/collapse all  [yellow]q[white] quit")

	searchBox.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			search.refresh(rootNode, searchBox.GetText())
			updateSearchLabel(searchBox, search)
			if match := search.current(); match != nil {
				revealNode(rootNode, match)
				tree.SetCurrentNode(match)
				searchBox.SetFieldBackgroundColor(tcell.ColorDefault)
				searchBox.SetFieldTextColor(tcell.ColorWhite)
				app.SetFocus(tree)
			} else {
				searchBox.SetFieldBackgroundColor(tcell.ColorMaroon).
					SetFieldTextColor(tcell.ColorWhite)
			}
		case tcell.KeyEscape:
			search.clear()
			updateSearchLabel(searchBox, search)
			searchBox.SetText("").
				SetFieldBackgroundColor(tcell.ColorDefault).
				SetPlaceholder("")
			app.SetFocus(tree)
		}
	})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tree, 0, 1, true).
		AddItem(searchBox, 1, 0, false).
		AddItem(helpBar, 1, 0, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			if searchBox.HasFocus() {
				return event
			}
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
			case 'n':
				if search.query == "" {
					search.refresh(rootNode, searchBox.GetText())
				}
				if match := search.next(); match != nil {
					revealNode(rootNode, match)
					tree.SetCurrentNode(match)
				}
				updateSearchLabel(searchBox, search)
				return nil
			case 'N', 'p':
				if search.query == "" {
					search.refresh(rootNode, searchBox.GetText())
				}
				if match := search.prev(); match != nil {
					revealNode(rootNode, match)
					tree.SetCurrentNode(match)
				}
				updateSearchLabel(searchBox, search)
				return nil
			case 'e':
				setChildrenExpanded(tree.GetCurrentNode(), true)
				return nil
			case 'c':
				setChildrenExpanded(tree.GetCurrentNode(), false)
				return nil
			case 'E':
				setExpandedRecursive(rootNode, true)
				return nil
			case 'C':
				setExpandedRecursive(rootNode, false)
				rootNode.SetExpanded(true)
				return nil
			}
		case tcell.KeyEsc:
			if searchBox.GetText() != "" {
				searchBox.SetText("").
					SetPlaceholder("")
				search.clear()
				updateSearchLabel(searchBox, search)
			}
		}
		return event
	})
	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
