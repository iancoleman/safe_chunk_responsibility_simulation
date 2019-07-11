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
const totalStored int = 1000000
const groupSize int = 8
const relocations int = 100

// How names for new / relocated vaults are chosen.
// - uniform means vault names are spaced evenly, eg [10, 20, 30, 40]
// - random means vault names are chosen randomly, eg [10, 11, 19, 33]
// - bestfit aims to put the next vault into the largest space
// - quietesthalf aims to put the next vault in the half with the least vaults
// - emptysubsection finds any subsections with no vaults and places randomly
//   in one of them.
const namingStrategy = "bestfit"

// How space between vaults is measured
// - linear uses bigName - smallName
// - xordistance uses bigName ^ smallName
const spacingStrategy = "linear"

// Which units to use for tracking storage
// - chunks counts the number of chunks per vault
// - megabytes counts the number of megabytes per vault since some chunks
//   may be less than 1 MB in size
const storageUnits = "megabytes"

// Structs

type Node struct {
	Name         uint64
	CurrentChunk uint64
	Stored       float64
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
	runTests()
	// set up random numbers
	nowNanos := time.Now().UnixNano()
	rand.Seed(nowNanos)
	// report the starting parameters
	fmt.Print("seed,", nowNanos, "\n")
	fmt.Print("totalNodes,", totalNodes, "\n")
	fmt.Print("totalStored,", totalStored, "\n")
	fmt.Print("groupSize,", groupSize, "\n")
	fmt.Print("namingStrategy,", namingStrategy, "\n")
	fmt.Print("spacingStrategy,", spacingStrategy, "\n")
	fmt.Print("storageUnits,", storageUnits, "\n")
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
	for i := 0; i < totalStored; i++ {
		chunkName := rand.Uint64()
		// set chunk name for sorting
		for j, _ := range nodes {
			nodes[j].CurrentChunk = chunkName
		}
		// find nodes that store this chunk
		sort.Sort(ByXorDistance(nodes))
		// add chunk to the closest group nodes
		for j := 0; j < groupSize; j++ {
			if storageUnits == "chunks" {
				nodes[j].Stored += 1
			} else if storageUnits == "megabytes" {
				mb := getRandomChunkSize()
				nodes[j].Stored += mb
			} else {
				panic("Invalid storage units")
			}
		}
	}
	// report
	sort.Sort(ByNodeName(nodes))
	fmt.Println("vault name," + storageUnits + " stored")
	for _, n := range nodes {
		fmt.Printf("%s,%f\n", nameStr(n.Name), n.Stored)
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
		nodeName = nameForQuietestHalf(names)
	} else if namingStrategy == "emptysubsection" {
		nodeName = nameForEmptySubsection(names)
	} else {
		panic("Invalid naming strategy")
	}
	// add new node to nodes
	node := Node{
		Name:   nodeName,
		Stored: 0,
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
	// get the maximum spacing between existing names
	var maxSpacing uint64
	var minName uint64
	var maxName uint64
	// if this is the first node
	// the name must be between 0 and MaxUint64
	if len(names) == 0 {
		maxSpacing = math.MaxUint64
		minName = 0
		maxName = math.MaxUint64
	} else {
		// find the maximum space between names
		sort.Sort(ByName(names))
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
			maxSpacing = lastSpacing
			minName = lastName
			maxName = math.MaxUint64
		}
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

func nameForQuietestHalf(names []uint64) uint64 {
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

func nameForEmptySubsection(names []uint64) uint64 {
	var searchDepth uint64 = 0
	// find all empty subsections, starting with the biggest subsection
	// and progressively testing smaller subsections.
	// slice of subsections with each subsections being [startName,endName]
	emptySubsections := [][]uint64{}
	for len(emptySubsections) == 0 {
		// generate all subsections for this searchDepth
		subsections := [][]uint64{}
		var totalSubsections uint64 = uint64(1) << searchDepth
		var subsectionSize uint64 = math.MaxUint64 >> searchDepth
		for i := uint64(0); i < totalSubsections; i++ {
			onlyOneSubsection := totalSubsections == 1
			if onlyOneSubsection {
				subsection := []uint64{0, subsectionSize}
				subsections = append(subsections, subsection)
			} else {
				start := i * (subsectionSize + 1)
				end := start + subsectionSize
				subsection := []uint64{start, end}
				subsections = append(subsections, subsection)
			}
		}
		// find any empty subsections
		for _, subsection := range subsections {
			isEmpty := true
			for _, name := range names {
				start := subsection[0]
				end := subsection[1]
				if name >= start && name <= end {
					// if this name is within this subsection the sector is not
					// empty
					isEmpty = false
					break
				}
			}
			if isEmpty {
				emptySubsections = append(emptySubsections, subsection)
			}
		}
		// search deeper
		searchDepth += 1
	}
	// generate a name within an empty subsection
	name := rand.Uint64()
	for true {
		for _, subsection := range emptySubsections {
			if name >= subsection[0] && name <= subsection[1] {
				return name
			}
		}
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

func runTests() {
	// standard deviation
	set := []uint64{5, 5, 5}
	dev := standardDeviation(set)
	if dev != 0 {
		panic("Fail standard deviation all equal")
	}
	set = []uint64{1000, 3000, 7000}
	dev = standardDeviation(set)
	if dev != 3055 {
		panic("Fail standard deviation flooring to int")
	}
	set = []uint64{math.MaxUint64, math.MaxUint64 - 99, math.MaxUint64 - 9999}
	dev = standardDeviation(set)
	if dev != 5744 {
		panic("Fail standard deviation very large numbers")
	}
	// average
	set = []uint64{5, 5, 5}
	avg := average(set)
	if avg != 5 {
		panic("Fail average all equal")
	}
	set = []uint64{1000, 3000, 7000}
	avg = average(set)
	if avg != 3666 {
		panic("Fail average flooring to int")
	}
	set = []uint64{math.MaxUint64, math.MaxUint64 - 99, math.MaxUint64 - 9999}
	avg = average(set)
	if avg != math.MaxUint64-3366 {
		panic("Fail average very large numbers")
	}
	// emptysubsection tests
	emptyA := []uint64{
		0x4000000000000000,
		0x5000000000000000 - 1,
	}
	emptyB := []uint64{
		0xB000000000000000,
		0xC000000000000000 - 1,
	}
	names := []uint64{
		0x0000000000003000,
		0x1000000000003000,
		0x2000000000003000,
		0x3000000000003000,
		//0x4000000000003000,
		0x5000000000003000,
		0x6000000000003000,
		0x7000000000003000,
		0x8000000000003000,
		0x9000000000003000,
		0xA000000000003000,
		//0xB000000000003000,
		0xC000000000003000,
		0xD000000000003000,
		0xE000000000003000,
		0xF000000000003000,
	}
	name := nameForEmptySubsection(names)
	if !((name >= emptyA[0] && name <= emptyA[1]) || (name >= emptyB[0] && name <= emptyB[1])) {
		panic("Name for empty subsection is wrong")
	}
}

func getRandomChunkSize() float64 {
	// returns a chunk size in MB
	// distribution of chunk sizes taken from
	// https://safenetforum.org/t/traffic-sizes-on-the-safe-network/22213
	i := rand.Float64()
	if i < 0.709159 {
		// between 0-100 KB
		return rand.Float64() * 0.1
	} else if i < 0.774634 {
		// between 100-200 KB
		return rand.Float64()*0.1 + 0.1
	} else if i < 0.777539 {
		// between 200-300 KB
		return rand.Float64()*0.1 + 0.2
	} else if i < 0.778139 {
		// between 300-400 KB
		return rand.Float64()*0.1 + 0.3
	} else if i < 0.778459 {
		// between 400-500 KB
		return rand.Float64()*0.1 + 0.4
	} else if i < 0.779100 {
		// between 500-600 KB
		return rand.Float64()*0.1 + 0.5
	} else if i < 0.779342 {
		// between 600-700 KB
		return rand.Float64()*0.1 + 0.6
	} else if i < 0.779450 {
		// between 700-800 KB
		return rand.Float64()*0.1 + 0.7
	} else if i < 0.779588 {
		// between 800-900 KB
		return rand.Float64()*0.1 + 0.8
	} else if i < 0.779730 {
		// between 900-1000 KB
		return rand.Float64()*0.1 + 0.9
	} else {
		// 1000+
		return 1
	}
}
