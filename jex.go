package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/net/html/charset"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type pathToken struct {
	Key     string
	Index   int
	IsIndex bool
}

type nodeMeta struct {
	Path      []pathToken
	Container bool
	Editable  bool
	Deletable bool
}

type objectEntry struct {
	Key   string
	Value interface{}
}

type orderedObject struct {
	Entries []objectEntry
}

func newOrderedObject() *orderedObject {
	return &orderedObject{Entries: make([]objectEntry, 0)}
}

func (o *orderedObject) Len() int {
	if o == nil {
		return 0
	}
	return len(o.Entries)
}

func (o *orderedObject) Get(key string) (interface{}, bool) {
	for _, entry := range o.Entries {
		if entry.Key == key {
			return entry.Value, true
		}
	}
	return nil, false
}

func (o *orderedObject) Set(key string, value interface{}) {
	for idx := range o.Entries {
		if o.Entries[idx].Key == key {
			o.Entries[idx].Value = value
			return
		}
	}
	o.Entries = append(o.Entries, objectEntry{Key: key, Value: value})
}

func (o *orderedObject) Delete(key string) bool {
	for idx := range o.Entries {
		if o.Entries[idx].Key == key {
			o.Entries = append(o.Entries[:idx], o.Entries[idx+1:]...)
			return true
		}
	}
	return false
}

func clonePath(path []pathToken) []pathToken {
	cloned := make([]pathToken, len(path))
	copy(cloned, path)
	return cloned
}

func appendKeyPath(path []pathToken, key string) []pathToken {
	next := clonePath(path)
	return append(next, pathToken{Key: key})
}

func appendIndexPath(path []pathToken, index int) []pathToken {
	next := clonePath(path)
	return append(next, pathToken{Index: index, IsIndex: true})
}

func isScalar(i interface{}) bool {
	switch i.(type) {
	case string, json.Number, bool, nil:
		return true
	default:
		return false
	}
}

func GetColor(i interface{}) tcell.Color {
	switch i.(type) {
	case []interface{}:
		return tcell.ColorGreen
	case *orderedObject:
		return tcell.ColorBlue
	case bool:
		return tcell.ColorOlive
	case string, json.Number:
		return tcell.ColorWhite
	case int, int32, int64, float32, float64:
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
	case *orderedObject:
		return fmt.Sprintf("{ %d }", i.(*orderedObject).Len())
	case bool:
		return fmt.Sprintf("%t", i.(bool))
	case json.Number:
		return string(i.(json.Number))
	case int:
		return strconv.Itoa(i.(int))
	case int32:
		return strconv.FormatInt(int64(i.(int32)), 10)
	case int64:
		return strconv.FormatInt(i.(int64), 10)
	case float32:
		return strconv.FormatFloat(float64(i.(float32)), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(i.(float64), 'f', -1, 64)
	case string:
		return fmt.Sprintf("%q", i.(string))
	case nil:
		return "null"
	default:
		panic(fmt.Sprintf("unsupported data type %T in json!", i))
	}
}

func createDataNode(i interface{}, label string, path []pathToken) (*tview.TreeNode, error) {
	nodeText := AsString(i)
	if label != "" {
		nodeText = fmt.Sprintf("%s: %s", label, nodeText)
	}

	node := tview.NewTreeNode(nodeText).SetColor(GetColor(i))
	meta := &nodeMeta{
		Path:      clonePath(path),
		Container: false,
		Editable:  isScalar(i),
		Deletable: len(path) > 0,
	}

	switch typed := i.(type) {
	case string, json.Number, bool, nil, int, int32, int64, float32, float64:
		// scalar values do not have children
	case []interface{}:
		meta.Container = true
		for idx, entry := range typed {
			childPath := appendIndexPath(path, idx)
			child, err := createDataNode(entry, fmt.Sprintf("[%d]", idx), childPath)
			if err != nil {
				return node, err
			}
			node.AddChild(child)
		}
	case *orderedObject:
		meta.Container = true
		for _, entry := range typed.Entries {
			childPath := appendKeyPath(path, entry.Key)
			value := entry.Value
			child, err := createDataNode(value, entry.Key, childPath)
			if err != nil {
				return node, err
			}
			node.AddChild(child)
		}
	default:
		return node, fmt.Errorf("unsupported type %T for node creation", i)
	}

	node.SetReference(meta)
	node.SetExpanded(false)
	return node, nil
}

func createDataRootNode(i interface{}) (*tview.TreeNode, error) {
	return createDataNode(i, "", nil)
}

type xmlElement struct {
	Name     string
	Attrs    []xml.Attr
	Text     string
	Children []*xmlElement
}

func parseXML(data []byte) (*xmlElement, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charset.NewReaderLabel
	var stack []*xmlElement
	var root *xmlElement

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			current := &xmlElement{
				Name:  t.Name.Local,
				Attrs: t.Attr,
			}
			if len(stack) == 0 {
				root = current
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, current)
			}
			stack = append(stack, current)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text == "" || len(stack) == 0 {
				continue
			}
			current := stack[len(stack)-1]
			if current.Text == "" {
				current.Text = text
			} else {
				current.Text += " " + text
			}
		}
	}

	if root == nil {
		return nil, fmt.Errorf("no XML root element found")
	}
	return root, nil
}

func createXMLNodeRecursive(element *xmlElement) *tview.TreeNode {
	label := fmt.Sprintf("<%s>", element.Name)
	node := tview.NewTreeNode(label).SetColor(tcell.ColorTeal)

	if len(element.Attrs) > 0 || element.Text != "" || len(element.Children) > 0 {
		node.SetReference(element.Name)
	}

	for _, attr := range element.Attrs {
		attrText := fmt.Sprintf("@%s: %q", attr.Name.Local, attr.Value)
		node.AddChild(tview.NewTreeNode(attrText).SetColor(tcell.ColorWhite))
	}

	if element.Text != "" {
		textNode := tview.NewTreeNode(fmt.Sprintf("#text: %q", element.Text)).SetColor(tcell.ColorWhite)
		node.AddChild(textNode)
	}

	for _, child := range element.Children {
		node.AddChild(createXMLNodeRecursive(child))
	}

	node.SetExpanded(false)
	return node
}

func parseJSONValue(decoder *json.Decoder) (interface{}, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	switch typed := token.(type) {
	case json.Delim:
		switch typed {
		case '{':
			obj := newOrderedObject()
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, fmt.Errorf("invalid JSON object key type %T", keyToken)
				}
				value, err := parseJSONValue(decoder)
				if err != nil {
					return nil, err
				}
				obj.Set(key, value)
			}
			endToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			end, ok := endToken.(json.Delim)
			if !ok || end != '}' {
				return nil, fmt.Errorf("invalid JSON object terminator")
			}
			return obj, nil
		case '[':
			items := make([]interface{}, 0)
			for decoder.More() {
				value, err := parseJSONValue(decoder)
				if err != nil {
					return nil, err
				}
				items = append(items, value)
			}
			endToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			end, ok := endToken.(json.Delim)
			if !ok || end != ']' {
				return nil, fmt.Errorf("invalid JSON array terminator")
			}
			return items, nil
		default:
			return nil, fmt.Errorf("unexpected JSON delimiter %q", typed)
		}
	case string, bool, nil, json.Number:
		return typed, nil
	default:
		return nil, fmt.Errorf("unsupported JSON token type %T", typed)
	}
}

func parseJSON(data []byte) (interface{}, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	value, err := parseJSONValue(decoder)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("invalid trailing JSON content")
	}
	return value, nil
}

func parseYAMLNode(node *yaml.Node) (interface{}, error) {
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil, nil
		}
		return parseYAMLNode(node.Content[0])
	case yaml.MappingNode:
		obj := newOrderedObject()
		for idx := 0; idx+1 < len(node.Content); idx += 2 {
			keyNode := node.Content[idx]
			valueNode := node.Content[idx+1]
			key := keyNode.Value
			value, err := parseYAMLNode(valueNode)
			if err != nil {
				return nil, err
			}
			obj.Set(key, value)
		}
		return obj, nil
	case yaml.SequenceNode:
		items := make([]interface{}, 0, len(node.Content))
		for _, child := range node.Content {
			value, err := parseYAMLNode(child)
			if err != nil {
				return nil, err
			}
			items = append(items, value)
		}
		return items, nil
	case yaml.ScalarNode:
		if node.ShortTag() == "!!null" {
			return nil, nil
		}
		if node.ShortTag() == "!!bool" {
			return strconv.ParseBool(node.Value)
		}
		if node.ShortTag() == "!!int" || node.ShortTag() == "!!float" {
			return json.Number(node.Value), nil
		}
		return node.Value, nil
	case yaml.AliasNode:
		if node.Alias == nil {
			return nil, fmt.Errorf("invalid YAML alias")
		}
		return parseYAMLNode(node.Alias)
	default:
		return nil, fmt.Errorf("unsupported YAML node kind %d", node.Kind)
	}
}

func parseYAML(data []byte) (interface{}, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}
	return parseYAMLNode(&node)
}

func buildRootNode(path string) (*tview.TreeNode, string, interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	trimmed := strings.TrimSpace(string(content))

	if ext == ".xml" {
		element, err := parseXML(content)
		if err != nil {
			return nil, "", nil, err
		}
		return createXMLNodeRecursive(element), "XML", nil, nil
	}

	if ext == ".yaml" || ext == ".yml" {
		m, err := parseYAML(content)
		if err != nil {
			return nil, "", nil, err
		}
		rootNode, err := createDataRootNode(m)
		if err != nil {
			return nil, "", nil, err
		}
		return rootNode, "YAML", m, nil
	}

	m, err := parseJSON(content)
	if err != nil {
		if ext != ".json" {
			if strings.HasPrefix(trimmed, "<") {
				element, xmlErr := parseXML(content)
				if xmlErr == nil {
					return createXMLNodeRecursive(element), "XML", nil, nil
				}
			}

			yamlData, yamlErr := parseYAML(content)
			if yamlErr == nil {
				rootNode, nodeErr := createDataRootNode(yamlData)
				if nodeErr == nil {
					return rootNode, "YAML", yamlData, nil
				}
			}
		}
		return nil, "", nil, err
	}
	rootNode, err := createDataRootNode(m)
	if err != nil {
		return nil, "", nil, err
	}
	return rootNode, "JSON", m, nil
}

type editorState struct {
	path     string
	docType  string
	data     interface{}
	dirty    bool
	rootNode *tview.TreeNode
}

func pathsEqual(a, b []pathToken) bool {
	if len(a) != len(b) {
		return false
	}
	for idx := range a {
		if a[idx].IsIndex != b[idx].IsIndex {
			return false
		}
		if a[idx].IsIndex {
			if a[idx].Index != b[idx].Index {
				return false
			}
			continue
		}
		if a[idx].Key != b[idx].Key {
			return false
		}
	}
	return true
}

func findNodeByPath(root *tview.TreeNode, path []pathToken) *tview.TreeNode {
	var found *tview.TreeNode
	walkTree(root, func(node *tview.TreeNode) {
		if found != nil {
			return
		}
		meta, ok := node.GetReference().(*nodeMeta)
		if !ok || meta == nil {
			return
		}
		if pathsEqual(meta.Path, path) {
			found = node
		}
	})
	return found
}

func getValueAtPath(root interface{}, path []pathToken) (interface{}, error) {
	current := root
	for _, token := range path {
		switch typed := current.(type) {
		case *orderedObject:
			if token.IsIndex {
				return nil, fmt.Errorf("expected object key, got index")
			}
			next, ok := typed.Get(token.Key)
			if !ok {
				return nil, fmt.Errorf("missing key %q", token.Key)
			}
			current = next
		case []interface{}:
			if !token.IsIndex {
				return nil, fmt.Errorf("expected array index, got key")
			}
			if token.Index < 0 || token.Index >= len(typed) {
				return nil, fmt.Errorf("index %d out of bounds", token.Index)
			}
			current = typed[token.Index]
		default:
			return nil, fmt.Errorf("path enters scalar type %T", typed)
		}
	}
	return current, nil
}

func setValueAtPath(root *interface{}, path []pathToken, value interface{}) error {
	if len(path) == 0 {
		*root = value
		return nil
	}

	parentPath := path[:len(path)-1]
	last := path[len(path)-1]
	parent, err := getValueAtPath(*root, parentPath)
	if err != nil {
		return err
	}

	switch typed := parent.(type) {
	case *orderedObject:
		if last.IsIndex {
			return fmt.Errorf("cannot use index on object")
		}
		typed.Set(last.Key, value)
		return nil
	case []interface{}:
		if !last.IsIndex {
			return fmt.Errorf("cannot use key on array")
		}
		if last.Index < 0 || last.Index >= len(typed) {
			return fmt.Errorf("index %d out of bounds", last.Index)
		}
		typed[last.Index] = value
		return nil
	default:
		return fmt.Errorf("parent is not a container: %T", parent)
	}
}

func deleteAtPath(root *interface{}, path []pathToken) error {
	if len(path) == 0 {
		return fmt.Errorf("cannot delete root value")
	}

	parentPath := path[:len(path)-1]
	last := path[len(path)-1]
	parent, err := getValueAtPath(*root, parentPath)
	if err != nil {
		return err
	}

	switch typed := parent.(type) {
	case *orderedObject:
		if last.IsIndex {
			return fmt.Errorf("cannot use index on object")
		}
		if _, ok := typed.Get(last.Key); !ok {
			return fmt.Errorf("missing key %q", last.Key)
		}
		typed.Delete(last.Key)
		return nil
	case []interface{}:
		if !last.IsIndex {
			return fmt.Errorf("cannot use key on array")
		}
		if last.Index < 0 || last.Index >= len(typed) {
			return fmt.Errorf("index %d out of bounds", last.Index)
		}
		next := append(typed[:last.Index], typed[last.Index+1:]...)
		return setValueAtPath(root, parentPath, next)
	default:
		return fmt.Errorf("parent is not a container: %T", parent)
	}
}

func addToContainer(root *interface{}, path []pathToken, key string, value interface{}) error {
	container, err := getValueAtPath(*root, path)
	if err != nil {
		return err
	}

	switch typed := container.(type) {
	case *orderedObject:
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("key cannot be empty")
		}
		if _, exists := typed.Get(key); exists {
			return fmt.Errorf("key %q already exists", key)
		}
		typed.Set(key, value)
		return nil
	case []interface{}:
		next := append(typed, value)
		return setValueAtPath(root, path, next)
	default:
		return fmt.Errorf("selected node is not a container")
	}
}

func parseNumber(input string) (json.Number, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("number cannot be empty")
	}

	var tmp interface{}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&tmp); err != nil {
		return "", err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return "", fmt.Errorf("invalid number")
	}

	number, ok := tmp.(json.Number)
	if !ok {
		return "", fmt.Errorf("value is not a number")
	}
	return number, nil
}

func parseValueForExistingType(current interface{}, input string) (interface{}, error) {
	switch current.(type) {
	case string:
		return input, nil
	case bool:
		return strconv.ParseBool(strings.TrimSpace(input))
	case nil:
		trimmed := strings.TrimSpace(input)
		switch trimmed {
		case "null":
			return nil, nil
		case "true", "false":
			return strconv.ParseBool(trimmed)
		default:
			number, err := parseNumber(trimmed)
			if err == nil {
				return number, nil
			}
			return input, nil
		}
	case json.Number, int, int32, int64, float32, float64:
		return parseNumber(input)
	default:
		return nil, fmt.Errorf("type %T is not directly editable", current)
	}
}

func valueTextForInput(v interface{}) string {
	switch typed := v.(type) {
	case string:
		return typed
	case bool:
		return fmt.Sprintf("%t", typed)
	case json.Number:
		return typed.String()
	case nil:
		return "null"
	case int, int32, int64, float32, float64:
		return AsString(typed)
	default:
		return ""
	}
}

func valueForType(typeName, raw string) (interface{}, error) {
	switch typeName {
	case "object":
		return newOrderedObject(), nil
	case "array":
		return []interface{}{}, nil
	case "string":
		return raw, nil
	case "number":
		return parseNumber(raw)
	case "bool":
		return strconv.ParseBool(strings.TrimSpace(raw))
	case "null":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported type %q", typeName)
	}
}

func writeJSONValue(buf *bytes.Buffer, v interface{}, level int) error {
	indent := strings.Repeat("  ", level)
	nextIndent := strings.Repeat("  ", level+1)

	switch typed := v.(type) {
	case *orderedObject:
		if typed.Len() == 0 {
			buf.WriteString("{}")
			return nil
		}
		buf.WriteString("{\n")
		for idx, entry := range typed.Entries {
			keyBytes, _ := json.Marshal(entry.Key)
			buf.WriteString(nextIndent)
			buf.Write(keyBytes)
			buf.WriteString(": ")
			if err := writeJSONValue(buf, entry.Value, level+1); err != nil {
				return err
			}
			if idx < len(typed.Entries)-1 {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
		}
		buf.WriteString(indent)
		buf.WriteString("}")
		return nil
	case []interface{}:
		if len(typed) == 0 {
			buf.WriteString("[]")
			return nil
		}
		buf.WriteString("[\n")
		for idx, item := range typed {
			buf.WriteString(nextIndent)
			if err := writeJSONValue(buf, item, level+1); err != nil {
				return err
			}
			if idx < len(typed)-1 {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
		}
		buf.WriteString(indent)
		buf.WriteString("]")
		return nil
	case string:
		raw, _ := json.Marshal(typed)
		buf.Write(raw)
		return nil
	case json.Number:
		buf.WriteString(typed.String())
		return nil
	case bool:
		if typed {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil
	case nil:
		buf.WriteString("null")
		return nil
	default:
		return fmt.Errorf("unsupported JSON value type %T", typed)
	}
}

func encodeOrderedJSON(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeJSONValue(&buf, data, 0); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func toYAMLNode(v interface{}) (*yaml.Node, error) {
	switch typed := v.(type) {
	case *orderedObject:
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, entry := range typed.Entries {
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: entry.Key}
			valueNode, err := toYAMLNode(entry.Value)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, keyNode, valueNode)
		}
		return node, nil
	case []interface{}:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range typed {
			child, err := toYAMLNode(item)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, child)
		}
		return node, nil
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: typed}, nil
	case json.Number:
		tag := "!!int"
		if strings.ContainsAny(typed.String(), ".eE") {
			tag = "!!float"
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: typed.String()}, nil
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(typed)}, nil
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
	default:
		return nil, fmt.Errorf("unsupported YAML value type %T", typed)
	}
}

func encodeOrderedYAML(data interface{}) ([]byte, error) {
	root, err := toYAMLNode(data)
	if err != nil {
		return nil, err
	}
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
	return yaml.Marshal(doc)
}

func saveDocument(path, docType string, data interface{}) error {
	switch docType {
	case "JSON":
		payload, err := encodeOrderedJSON(data)
		if err != nil {
			return err
		}
		return os.WriteFile(path, payload, 0o644)
	case "YAML":
		payload, err := encodeOrderedYAML(data)
		if err != nil {
			return err
		}
		return os.WriteFile(path, payload, 0o644)
	default:
		return fmt.Errorf("%s documents are read-only", docType)
	}
}

func rebuildDataTree(state *editorState, tree *tview.TreeView, selectedPath []pathToken) error {
	rootNode, err := createDataRootNode(state.data)
	if err != nil {
		return err
	}
	rootNode.SetExpanded(true)
	state.rootNode = rootNode
	tree.SetRoot(rootNode)

	target := findNodeByPath(rootNode, selectedPath)
	if target == nil {
		target = rootNode
	}
	tree.SetCurrentNode(target)
	revealNode(rootNode, target)
	return nil
}

func updateTreeTitle(tree *tview.TreeView, state *editorState) {
	fileName := filepath.Base(state.path)
	dirty := ""
	if state.dirty {
		dirty = " *"
	}
	tree.SetTitle(fmt.Sprintf("[red:yellow]j[black:yellow]e[red:yellow]x[black:yellow]plorer (%s) - %s%s", state.docType, fileName, dirty))
}

func activeMeta(tree *tview.TreeView) (*nodeMeta, bool) {
	current := tree.GetCurrentNode()
	if current == nil {
		return nil, false
	}
	meta, ok := current.GetReference().(*nodeMeta)
	if !ok || meta == nil {
		return nil, false
	}
	return meta, true
}

func centeredPrimitive(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, height, 1, true).
				AddItem(nil, 0, 1, false),
			width,
			1,
			true,
		).
		AddItem(nil, 0, 1, false)
}

func showMessageDialog(app *tview.Application, pages *tview.Pages, focus tview.Primitive, message string) {
	const pageName = "dialog-message"
	pages.RemovePage(pageName)
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			pages.RemovePage(pageName)
			app.SetFocus(focus)
		})
	pages.AddPage(pageName, centeredPrimitive(modal, 60, 8), true, true)
	app.SetFocus(modal)
}

func showTypeDialog(app *tview.Application, pages *tview.Pages, focus tview.Primitive, onSelect func(string)) {
	const pageName = "dialog-type"
	pages.RemovePage(pageName)
	choices := []string{"object", "array", "string", "number", "bool", "null", "cancel"}
	modal := tview.NewModal().
		SetText("Select new node type").
		AddButtons(choices).
		SetDoneFunc(func(_ int, label string) {
			pages.RemovePage(pageName)
			if label == "cancel" {
				app.SetFocus(focus)
				return
			}
			onSelect(label)
		})
	pages.AddPage(pageName, centeredPrimitive(modal, 60, 12), true, true)
	app.SetFocus(modal)
}

func showInputDialog(app *tview.Application, pages *tview.Pages, focus tview.Primitive, title, label, initial string, onSubmit func(string)) {
	const pageName = "dialog-input"
	pages.RemovePage(pageName)

	input := tview.NewInputField().SetLabel(label).SetText(initial)
	form := tview.NewForm().
		AddFormItem(input).
		AddButton("OK", func() {
			pages.RemovePage(pageName)
			onSubmit(input.GetText())
		}).
		AddButton("Cancel", func() {
			pages.RemovePage(pageName)
			app.SetFocus(focus)
		})
	form.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft)
	form.SetButtonsAlign(tview.AlignRight)

	pages.AddPage(pageName, centeredPrimitive(form, 72, 10), true, true)
	app.SetFocus(input)
}

func selected(node *tview.TreeNode) {
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
	children := node.GetChildren()
	if len(children) == 0 {
		return
	}
	// Keep the selected node visible so child expand/collapse has a visible effect.
	node.SetExpanded(true)
	for _, child := range children {
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

func pageStep(tree *tview.TreeView) int {
	_, _, _, height := tree.GetInnerRect()
	if height <= 1 {
		return 1
	}
	return height - 1
}

type searchState struct {
	query            string
	matches          []*tview.TreeNode
	index            int
	highlighted      *tview.TreeNode
	highlightedStyle tcell.Style
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
	s.setActiveMatch(s.current())
}

func (s *searchState) clear() {
	s.setActiveMatch(nil)
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
		s.setActiveMatch(nil)
		return nil
	}
	s.index = (s.index + 1) % len(s.matches)
	match := s.matches[s.index]
	s.setActiveMatch(match)
	return match
}

func (s *searchState) prev() *tview.TreeNode {
	if len(s.matches) == 0 {
		s.setActiveMatch(nil)
		return nil
	}
	s.index--
	if s.index < 0 {
		s.index = len(s.matches) - 1
	}
	match := s.matches[s.index]
	s.setActiveMatch(match)
	return match
}

func (s *searchState) setActiveMatch(node *tview.TreeNode) {
	if s.highlighted != nil {
		s.highlighted.SetTextStyle(s.highlightedStyle)
		s.highlighted = nil
	}
	if node == nil {
		return
	}

	s.highlighted = node
	s.highlightedStyle = node.GetTextStyle()
	// Muted blue-gray background to keep the active search match visible even
	// when the cursor moves elsewhere.
	s.highlighted.SetTextStyle(s.highlightedStyle.Background(tcell.NewRGBColor(32, 45, 58)))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <json-xml-or-yaml-file>\n", os.Args[0])
		os.Exit(1)
	}

	rootNode, docType, data, err := buildRootNode(os.Args[1])
	if err != nil {
		panic(err)
	}
	rootNode.SetExpanded(true)
	state := &editorState{
		path:     os.Args[1],
		docType:  docType,
		data:     data,
		dirty:    false,
		rootNode: rootNode,
	}

	app := tview.NewApplication()
	search := &searchState{}

	tree := tview.NewTreeView().SetRoot(rootNode).SetCurrentNode(rootNode)
	tree.SetSelectedFunc(selected).
		SetBorder(true).
		SetBorderAttributes(tcell.AttrBold).
		SetBorderColor(tcell.ColorYellow)
	updateTreeTitle(tree, state)

	searchBox := tview.NewInputField().
		SetLabel("Search (/) ").
		SetFieldBackgroundColor(tcell.ColorDefault)

	helpBar := tview.NewTextView().
		SetDynamicColors(true).
		SetText("Keys: [yellow]/[white] search  [yellow]Enter[white] toggle  [yellow]n/p[white] next/prev  [yellow]e[white] edit  [yellow]a[white] add  [yellow]D[white] delete  [yellow]s[white] save  [yellow]u/d[white] page up/down  [yellow]x/c[white] expand/collapse children  [yellow]X/C[white] expand/collapse all  [yellow]q[white] quit")

	searchBox.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			search.refresh(state.rootNode, searchBox.GetText())
			updateSearchLabel(searchBox, search)
			if match := search.current(); match != nil {
				revealNode(state.rootNode, match)
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

	pages := tview.NewPages().AddPage("main", flex, true, true)

	refreshAfterMutation := func(selectedPath []pathToken) {
		if err := rebuildDataTree(state, tree, selectedPath); err != nil {
			showMessageDialog(app, pages, tree, err.Error())
			return
		}
		if strings.TrimSpace(search.query) != "" {
			search.refresh(state.rootNode, search.query)
			updateSearchLabel(searchBox, search)
			if match := search.current(); match != nil {
				revealNode(state.rootNode, match)
				tree.SetCurrentNode(match)
			}
		}
		updateTreeTitle(tree, state)
	}

	editCurrent := func() {
		if state.data == nil {
			showMessageDialog(app, pages, tree, "Editing is only available for JSON/YAML.")
			return
		}
		meta, ok := activeMeta(tree)
		if !ok || !meta.Editable {
			showMessageDialog(app, pages, tree, "Select a scalar value to edit.")
			return
		}

		currentValue, err := getValueAtPath(state.data, meta.Path)
		if err != nil {
			showMessageDialog(app, pages, tree, err.Error())
			return
		}

		showInputDialog(app, pages, tree, "Edit Value", "Value: ", valueTextForInput(currentValue), func(input string) {
			nextValue, parseErr := parseValueForExistingType(currentValue, input)
			if parseErr != nil {
				showMessageDialog(app, pages, tree, parseErr.Error())
				return
			}

			if setErr := setValueAtPath(&state.data, meta.Path, nextValue); setErr != nil {
				showMessageDialog(app, pages, tree, setErr.Error())
				return
			}

			state.dirty = true
			refreshAfterMutation(meta.Path)
			app.SetFocus(tree)
		})
	}

	addCurrent := func() {
		if state.data == nil {
			showMessageDialog(app, pages, tree, "Adding is only available for JSON/YAML.")
			return
		}

		meta, ok := activeMeta(tree)
		if !ok || !meta.Container {
			showMessageDialog(app, pages, tree, "Select an object or array to add into.")
			return
		}

		container, err := getValueAtPath(state.data, meta.Path)
		if err != nil {
			showMessageDialog(app, pages, tree, err.Error())
			return
		}

		finalizeAdd := func(key string, typeName string) {
			needsValue := typeName == "string" || typeName == "number" || typeName == "bool"
			applyValue := func(raw string) {
				value, valueErr := valueForType(typeName, raw)
				if valueErr != nil {
					showMessageDialog(app, pages, tree, valueErr.Error())
					return
				}
				if addErr := addToContainer(&state.data, meta.Path, key, value); addErr != nil {
					showMessageDialog(app, pages, tree, addErr.Error())
					return
				}

				state.dirty = true
				nextSelection := meta.Path
				switch typed := container.(type) {
				case *orderedObject:
					_ = typed
					nextSelection = appendKeyPath(meta.Path, key)
				case []interface{}:
					nextSelection = appendIndexPath(meta.Path, len(typed))
				}
				refreshAfterMutation(nextSelection)
				app.SetFocus(tree)
			}

			if needsValue {
				showInputDialog(app, pages, tree, "New "+typeName+" value", "Value: ", "", applyValue)
				return
			}
			applyValue("")
		}

		switch container.(type) {
		case *orderedObject:
			showInputDialog(app, pages, tree, "New object key", "Key: ", "", func(key string) {
				key = strings.TrimSpace(key)
				if key == "" {
					showMessageDialog(app, pages, tree, "Key cannot be empty.")
					return
				}
				showTypeDialog(app, pages, tree, func(typeName string) {
					finalizeAdd(key, typeName)
				})
			})
		case []interface{}:
			showTypeDialog(app, pages, tree, func(typeName string) {
				finalizeAdd("", typeName)
			})
		default:
			showMessageDialog(app, pages, tree, "Selected node is not a container.")
		}
	}

	deleteCurrent := func() {
		if state.data == nil {
			showMessageDialog(app, pages, tree, "Deleting is only available for JSON/YAML.")
			return
		}

		meta, ok := activeMeta(tree)
		if !ok || !meta.Deletable {
			showMessageDialog(app, pages, tree, "Root node cannot be deleted.")
			return
		}

		const pageName = "dialog-confirm-delete"
		pages.RemovePage(pageName)
		modal := tview.NewModal().
			SetText("Delete selected node?").
			AddButtons([]string{"Delete", "Cancel"}).
			SetDoneFunc(func(_ int, label string) {
				pages.RemovePage(pageName)
				if label != "Delete" {
					app.SetFocus(tree)
					return
				}
				if err := deleteAtPath(&state.data, meta.Path); err != nil {
					showMessageDialog(app, pages, tree, err.Error())
					return
				}
				state.dirty = true
				parentPath := meta.Path[:len(meta.Path)-1]
				refreshAfterMutation(parentPath)
				app.SetFocus(tree)
			})
		pages.AddPage(pageName, centeredPrimitive(modal, 60, 8), true, true)
		app.SetFocus(modal)
	}

	saveCurrent := func() {
		if state.data == nil {
			showMessageDialog(app, pages, tree, "Saving changes is only available for JSON/YAML.")
			return
		}
		if err := saveDocument(state.path, state.docType, state.data); err != nil {
			showMessageDialog(app, pages, tree, err.Error())
			return
		}
		state.dirty = false
		updateTreeTitle(tree, state)
		showMessageDialog(app, pages, tree, "Saved.")
	}

	confirmQuit := func() {
		if !state.dirty {
			app.Stop()
			return
		}

		const pageName = "dialog-confirm-quit"
		pages.RemovePage(pageName)
		modal := tview.NewModal().
			SetText("Save changes before quitting?").
			AddButtons([]string{"Save", "Discard", "Cancel"}).
			SetDoneFunc(func(_ int, label string) {
				pages.RemovePage(pageName)
				switch label {
				case "Save":
					if err := saveDocument(state.path, state.docType, state.data); err != nil {
						showMessageDialog(app, pages, tree, err.Error())
						return
					}
					state.dirty = false
					updateTreeTitle(tree, state)
					app.Stop()
				case "Discard":
					app.Stop()
				default:
					app.SetFocus(tree)
				}
			})
		pages.AddPage(pageName, centeredPrimitive(modal, 60, 8), true, true)
		app.SetFocus(modal)
	}

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		frontPage, _ := pages.GetFrontPage()
		if frontPage != "main" {
			return event
		}

		switch event.Key() {
		case tcell.KeyRune:
			if searchBox.HasFocus() {
				return event
			}
			switch event.Rune() {
			case 'q':
				confirmQuit()
				return nil
			case '/':
				if !searchBox.HasFocus() {
					searchBox.SetPlaceholder("tag or value").
						SetPlaceholderTextColor(tcell.ColorRed).
						SetFieldBackgroundColor(tcell.ColorYellow).
						SetFieldTextColor(tcell.ColorBlack)
					app.SetFocus(searchBox)
					return &tcell.EventKey{}
				}
			case 'e':
				editCurrent()
				return nil
			case 'a':
				addCurrent()
				return nil
			case 'D':
				deleteCurrent()
				return nil
			case 's':
				saveCurrent()
				return nil
			case 'n':
				if search.query == "" {
					search.refresh(state.rootNode, searchBox.GetText())
				}
				if match := search.next(); match != nil {
					revealNode(state.rootNode, match)
					tree.SetCurrentNode(match)
				}
				updateSearchLabel(searchBox, search)
				return nil
			case 'N', 'p':
				if search.query == "" {
					search.refresh(state.rootNode, searchBox.GetText())
				}
				if match := search.prev(); match != nil {
					revealNode(state.rootNode, match)
					tree.SetCurrentNode(match)
				}
				updateSearchLabel(searchBox, search)
				return nil
			case 'x':
				setChildrenExpanded(tree.GetCurrentNode(), true)
				return nil
			case 'c':
				setChildrenExpanded(tree.GetCurrentNode(), false)
				return nil
			case 'u':
				tree.Move(-pageStep(tree))
				return nil
			case 'd':
				tree.Move(pageStep(tree))
				return nil
			case 'X':
				setExpandedRecursive(state.rootNode, true)
				return nil
			case 'C':
				setExpandedRecursive(state.rootNode, false)
				state.rootNode.SetExpanded(true)
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
	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
