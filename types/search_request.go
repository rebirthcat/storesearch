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

import (
	"github.com/rebirthcat/riot/distscore"
	"github.com/rebirthcat/riot/geofilter"
)

// SearchReq search request options
type SearchReq struct {
	// 搜索的短语（必须是 UTF-8 格式），会被分词
	// 当值为空字符串时关键词会从下面的 Tokens 读入
	Text string

	// 关键词（必须是 UTF-8 格式），当 Text 不为空时优先使用 Text
	// 通常你不需要自己指定关键词，除非你运行自己的分词程序
	Tokens []string

	// 文档标签（必须是 UTF-8 格式），标签不存在文档文本中，
	// 但也属于搜索键的一种
	Labels []string


	// 当不为 nil 时，仅从这些 DocIds 包含的键中搜索（忽略值）
	DocIds map[string]bool

	// 自定义评分接口
	//ScoringCriteria ScoringCriteria
	DistScoreCriteria distscore.DistScoreCriteria

	//是否倒序排序
	OrderReverse bool

	//过滤选项
	GeoFilter   geofilter.GeoFilterCriteria

	// 超时，单位毫秒（千分之一秒）。此值小于等于零时不设超时。
	// 搜索超时的情况下仍有可能返回部分排序结果。
	Timeout int

	// 设为 true 时仅统计搜索到的文档个数，不返回具体的文档
	CountDocsOnly bool

	// 不排序，对于可在引擎外部（比如客户端）排序情况适用
	// 对返回文档很多的情况打开此选项可以有效节省时间
	Orderless bool

	//Page *OutputPage
	//指的是所有数组中最后一个元素的数组下标值，如果不设置默认输出所有
	MaxOutputNum int
}

