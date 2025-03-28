package fpgrowth

import "strings"

type itemCount struct {
	counts []int
}

func makeCounts() itemCount {
	return itemCount{counts: make([]int, 0)}
}

func ensureInBounds(slice []int, index int) []int {
	if index < len(slice) {
		return slice
	}
	delta := 1 + index - len(slice)
	return append(slice, make([]int, delta)...)
}

func (ic *itemCount) increment(item Item, count int) {
	idx := int(item)
	ic.counts = ensureInBounds(ic.counts, idx)
	ic.counts[idx] += count
}

func (ic *itemCount) get(item Item) int {
	idx := int(item)
	if idx >= len(ic.counts) {
		return 0
	}
	return ic.counts[idx]
}

// Itemizer converts between a string to an Item type, and vice versa.
// Allows you to ingest a CSV of human readable strings, convert to a more
// efficient representation for rule/itemset generation, and only convert back
// to human readable strings when you come to output the itemsets and rules.
type Itemizer struct {
	strToItem map[string]Item
	itemToStr map[Item]string
	numItems  int
}

// Itemize converts a slice of strings to a slice of Items.
func (it *Itemizer) Itemize(values []string) []Item {
	items := make([]Item, len(values))
	j := 0
	it.forEachItem(values, func(i Item) {
		items[j] = i
		j++
	})
	return items[:j]
}

// ToStor converts an Item back to origianl string representation.
func (it *Itemizer) ToStr(item Item) string {
	s, found := it.itemToStr[item]
	if !found {
		panic("Failed to convert item to string!")
	}
	return s
}

func (it *Itemizer) filter(tokens []string, filter func(Item) bool) []Item {
	items := make([]Item, 0, len(tokens))
	it.forEachItem(tokens, func(i Item) {
		if filter(i) {
			items = append(items, i)
		}
	})
	return items
}

func (it *Itemizer) forEachItem(tokens []string, fn func(Item)) {
	for _, val := range tokens {
		val = strings.TrimSpace(val)
		if len(val) == 0 {
			continue
		}
		itemID, found := it.strToItem[val]
		if !found {
			it.numItems++
			itemID = Item(it.numItems)
			it.strToItem[val] = itemID
			it.itemToStr[itemID] = val
		}
		fn(itemID)
	}
}

func (it *Itemizer) cmp(a Item, b Item) bool {
	return it.itemToStr[a] < it.itemToStr[b]
}

func newItemizer() Itemizer {
	return Itemizer{
		strToItem: make(map[string]Item),
		itemToStr: make(map[Item]string),
		numItems:  0,
	}
}
