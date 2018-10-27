// Copyright 2018 Chris Pearce
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"log"
	"sort"
	"time"
)

// Rule represents an antecedent implies consequent rule, and stores its
// support, confidence, and lift.
type Rule struct {
	Antecedent []Item
	Consequent []Item
	Support    float64
	Confidence float64
	Lift       float64
}

// NewRule creates a new rule.
func NewRule(antecedent []Item, consequent []Item, support float64, confidence float64, lift float64) Rule {
	return Rule{
		Antecedent: antecedent,
		Consequent: consequent,
		Support:    support,
		Confidence: confidence,
		Lift:       lift,
	}
}

type ruleTreeNode struct {
	antecedents map[Item]*ruleTreeNode
	consequents map[Item]*ruleTreeNode
	hasRule     bool
	index       int
}

func newRuleTreeNode() *ruleTreeNode {
	return &ruleTreeNode{
		antecedents: make(map[Item]*ruleTreeNode),
		consequents: make(map[Item]*ruleTreeNode),
	}
}

// RuleSet stores a set of rules in a compact tree structure.
type RuleSet struct {
	root  *ruleTreeNode
	rules []Rule
}

// NewRuleSet creates a new RuleSet().
func NewRuleSet() RuleSet {
	return RuleSet{root: newRuleTreeNode()}
}

// Insert inserts a rule into a RuleSet.
func (ruleSet *RuleSet) Insert(rule Rule) {
	parent := ruleSet.root
	for _, item := range rule.Antecedent {
		node, found := parent.antecedents[item]
		if !found {
			node = newRuleTreeNode()
			parent.antecedents[item] = node
		}
		parent = node
	}
	for _, item := range rule.Consequent {
		node, found := parent.consequents[item]
		if !found {
			node = newRuleTreeNode()
			parent.consequents[item] = node
		}
		parent = node
	}
	if !parent.hasRule {
		ruleSet.rules = append(ruleSet.rules, rule)
		parent.hasRule = true
		parent.index = len(ruleSet.rules) - 1
	}
}

// Size returns the number of rules in the set.
func (ruleSet *RuleSet) Size() int {
	return len(ruleSet.rules)
}

// Rules returns the set of rules.
func (ruleSet *RuleSet) Rules() []Rule {
	return ruleSet.rules
}

// Get returns (rule,true) if this RuleSet contains the rule, (nil,false)
// otherwise.
func (ruleSet *RuleSet) Get(rule *Rule) (*Rule, bool) {
	parent := ruleSet.root
	for _, item := range rule.Antecedent {
		node, found := parent.antecedents[item]
		if !found {
			return nil, false
		}
		parent = node
	}
	for _, item := range rule.Consequent {
		node, found := parent.consequents[item]
		if !found {
			return nil, false
		}
		parent = node
	}
	if !parent.hasRule {
		return nil, false
	}
	return &ruleSet.rules[parent.index], true
}

type itemsetWithSupport struct {
	itemset []Item
	support float64
}

func (isl itemsetSupportLookup) Len() int {
	return len(isl.itemsets)
}

func (isl *itemsetSupportLookup) Swap(i, j int) {
	isl.itemsets[i], isl.itemsets[j] = isl.itemsets[j], isl.itemsets[i]
}

func (isl *itemsetSupportLookup) Less(i, j int) bool {
	return itemSliceLess(isl.itemsets[i].itemset, isl.itemsets[j].itemset)
}

type itemsetSupportLookup struct {
	itemsets []itemsetWithSupport
}

func newItemsetSupportLookup() *itemsetSupportLookup {
	return &itemsetSupportLookup{
		itemsets: make([]itemsetWithSupport, 0),
	}
}

func (isl *itemsetSupportLookup) insert(itemset []Item, support float64) {
	isl.itemsets = append(isl.itemsets, itemsetWithSupport{itemset: itemset, support: support})
}

func (isl *itemsetSupportLookup) sort() {
	sort.Sort(isl)
}

func (isl *itemsetSupportLookup) lookup(itemset []Item) float64 {
	idx := sort.Search(len(isl.itemsets), func(idx int) bool {
		return !itemSliceLess(isl.itemsets[idx].itemset, itemset)
	})
	if !itemSliceEquals(isl.itemsets[idx].itemset, itemset) {
		panic("Failed to retrieve itemset support")
	}
	return isl.itemsets[idx].support
}

func createSupportLookup(itemsets []itemsetWithCount, numTransactions int) *itemsetSupportLookup {
	isl := newItemsetSupportLookup()
	f := float64(numTransactions)
	for _, is := range itemsets {
		isl.insert(is.itemset, float64(is.count)/f)
	}
	isl.sort()

	return isl
}

func makeStats(a []Item, c []Item, supportLookup *itemsetSupportLookup) (float64, float64, float64) {
	ac := union(a, c)
	acSup := supportLookup.lookup(ac)
	aSup := supportLookup.lookup(a)
	confidence := acSup / aSup
	cSup := supportLookup.lookup(c)
	lift := acSup / (aSup * cSup)
	return acSup, confidence, lift
}

func itemSliceLess(a, b []Item) bool {
	if len(a) < len(b) {
		return true
	} else if len(a) > len(b) {
		return false
	}
	for idx := range a {
		if a[idx] > b[idx] {
			return false
		}
		if a[idx] < b[idx] {
			return true
		}
	}
	return false
}

func sliceOfItemSliceLessThan(slices [][]Item) func(i, j int) bool {
	return func(i, j int) bool {
		return itemSliceLess(slices[i], slices[j])
	}
}

func sortCandidates(candidates [][]Item) {
	sort.SliceStable(candidates, sliceOfItemSliceLessThan(candidates))
}

func generateRules(itemsets []itemsetWithCount, numTransactions int, minConfidence float64, minLift float64) RuleSet {
	output := NewRuleSet()
	itemsetSupport := createSupportLookup(itemsets, numTransactions)

	lastFeedback := time.Now()

	for index, itemset := range itemsets {
		if time.Since(lastFeedback).Seconds() > 20 {
			numRules := len(itemsets)
			lastFeedback = time.Now()
			percentComplete := int(float64(index)/float64(numRules)*100 + 0.5)
			log.Printf("Progress: %d of %d itemsets processed (%d%%), generated %d rules so far",
				index, len(itemsets), percentComplete, output.Size())
		}
		if len(itemset.itemset) < 2 {
			continue
		}
		// First generation is all possible rules with consequents of size 1.
		candidates := make([][]Item, 0)
		for _, item := range itemset.itemset {
			consequent := []Item{item}
			antecedent := setMinus(itemset.itemset, consequent)
			support, confidence, lift := makeStats(antecedent, consequent, itemsetSupport)
			if confidence < minConfidence {
				continue
			}
			if lift >= minLift {
				output.Insert(NewRule(antecedent, consequent, support, confidence, lift))
			}
			candidates = append(candidates, consequent)
		}
		// Note: candidates should be sorted here.

		// Create subsequent generations by merging consequents which have size-1 items
		// in common in the consequent.
		k := len(itemset.itemset) // size of frequent itemset
		for len(candidates) > 0 && len(candidates[0])+1 < k {
			nextGen := make([][]Item, 0)
			for idx1, c1 := range candidates {
				m := len(c1) // size of consequent.
				for idx2 := idx1 + 1; idx2 < len(candidates); idx2++ {
					if intersectionSize(c1, candidates[idx2]) != m-1 {
						break
					}
					consequent := union(c1, candidates[idx2])
					antecedent := setMinus(itemset.itemset, consequent)

					support, confidence, lift := makeStats(antecedent, consequent, itemsetSupport)
					if confidence < minConfidence {
						continue
					}
					nextGen = append(nextGen, consequent)
					if lift >= minLift {
						rule := NewRule(antecedent, consequent, support, confidence, lift)
						output.Insert(rule)
					}
				}
			}
			candidates = nextGen
			sortCandidates(candidates)
		}
	}

	return output
}
