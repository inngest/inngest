package overlay

import (
	"fmt"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"strings"
)

// ApplyTo will take an overlay and apply its changes to the given YAML
// document.
func (o *Overlay) ApplyTo(root *yaml.Node) error {
	for _, action := range o.Actions {
		var err error
		if action.Remove {
			err = applyRemoveAction(root, action)
		} else {
			err = applyUpdateAction(root, action, &[]string{})
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Overlay) ApplyToStrict(root *yaml.Node) (error, []string) {
	multiError := []string{}
	warnings := []string{}
	for i, action := range o.Actions {
		err := validateSelectorHasAtLeastOneTarget(root, action)
		if err != nil {
			multiError = append(multiError, err.Error())
		}
		if action.Remove {
			err = applyRemoveAction(root, action)
		} else {
			actionWarnings := []string{}
			err = applyUpdateAction(root, action, &actionWarnings)
			for _, warning := range actionWarnings {
				warnings = append(warnings, fmt.Sprintf("update action (%v / %v) target=%s: %s", i+1, len(o.Actions), action.Target, warning))
			}
		}
	}
	if len(multiError) > 0 {
		return fmt.Errorf("error applying overlay (strict): %v", strings.Join(multiError, ",")), warnings
	}
	return nil, warnings
}

func validateSelectorHasAtLeastOneTarget(root *yaml.Node, action Action) error {
	if action.Target == "" {
		return nil
	}

	p, err := yamlpath.NewPath(action.Target)
	if err != nil {
		return err
	}

	nodes, err := p.Find(root)
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		return fmt.Errorf("selector %q did not match any targets", action.Target)
	}

	return nil
}

func applyRemoveAction(root *yaml.Node, action Action) error {
	if action.Target == "" {
		return nil
	}

	idx := newParentIndex(root)

	p, err := yamlpath.NewPath(action.Target)
	if err != nil {
		return err
	}

	nodes, err := p.Find(root)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		removeNode(idx, node)
	}

	return nil
}

func removeNode(idx parentIndex, node *yaml.Node) {
	parent := idx.getParent(node)
	if parent == nil {
		return
	}

	for i, child := range parent.Content {
		if child == node {
			switch parent.Kind {
			case yaml.MappingNode:
				// we have to delete the key too
				parent.Content = append(parent.Content[:i-1], parent.Content[i+1:]...)
				return
			case yaml.SequenceNode:
				parent.Content = append(parent.Content[:i], parent.Content[i+1:]...)
				return
			}
		}
	}
}

func applyUpdateAction(root *yaml.Node, action Action, warnings *[]string) error {
	if action.Target == "" {
		return nil
	}

	if action.Update.IsZero() {
		return nil
	}

	p, err := yamlpath.NewPath(action.Target)
	if err != nil {
		return err
	}

	nodes, err := p.Find(root)
	if err != nil {
		return err
	}

	prior, err := yaml.Marshal(root)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		if err := updateNode(node, action.Update); err != nil {
			return err
		}
	}
	post, err := yaml.Marshal(root)
	if err != nil {
		return err
	}
	if warnings != nil && string(prior) == string(post) {
		*warnings = append(*warnings, "does nothing")
	}

	return nil
}

func updateNode(node *yaml.Node, updateNode yaml.Node) error {
	mergeNode(node, updateNode)
	return nil
}

func mergeNode(node *yaml.Node, merge yaml.Node) {
	if node.Kind != merge.Kind {
		*node = merge
		return
	}
	switch node.Kind {
	default:
		node.Value = merge.Value
	case yaml.MappingNode:
		mergeMappingNode(node, merge)
	case yaml.SequenceNode:
		mergeSequenceNode(node, merge)
	}
}

// mergeMappingNode will perform a shallow merge of the merge node into the main
// node.
func mergeMappingNode(node *yaml.Node, merge yaml.Node) {
NextKey:
	for i := 0; i < len(merge.Content); i += 2 {
		mergeKey := merge.Content[i].Value
		mergeValue := merge.Content[i+1]

		for j := 0; j < len(node.Content); j += 2 {
			nodeKey := node.Content[j].Value
			if nodeKey == mergeKey {
				mergeNode(node.Content[j+1], *mergeValue)
				continue NextKey
			}
		}

		node.Content = append(node.Content, merge.Content[i], mergeValue)
	}
}

// mergeSequenceNode will append the merge node's content to the original node.
func mergeSequenceNode(node *yaml.Node, merge yaml.Node) {
	node.Content = append(node.Content, merge.Content...)
}
