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

package storesearch

import (
	"github.com/rebirthcat/storesearch/distscore"
	"github.com/rebirthcat/storesearch/geofilter"
	"github.com/rebirthcat/storesearch/types"

)

type indexerAddDocReq struct {
	doc         *types.DocIndex
	forceUpdate bool
}

type indexerLookupReq struct {
	countDocsOnly bool
	tokens        []string
	labels        []string

	docIds           map[string]bool
	orderReverse     bool
	disScoreCriteria distscore.DistScoreCriteria
	geoFilter        geofilter.GeoFilterCriteria
	rankerReturnChan chan rankerReturnReq
	orderless        bool
}

type indexerRemoveDocReq struct {
	docId       string
	forceUpdate bool
}



func (engine *Engine) indexerAddDoc(shard int) {
	for {
		request := <-engine.indexerAddDocChans[shard]
		engine.indexers[shard].AddDocToCache(request.doc, request.forceUpdate)
	}
}

func (engine *Engine) indexerRemoveDoc(shard int) {
	for {
		request := <-engine.indexerRemoveDocChans[shard]
		engine.indexers[shard].RemoveDocToCache(request.docId, request.forceUpdate)
	}
}

func (engine *Engine) indexerLookup(shard int) {
	for {
		request := <-engine.indexerLookupChans[shard]

		docs, numDocs := engine.indexers[shard].Lookup(
			request.tokens, request.labels,
			request.docIds, request.countDocsOnly, request.disScoreCriteria, request.geoFilter, request.orderReverse)

		request.rankerReturnChan <- rankerReturnReq{
			docs: docs, numDocs: numDocs}

	}
}
