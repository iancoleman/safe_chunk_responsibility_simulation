package main

// Simulates chunks being stored in vaults on the SAFE network.
// Returns a csv list of vault names and total chunks stored.

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"time"
)

// Parameters

const totalNodes int = 100
const totalChunks int = 1000000
const groupSize int = 8
const namingStrategy = "bestfit" // uniform, random, bestfit, quietesthalf
const spacingStrategy = "xordistance" // linear, xordistance

// Structs

type Node struct {
	Name         uint64
	CurrentChunk uint64
	Chunks       int
}

// Sorters

type ByXorDistance []Node

func (a ByXorDistance) Len() int      { return len(a) }
func (a ByXorDistance) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByXorDistance) Less(i, j int) bool {
	return a[i].Name^a[i].CurrentChunk < a[j].Name^a[j].CurrentChunk
}

type ByNodeName []Node

func (a ByNodeName) Len() int           { return len(a) }
func (a ByNodeName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByNodeName) Less(i, j int) bool { return a[i].Name < a[j].Name }

type ByName []uint64

func (a ByName) Len() int           { return len(a) }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return a[i] < a[j] }

// Functions

func main() {
	// set up random numbers
	nowNanos := time.Now().UnixNano()
	rand.Seed(nowNanos)
	fmt.Println("Seed is", nowNanos)
	// create nodes
	nodes := []Node{}
	for i := 0; i < totalNodes; i++ {
		// get name that suits the naming strategy
		var nodeName uint64
		// get current names
		names := []uint64{}
		for _, node := range nodes {
			names = append(names, node.Name)
		}
		// generate the next node name
		if namingStrategy == "uniform" {
			progress := float64(i) / float64(totalNodes)
			nodeName = uint64(float64(math.MaxUint64) * progress)
		} else if namingStrategy == "random" {
			nodeName = rand.Uint64()
		} else if namingStrategy == "bestfit" {
			nodeName = nameForBestFit(names)
		} else if namingStrategy == "quietesthalf" {
			nodeName = nameForQuiestestHalf(names)
		} else {
			panic("Invalid naming strategy")
		}
		// add new node to nodes
		node := Node{
			Name:   nodeName,
			Chunks: 0,
		}
		nodes = append(nodes, node)
	}
	// create chunks
	for i := 0; i < totalChunks; i++ {
		chunkName := rand.Uint64()
		// set chunk name for sorting
		for j, _ := range nodes {
			nodes[j].CurrentChunk = chunkName
		}
		// find nodes that store this chunk
		sort.Sort(ByXorDistance(nodes))
		// add chunk to the closest group nodes
		for j := 0; j < groupSize; j++ {
			nodes[j].Chunks += 1
		}
	}
	// report
	sort.Sort(ByNodeName(nodes))
	fmt.Println("vault name,chunks stored")
	for _, n := range nodes {
		fmt.Printf("%s,%d\n", nameStr(n.Name), n.Chunks)
	}
}

func nameStr(i uint64) string {
	// hex truncated to first seven characters
	s := strconv.FormatUint(i, 16)
	for len(s) < 16 {
		s = "0" + s
	}
	return s[0:7]
}

func nameForBestFit(names []uint64) uint64 {
	name := rand.Uint64()
	// if this is the first node
	// or names are random (ie the largest space is not being targeted)
	// add it now
	if len(names) == 0 {
		return name
	}
	// get the maximum spacing between existing names
	var maxSpacing uint64
	var minName uint64
	var maxName uint64
	sort.Sort(ByName(names))
	// find the maximum space between names
	for i, _ := range names {
		thisName := names[i]
		var previousName uint64 = 0
		if i > 0 {
			previousName = names[i-1]
		}
		var spacing uint64
		if spacingStrategy == "linear" {
			spacing = thisName - previousName
		} else if spacingStrategy == "xordistance" {
			spacing = thisName ^ previousName
		} else {
			panic("unknown spacing strategy")
		}
		if spacing > maxSpacing {
			maxSpacing = spacing
			minName = previousName
			maxName = thisName
		}
	}
	// check the space between the last node and MaxUint64
	lastName := names[len(names)-1]
	lastSpacing := math.MaxUint64 - lastName
	if lastSpacing > maxSpacing {
		minName = lastName
		maxName = math.MaxUint64
	}
	// find a new name within this spacing
	for name <= minName && name >= maxName {
		name = rand.Uint64()
	}
	return name
}

func nameForQuiestestHalf(names []uint64) uint64 {
	// count the vaults in each half
	var halfway uint64 = math.MaxUint64 / 2
	firstHalfVaults := 0
	secondHalfVaults := 0
	for _, name := range names {
		if name < halfway {
			firstHalfVaults = firstHalfVaults + 1
		} else {
			secondHalfVaults = secondHalfVaults + 1
		}
	}
	var minName uint64 = 0
	var maxName uint64 = math.MaxUint64
	if firstHalfVaults > secondHalfVaults {
		minName = halfway
	} else {
		maxName = halfway
	}
	// find a new name within this spacing
	name := rand.Uint64()
	for name <= minName && name >= maxName {
		name = rand.Uint64()
	}
	return name
}
