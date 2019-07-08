package main

// Simulates chunks being stored in vaults on the SAFE network.
// Returns a csv list of vault names and total chunks stored.

import (
	"fmt"
	"math"
	"math/big"
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
const relocations int = 100

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
	// report the starting parameters
	fmt.Print("seed,", nowNanos, "\n")
	fmt.Print("totalNodes,", totalNodes, "\n")
	fmt.Print("totalChunks,", totalChunks, "\n")
	fmt.Print("groupSize,", groupSize, "\n")
	fmt.Print("namingStrategy,", namingStrategy, "\n")
	fmt.Print("spacingStrategy,", spacingStrategy, "\n")
	fmt.Print("relocations,", relocations, "\n")
	fmt.Println()
	// create nodes
	nodes := []Node{}
	for i := 0; i < totalNodes; i++ {
		nodes = addNewNode(nodes)
	}
	// do relocations
	if namingStrategy != "uniform" {
		for i := 0; i < relocations; i++ {
			nodes = removeRandomNode(nodes)
			nodes = addNewNode(nodes)
		}
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
	spacings := getAllSpacings(nodes)
	fmt.Println("\nStandard deviation of spacings:")
	fmt.Println(standardDeviation(spacings))
}

func addNewNode(nodes []Node) []Node {
	// get name that suits the naming strategy
	var nodeName uint64
	// get current names
	names := []uint64{}
	for _, node := range nodes {
		names = append(names, node.Name)
	}
	// generate the next node name
	if namingStrategy == "uniform" {
		progress := float64(len(nodes)) / float64(totalNodes)
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
	return nodes
}

func removeRandomNode(nodes []Node) []Node {
	index := rand.Intn(len(nodes))
	return append(nodes[0:index], nodes[index+1:]...)
}

func nameStr(i uint64) string {
	// hex
	s := strconv.FormatUint(i, 16)
	for len(s) < 16 {
		s = "0" + s
	}
	return s
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
		spacing := getSpacing(thisName, previousName)
		if spacing > maxSpacing {
			maxSpacing = spacing
			minName = previousName
			maxName = thisName
		}
	}
	// check the space between the last node and MaxUint64
	lastName := names[len(names)-1]
	lastSpacing := getSpacing(math.MaxUint64, lastName)
	if lastSpacing > maxSpacing {
		minName = lastName
		maxName = math.MaxUint64
	}
	// adjust the names to be in a more precise gap
	// https://safenetforum.org/t/chunk-distribution-within-sections/29187/34
	minName = minName + (maxSpacing / 3)
	maxName = maxName - (maxSpacing / 3)
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

func standardDeviation(numbers []uint64) int64 {
	avg := average(numbers)
	bigAvg := big.NewInt(0).SetUint64(avg)
	totalDiffs := big.NewInt(0)
	for _, number := range numbers {
		bigNumber := big.NewInt(0).SetUint64(number)
		bigDiff := big.NewInt(0).Sub(bigNumber, bigAvg)
		bigDiffSquared := big.NewInt(0).Mul(bigDiff, bigDiff)
		totalDiffs = big.NewInt(0).Add(totalDiffs, bigDiffSquared)
	}
	bigDeviation := totalDiffs.Div(totalDiffs, big.NewInt(int64(len(numbers)-1)))
	return bigDeviation.Sqrt(bigDeviation).Int64()
}

func average(numbers []uint64) uint64 {
	total := big.NewInt(0)
	for _, number := range numbers {
		bigNumber := big.NewInt(0).SetUint64(number)
		total = total.Add(total, bigNumber)
	}
	bigLen := big.NewInt(int64(len(numbers)))
	bigAverage := total.Div(total, bigLen)
	return bigAverage.Uint64()
}

func getAllSpacings(nodes []Node) []uint64 {
	spacings := []uint64{}
	// spacing from 0 to first name
	firstSpacing := getSpacing(nodes[0].Name, 0)
	spacings = append(spacings, firstSpacing)
	// all other spacing between names
	for i, _ := range nodes {
		if i == 0 {
			continue
		}
		spacing := getSpacing(nodes[i].Name, nodes[i-1].Name)
		spacings = append(spacings, spacing)
	}
	// spacing from last name to MaxUint64
	lastName := nodes[len(nodes)-1].Name
	lastSpacing := getSpacing(math.MaxUint64, lastName)
	spacings = append(spacings, lastSpacing)
	return spacings
}

func getSpacing(bigName, smallName uint64) uint64 {
	var spacing uint64
	if spacingStrategy == "linear" {
		spacing = bigName - smallName
	} else if spacingStrategy == "xordistance" {
		spacing = bigName ^ smallName
	} else {
		panic("unknown spacing strategy")
	}
	return spacing
}
