package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"strings"
	"text/scanner"
	"strconv"
	"regexp"
	"bytes"
	"github.com/caneroj1/stemmer"
)


var IsValidString = regexp.MustCompile("[a-z]+$|[0-9]+$").MatchString

var documentsIndexes = make(map[int]string)
var stemmingMap = make(map[string]map[string]int) // Map of type "STEM" -> {"term1" -> Position in Indexes file}
var indexesMap = make(map[string]map[int]int)     // Map of type "term" -> map{Key -> Value}
var sortedIndexesMap = make(map[string]PostingsList) // Map of type "term" -> []{Posting1 (Key, Value)}

var scan scanner.Scanner

func createInvertedIndex(packageToScan string, err error) {
	checkError(err)
	files, err := ioutil.ReadDir(packageToScan)
	checkError(err)

	for _, file := range files {
		analyzedDocuments := analyzeDocuments(packageToScan, file)
		tokenizeDocuments(analyzedDocuments)
	}
	for key, value := range indexesMap {
		sortedIndexesMap[key] = sortPostingsByTermFrequency(value) // Sort indexes by their term frequencies
	}
	createInvertedIndexFile()
	createStemFile()
	createDocumentsIndexFile()
}

func analyzeDocuments(packageName string, file os.FileInfo) map[int]string {
	result := make(map[int]string)
	re := regexp.MustCompile("[0-9]+")
	fileData, err := ioutil.ReadFile(packageName + "/" + file.Name())
	checkError(err)
	documents := strings.Split(string(fileData), "********************************************")
	for _, document := range documents {
		trimmedDocument := strings.TrimSpace(document)
		title := strings.Split(trimmedDocument, "\n")[0]
		content := strings.Join(strings.Split(trimmedDocument, "\n")[1:], "\n")
		if title != "" {
			docId, err := strconv.ParseInt(re.FindString(title), 10, 64)
			checkError(err)
			result[int(docId)] = content
			documentsIndexes[int(docId)] = content
		}
	}
	return result
}

func tokenizeDocuments(analyzedDocuments map[int] string) {
	for key, value := range analyzedDocuments {
		scan.Init(strings.NewReader(value))
		for token := scan.Scan(); token != scanner.EOF; token = scan.Scan() {
			term := strings.ToLower(scan.TokenText())
			if IsValidString(term) {
				if val, ok := indexesMap[term]; ok {  // If term exists as key
					if value, newOk := val[key]; newOk {  // If Key exists as a key
						val[key] = value + 1
					} else {
						val[key] = 1
					}
					indexesMap[term] = val
				} else {
					tfMap := make(map[int]int)
					tfMap[key] = 1
					indexesMap[term] = tfMap
				}
			}
		}
	}
}


func createStemFile() {
	stemFile, err := os.Create("index/stemmingData")
	checkError(err)

	defer stemFile.Close()

	stemMap := createStemPairsList(stemmingMap)
	for key, value := range stemMap {
		var buffer bytes.Buffer
		buffer.WriteString(key + "->")
		for _, pair := range value {
			buffer.WriteString(fmt.Sprintf("<%s:%d>", pair.Key, pair.Value))
		}
		buffer.WriteString("\n")
		stemFile.WriteString(buffer.String())
	}
}

func createInvertedIndexFile() {
	indexesFile, err := os.Create("index/invertedIndex")
	checkError(err)

	defer indexesFile.Close()

	position := 0
	for key, value := range sortedIndexesMap {
		var buffer bytes.Buffer
		buffer.WriteString(key + "->")
		for _, pair := range value {
			buffer.WriteString(fmt.Sprintf("<%d:%d>", pair.Key, pair.Value))
		}
		buffer.WriteString("\n")
		indexesFile.WriteString(buffer.String())
		stem := stemmer.Stem(key)
		if val, ok := stemmingMap[stem]; ok {
			val[key] = position
		} else {
			stMap := make(map[string]int)
			stMap[key] = position
			stemmingMap[stem] =  stMap
		}
		position += 1
	}
}

func createDocumentsIndexFile() {
	documentsIndexesFile, err := os.Create("index/documentsIndexes")
	checkError(err)

	defer documentsIndexesFile.Close()

	for key, value := range documentsIndexes {
		var buffer bytes.Buffer
		buffer.WriteString(fmt.Sprintf("%d", key) + "->" + value + "\n|\n")
		documentsIndexesFile.WriteString(buffer.String())
	}
}