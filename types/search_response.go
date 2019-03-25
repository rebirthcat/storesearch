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

package types

import "sync"

// BaseResp search response options
type BaseResp struct {
	// 搜索用到的关键词
	Tokens []string

	// 搜索是否超时。超时的情况下也可能会返回部分结果
	Timeout bool

	// 搜索到的文档个数。注意这是全部文档中满足条件的个数，可能比返回的文档数要大
	NumDocs int
}

// SearchResp search response options
type SearchResp struct {
	BaseResp
	// 搜索到的文档，已排序
	Docs interface{}
}


// SearchID search response options
type SearchID struct {
	BaseResp
	// 搜索到的文档，已排序
	Docs []ScoredID
}


/*
  ______   .__   __.  __      ____    ____  __   _______
 /  __  \  |  \ |  | |  |     \   \  /   / |  | |       \
|  |  |  | |   \|  | |  |      \   \/   /  |  | |  .--.  |
|  |  |  | |  . `  | |  |       \_    _/   |  | |  |  |  |
|  `--'  | |  |\   | |  `----.    |  |     |  | |  '--'  |
 \______/  |__| \__| |_______|    |__|     |__| |_______/

*/

// ScoredID scored doc only id
type ScoredID struct {
	DocId string

	//BM25 float32
	// 文档的打分值
	Scores float32

	// TokenProximity 关键词在文档中的紧邻距离，
	// 紧邻距离的含义见 computeTokenProximity 的注释。
	// 仅当索引类型为 LocsIndex 时返回有效值。
	TokenProximity int32


	// 用于生成摘要的关键词在文本中的字节位置，
	// 该切片长度和 SearchResp.Tokens 的长度一样
	// 只有当 IndexType == LocsIndex 时不为空
	TokenSnippetLocs []int32

	// 关键词出现的位置
	// 只有当 IndexType == LocsIndex 时不为空
	TokenLocs [][]int32
}

// ScoredIDs 为了方便排序
type ScoredIDs []*ScoredID

func (docs ScoredIDs) Len() int {
	return len(docs)
}

func (docs ScoredIDs) Swap(i, j int) {
	docs[i], docs[j] = docs[j], docs[i]
}

func (docs ScoredIDs) Less(i, j int) bool {

	return docs[i].Scores > docs[j].Scores
}

type HeapNode struct {
	ScoreObj *ScoredID
	ShareNum int
	IndexPointer int
}

type NodeHeap struct {
	Arr []HeapNode
	IsSmallTop bool
}


func (h NodeHeap) Len() int {
	return len(h.Arr)
}

func (h NodeHeap) Swap(i, j int) {
	h.Arr[i],h.Arr[j]=h.Arr[j],h.Arr[i]
}

func (h NodeHeap)Less(i,j int) bool {
	if h.IsSmallTop {
		return h.Arr[i].ScoreObj.Scores<h.Arr[j].ScoreObj.Scores
	}else {
		return h.Arr[i].ScoreObj.Scores>h.Arr[j].ScoreObj.Scores
	}

}

func (h *NodeHeap) Push(x interface{}) {
	h.Arr=append(h.Arr,x.(HeapNode))
}

func (h *NodeHeap)Pop()interface{}  {
	old:=h.Arr
	n:=len(old)
	if n == 0 {
		return nil
	}
	x:=old[n-1]
	h.Arr=old[0:n-1]
	return x
}

type OutputPage struct {
	PageSize int
	PageNum   int
}




var ScoreIDPool=&sync.Pool{
	New: func() interface{} {
		return new(ScoredID)
	},
}


