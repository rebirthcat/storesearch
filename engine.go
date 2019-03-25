// Copyright 2013 Hui Chen
// Copyright 2016 ego authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

/*

Package riot is riot engine
*/
package storesearch

import (
	"container/heap"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-ego/gse"
	"github.com/go-ego/murmur"
	"storesearch/core"
	"storesearch/types"
)

const (
	// Version get the riot version
	Version string = "v0.10.0.425, Danube River!"

	// NumNanosecondsInAMillisecond nano-seconds in a milli-second num
	NumNanosecondsInAMillisecond = 1000000
	// StoreFilePrefix persistent store file prefix
	StoreFilePrefix = "riot"

	// DefaultPath default db path
	DefaultPath = "./riot-index"
)

// GetVersion get the riot version
func GetVersion() string {
	return Version
}

// Engine initialize the engine
type Engine struct {

	// 记录初始化参数
	initOptions types.EngineOpts
	initialized bool

	indexers   []*core.Indexer
	segmenter  gse.Segmenter
	loaded     bool
	stopTokens StopTokens
	// 建立索引器使用的通信通道
	segmenterChan         chan segmenterReq
	indexerAddDocChans    []chan indexerAddDocReq
	indexerRemoveDocChans []chan indexerRemoveDocReq

	// 建立排序器使用的通信通道
	indexerLookupChans []chan indexerLookupReq
}

type rankerReturnReq struct {
	docs    []*types.ScoredID
	numDocs int
}

// Indexer initialize the indexer channel
func (engine *Engine) Indexer(options types.EngineOpts) {
	engine.indexerAddDocChans = make(
		[]chan indexerAddDocReq, options.NumShards)

	engine.indexerRemoveDocChans = make(
		[]chan indexerRemoveDocReq, options.NumShards)

	engine.indexerLookupChans = make(
		[]chan indexerLookupReq, options.NumShards)

	for shard := 0; shard < options.NumShards; shard++ {
		engine.indexerAddDocChans[shard] = make(
			chan indexerAddDocReq, options.IndexerBufLen)

		engine.indexerRemoveDocChans[shard] = make(
			chan indexerRemoveDocReq, options.IndexerBufLen)

		engine.indexerLookupChans[shard] = make(
			chan indexerLookupReq, options.IndexerBufLen)
	}
}



// WithGse Using user defined segmenter
// If using a not nil segmenter and the dictionary is loaded,
// the `opt.GseDict` will be ignore.
func (engine *Engine) WithGse(segmenter gse.Segmenter) *Engine {
	if engine.initialized {
		log.Fatal(`Do not re-initialize the engine, 
			WithGse should call before initialize the engine.`)
	}

	engine.segmenter = segmenter
	engine.loaded = true
	return engine
}

func (engine *Engine) initDef(options types.EngineOpts) types.EngineOpts {
	if options.GseDict == "" && !options.NotUseGse && !engine.loaded {
		log.Printf("Dictionary file path is empty, load the default dictionary file.")
		options.GseDict = "zh"
	}

	if  options.StoreFolder == "" {
		log.Printf("Store file path is empty, use default folder path.")
		options.StoreFolder = DefaultPath
		// os.MkdirAll(DefaultPath, 0777)
	}

	return options
}

// Init initialize the engine
func (engine *Engine) Init(options types.EngineOpts) {
	// 初始化初始参数
	if engine.initialized {
		log.Fatal("Do not re-initialize the engine.")
	}
	options = engine.initDef(options)

	options.Init()
	engine.initOptions = options
	engine.initialized = true

	if !options.NotUseGse {
		if !engine.loaded {
			// 载入分词器词典
			engine.segmenter.LoadDict(options.GseDict)
			engine.loaded = true
		}
		// 初始化停用词
		engine.stopTokens.Init(options.StopTokenFile)
	}

	// 初始化索引器
	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		engine.indexers = append(engine.indexers, &core.Indexer{})
		dbPathForwardIndex := engine.initOptions.StoreFolder + "/" +
			StoreFilePrefix + ".forwardindex." + strconv.Itoa(shard)
		dbPathReverseIndex := engine.initOptions.StoreFolder + "/" +
			StoreFilePrefix + ".reversedindex." + strconv.Itoa(shard)
		engine.indexers[shard].Init(shard, engine.initOptions.StoreIndexBufLen, dbPathForwardIndex, dbPathReverseIndex,
			engine.initOptions.StoreEngine, engine.initOptions.DocNumber, engine.initOptions.TokenNumber, *engine.initOptions.IndexerOpts)
	}

	// 每个索引器内部持久化恢复过程
	if options.Recover {
		if options.StoreConcurrent {
			engine.StoreRecoverConcurrent()
		}else {
			engine.StoreRecoverOneByOne()
		}
	}
	log.Println("index recover finish")
	log.Printf("document number is %v", engine.NumDocsIndexed())
	log.Printf("tokens number is %v", engine.NumTokensAdded())
	// 初始化分词器通道
	engine.segmenterChan = make(
		chan segmenterReq, options.NumGseThreads)

	// 初始化索引器通道
	engine.Indexer(options)


	// 启动分词器
	for iThread := 0; iThread < options.NumGseThreads; iThread++ {
		go engine.segmenterWorker()
	}

	// 启动索引器以及各自对应的持久化工作协程
	for shard := 0; shard < options.NumShards; shard++ {
		go engine.indexerAddDoc(shard)
		go engine.indexerRemoveDoc(shard)
		go engine.indexers[shard].StoreUpdateForWardIndexWorker()
		go engine.indexers[shard].StoreUpdateReverseIndexWorker()

		for i := 0; i < options.NumIndexerThreads; i++ {
			go engine.indexerLookup(shard)
		}
	}
}

func (engine *Engine) StoreRecoverOneByOne() {

	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		engine.indexers[shard].StoreRecoverReverseIndex(engine.initOptions.TokenNumber, nil)
		engine.indexers[shard].StoreRecoverForwardIndex(engine.initOptions.DocNumber, nil)
		engine.indexers[shard].StoreUpdateBegin()
	}
}


func (engine *Engine) StoreRecoverConcurrent() {
	wg := sync.WaitGroup{}
	wg.Add(engine.initOptions.NumShards * 2)
	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		go engine.indexers[shard].StoreRecoverForwardIndex(engine.initOptions.DocNumber, &wg)
		go engine.indexers[shard].StoreRecoverReverseIndex(engine.initOptions.TokenNumber, &wg)
		engine.indexers[shard].StoreUpdateBegin()
	}
	wg.Wait()
}

func (engine *Engine) StoreReBuildOneByOne() {
	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		engine.indexers[shard].StoreReverseIndexOneTime(nil)
		engine.indexers[shard].StoreForwardIndexOneTime(nil)
		engine.indexers[shard].StoreUpdateBegin()
	}
}


func (engine *Engine) StoreReBuildConcurrent() {
	wg := sync.WaitGroup{}
	wg.Add(engine.initOptions.NumShards*2)
	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		go engine.indexers[shard].StoreForwardIndexOneTime(&wg)
		go engine.indexers[shard].StoreReverseIndexOneTime(&wg)
		engine.indexers[shard].StoreUpdateBegin()
	}
	wg.Wait()
}

// IndexDoc add the document to the index
// 将文档加入索引
//
// 输入参数：
//  docId	      标识文档编号，必须唯一，docId == 0 表示非法文档（用于强制刷新索引），[1, +oo) 表示合法文档
//  data	      见 DocIndexData 注释
//  forceUpdate 是否强制刷新 cache，如果设为 true，则尽快添加到索引，否则等待 cache 满之后一次全量添加
//
// 注意：
//      1. 这个函数是线程安全的，请尽可能并发调用以提高索引速度
//      2. 这个函数调用是非同步的，也就是说在函数返回时有可能文档还没有加入索引中，因此
//         如果立刻调用Search可能无法查询到这个文档。强制刷新索引请调用FlushIndex函数。
func (engine *Engine) IndexDoc(docId string, data types.DocData,
	forceUpdate ...bool) error {
	engine.index(docId, data, forceUpdate...)
	return nil
}

// Index add the document to the index
func (engine *Engine) index(docId string, data types.DocData,
	forceUpdate ...bool) {

	var force bool
	if len(forceUpdate) > 0 {
		force = forceUpdate[0]
	}

	// data.Tokens
	engine.internalIndexDoc(docId, data, force)
}

func (engine *Engine) internalIndexDoc(docId string, data types.DocData,
	forceUpdate bool) {

	if !engine.initialized {
		log.Fatal("The engine must be initialized first.")
	}

	hash := murmur.Sum32(fmt.Sprintf("%s%s", docId, data.Content))
	engine.segmenterChan <- segmenterReq{
		docId: docId, hash: hash, data: data, forceUpdate: forceUpdate}
}

// RemoveDoc remove the document from the index
// 将文档从索引中删除
//
// 输入参数：
//  docId	      标识文档编号，必须唯一，docId == 0 表示非法文档（用于强制刷新索引），[1, +oo) 表示合法文档
//  forceUpdate 是否强制刷新 cache，如果设为 true，则尽快删除索引，否则等待 cache 满之后一次全量删除
//
// 注意：
//      1. 这个函数是线程安全的，请尽可能并发调用以提高索引速度
//      2. 这个函数调用是非同步的，也就是说在函数返回时有可能文档还没有加入索引中，因此
//         如果立刻调用 Search 可能无法查询到这个文档。强制刷新索引请调用 FlushIndex 函数。
func (engine *Engine) RemoveDoc(docId string, forceUpdate ...bool) error {
	var force bool
	if len(forceUpdate) > 0 {
		force = forceUpdate[0]
	}

	if !engine.initialized {
		log.Fatal("The engine must be initialized first.")
	}

	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		engine.indexerRemoveDocChans[shard] <- indexerRemoveDocReq{
			docId: docId, forceUpdate: force}
	}
	return nil
}

// Segment get the word segmentation result of the text
// 获取文本的分词结果, 只分词与过滤弃用词
func (engine *Engine) Segment(content string) (keywords []string) {

	var segments []string
	hmm := engine.initOptions.Hmm

	if engine.initOptions.GseMode {
		segments = engine.segmenter.CutSearch(content, hmm)
	} else {
		segments = engine.segmenter.Cut(content, hmm)
	}

	for _, token := range segments {
		if !engine.stopTokens.IsStopToken(token) {
			keywords = append(keywords, token)
		}
	}

	return
}

// Tokens get the engine tokens
func (engine *Engine) Tokens(request types.SearchReq) (tokens []string) {
	// 收集关键词
	// tokens := []string{}
	if request.Text != "" {
		reqText := strings.ToLower(request.Text)
		if engine.initOptions.NotUseGse {
			tokens = strings.Split(reqText, " ")
		} else {
			// querySegments := engine.segmenter.Segment([]byte(reqText))
			// tokens = engine.Tokens([]byte(reqText))
			tokens = engine.Segment(reqText)
		}

		// 叠加 tokens
		for _, t := range request.Tokens {
			tokens = append(tokens, t)
		}

		return
	}

	for _, t := range request.Tokens {
		tokens = append(tokens, t)
	}
	return
}



// NotTimeOut not set engine timeout
func (engine *Engine) NotTimeOut(request types.SearchReq, rankerReturnChan chan rankerReturnReq) (
	rankOutID [][]*types.ScoredID, numDocs int) {
	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		rankerOutput := <-rankerReturnChan
		if !request.CountDocsOnly {
			if rankerOutput.docs != nil {
				rankOutID = append(rankOutID, rankerOutput.docs)
				//rankOutID[shard] = rankerOutput.docs
			}
		}
		numDocs += rankerOutput.numDocs
	}
	return
}

// TimeOut set engine timeout
func (engine *Engine) TimeOut(request types.SearchReq,
	rankerReturnChan chan rankerReturnReq) (
	rankOutID [][]*types.ScoredID, numDocs int, isTimeout bool) {

	deadline := time.Now().Add(time.Nanosecond *
		time.Duration(NumNanosecondsInAMillisecond*request.Timeout))
	rankOutID = make([][]*types.ScoredID, engine.initOptions.NumShards)
	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		select {
		case rankerOutput := <-rankerReturnChan:
			if !request.CountDocsOnly {
				if rankerOutput.docs != nil {
					rankOutID = append(rankOutID, rankerOutput.docs)
				}
			}
			numDocs += rankerOutput.numDocs
		case <-time.After(deadline.Sub(time.Now())):
			isTimeout = true
			break
		}
	}
	return
}

// RankID rank docs by types.ScoredIDs
func (engine *Engine) RankID(request types.SearchReq, tokens []string, rankerReturnChan chan rankerReturnReq) (
	output types.SearchID) {
	// 从通信通道读取排序器的输出
	numDocs := 0
	rankOutputArr := [][]*types.ScoredID{}

	//**********/ begin
	timeout := request.Timeout
	isTimeout := false
	if timeout <= 0 {
		// 不设置超时
		rankOutputArr, numDocs = engine.NotTimeOut(request, rankerReturnChan)
	} else {
		// 设置超时
		rankOutputArr, numDocs, isTimeout = engine.TimeOut(request, rankerReturnChan)
	}
	// 仅当 CountDocsOnly 为 false 时才充填 output.Docs
	if request.CountDocsOnly {
		output.Tokens = tokens
		output.NumDocs = numDocs
		output.Timeout = isTimeout
		freeObjToPool(rankOutputArr)
		return
	}

	// 再排序 使用堆排序
	//定义结果数组
	var res []types.ScoredID
	if request.MaxOutputNum == 0 || request.MaxOutputNum > numDocs {
		res = make([]types.ScoredID, numDocs)
	} else {
		res = make([]types.ScoredID, request.MaxOutputNum)
	}

	h := &types.NodeHeap{
		Arr:        []types.HeapNode{},
		IsSmallTop: request.OrderReverse,
	}
	numshard := len(rankOutputArr)
	for i := 0; i < numshard; i++ {
		if len(rankOutputArr[i]) > 0 {
			node := types.HeapNode{
				ScoreObj:     rankOutputArr[i][0],
				ShareNum:     i,
				IndexPointer: 0,
			}
			heap.Push(h, node)
		}
	}
	index := 0
	for index < len(res) {
		n := heap.Pop(h)
		if n == nil {
			break
		}
		node := n.(types.HeapNode)
		res[index] = *node.ScoreObj
		index++
		if node.IndexPointer+1 < len(rankOutputArr[node.ShareNum]) {
			node.IndexPointer++
			node.ScoreObj = rankOutputArr[node.ShareNum][node.IndexPointer]
			heap.Push(h, node)
		}
	}

	// 准备输出
	output.Docs = res
	output.Tokens = tokens
	output.NumDocs = index
	output.Timeout = isTimeout
	freeObjToPool(rankOutputArr)
	return
}

func freeObjToPool(objarr [][]*types.ScoredID) {
	if objarr == nil || len(objarr) == 0 {
		return
	}
	for i := 0; i < len(objarr); i++ {
		for _, obj := range objarr[i] {
			obj.Scores = 0
			obj.TokenProximity = 0
			obj.TokenLocs = nil
			obj.TokenSnippetLocs = nil
			types.ScoreIDPool.Put(obj)
		}
	}
	return
}


// Search find the document that satisfies the search criteria.
// This function is thread safe
// 查找满足搜索条件的文档，此函数线程安全
func (engine *Engine) Search(request types.SearchReq) (output types.SearchID) {
	if !engine.initialized {
		log.Fatal("The engine must be initialized first.")
	}

	tokens := engine.Tokens(request)

	// 建立排序器返回的通信通道
	rankerReturnChan := make(
		chan rankerReturnReq, engine.initOptions.NumShards)

	// 生成查找请求
	lookupRequest := indexerLookupReq{
		countDocsOnly:    request.CountDocsOnly,
		tokens:           tokens,
		labels:           request.Labels,
		docIds:           request.DocIds,
		disScoreCriteria: request.DistScoreCriteria,
		geoFilter:  	  request.GeoFilter,
		rankerReturnChan: rankerReturnChan,
		orderless:        request.Orderless,
	}

	// 向索引器发送查找请求
	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		engine.indexerLookupChans[shard] <- lookupRequest
	}

	output = engine.RankID(request, tokens, rankerReturnChan)

	return
}

// Flush block wait until all indexes are added
// 阻塞等待直到所有索引添加完毕
func (engine *Engine) Flush() {
	wg := sync.WaitGroup{}
	wg.Add(engine.initOptions.NumShards)
	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		go engine.indexers[shard].Flush(&wg)
	}
	wg.Wait()
}

// FlushIndex block wait until all indexes are added
// 阻塞等待直到所有索引添加完毕
func (engine *Engine) FlushIndex() {
	engine.Flush()
}

// Close close the engine
// 关闭引擎
func (engine *Engine) Close() {
	engine.Flush()
	//time.Sleep(time.Second*300)
	for i, indexer := range engine.indexers {
		log.Printf("indexer %v :document number is %v,reverse index table len is %v", i, indexer.GetNumDocs(), indexer.GetTableLen())
	}
	for _, indexer := range engine.indexers {
		dbf := indexer.GetForwardIndexDB()
		if dbf != nil {
			dbf.Close()
		}
		dbr := indexer.GetReverseIndexDB()
		if dbr != nil {
			dbr.Close()
		}
	}

}

// 从文本hash得到要分配到的 shard
func (engine *Engine) getShard(hash uint32) int {
	return int(hash - hash/uint32(engine.initOptions.NumShards)*
		uint32(engine.initOptions.NumShards))
}
