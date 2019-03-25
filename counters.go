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

package riot

// NumTokenAdded added token index number
func (engine *Engine) NumTokensAdded() uint64 {
	var num uint64
	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		num += engine.indexers[shard].GetNumTotalTokenLen()
	}
	return num
}

// NumIndexed documents indexed number
func (engine *Engine) NumDocsIndexed() uint64 {
	var num uint64
	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
		num += engine.indexers[shard].GetNumDocs()
	}
	return num
}

//func (engine *Engine) NumDocsIndexedStore() uint64 {
//	var num uint64
//	for shard := 0; shard < engine.initOptions.NumShards; shard++ {
//		num += engine.indexers[shard].GetNumDocsStore()
//	}
//	return num
//}
