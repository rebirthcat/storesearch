package core

import (
	"github.com/rebirthcat/storesearch/store"
	"log"
	"sync"
)





type StoreForwardIndexReq struct {
	DocID string
	Remove bool
	Field *DocField
}


//在线持久化请求结构
type StoreReverseIndexReq struct {
	Token string
	Indices *KeywordIndices
}


func (indexer *Indexer) GetForwardIndexDB() store.Store {
	return indexer.dbforwardIndex
}

func (indexer *Indexer)GetReverseIndexDB()store.Store  {
	return indexer.dbRevertIndex
}

func (indexer *Indexer) OpenForwardIndexDB(dbPath string,StoreEngine string)  {
	var erropen error
	indexer.dbforwardIndex, erropen= store.OpenStore(dbPath, StoreEngine)
	if indexer.dbforwardIndex == nil || erropen != nil {
		log.Fatal("Unable to open database ", dbPath, ": ", erropen)
	}
}

func (indexer *Indexer)OpenReverseIndexDB(dbPath string,StoreEngine string)  {
	var erropen error
	indexer.dbRevertIndex,erropen=store.OpenStore(dbPath,StoreEngine)
	if indexer.dbRevertIndex==nil||erropen!=nil {
		log.Fatal("Unable to open database ", dbPath, ": ", erropen)
	}
}


//系统启动时recover索引
func (indexer *Indexer)StoreRecoverForwardIndex(docNumber uint64, wg *sync.WaitGroup)  {
	//indexer中的字段
	if indexer.dbforwardIndex==nil {
		log.Fatalf("indexer %v dbforward is not open",indexer.shardNumber)
	}
	indexer.dbforwardIndex.ForEach(func(k, v []byte) error {
		docID := string(k)
		indexer.tableLock.docsState[docID]=0
		field:=&DocField{}
		field.Unmarshal(v)
		indexer.tableLock.totalTokenLen+=field.DocTokenLen
		indexer.tableLock.forwardtable[docID]=field
		indexer.tableLock.numDocs++
		return nil
	})
	//恢复indexer 中tableLock部分字段
	log.Printf("indexer%v forwardindex recover finish",indexer.shardNumber)
	if wg!=nil {
		wg.Done()
	}
}



func (indexer *Indexer)StoreRecoverReverseIndex(tokenNumber uint64, wg *sync.WaitGroup)  {

	if indexer.dbRevertIndex==nil {
		log.Fatalf("indexer %v dbreverse is not open",indexer.shardNumber)
	}
	indexer.dbRevertIndex.ForEach(func(k, v []byte) error {
		indices:=&KeywordIndices{}
		indices.Unmarshal(v)
		indexer.tableLock.table[string(k)]=indices
		return nil
	})
	log.Printf("indexer%v reverseindex recover finish",indexer.shardNumber)
	if wg!=nil {
		wg.Done()
	}

}


//系统启动时rebuild索引
func (indexer *Indexer)StoreForwardIndexOneTime(wg *sync.WaitGroup)  {
	if indexer.dbforwardIndex==nil {
		log.Fatalf("indexer %v dbforward is not open",indexer.shardNumber)
	}
	for docId,docField:=range indexer.tableLock.forwardtable{

		buf,_:=docField.Marshal(nil)
		indexer.dbforwardIndex.Set([]byte(docId), buf)
		//atomic.AddUint64(&indexer.numDocsStore, 1)
	}
	if wg!=nil {
		wg.Done()
	}
}

func (indexer *Indexer)StoreReverseIndexOneTime(wg *sync.WaitGroup)  {
	if indexer.dbRevertIndex==nil {
		log.Fatalf("indexer %v dbreverse is not open",indexer.shardNumber)
	}
	for token,indices:=range indexer.tableLock.table{
		buf,_:=indices.Marshal(nil)
		indexer.dbRevertIndex.Set([]byte(token),buf)
	}
	if wg!=nil {
		wg.Done()
	}
}

func (indexer *Indexer)StoreUpdateBegin()  {
	indexer.storeUpdateBegin=true
}


//系统正常运行中动态的添加索引的持久化
func (indexer *Indexer)StoreUpdateForWardIndexWorker()  {
	if indexer.dbforwardIndex==nil {
		log.Fatalf("indexer %v dbforward is not open",indexer.shardNumber)
	}

	for {
	 	request := <-indexer.storeUpdateForwardIndexChan
	 	//如果传过来的持久化请求中的DocTokenLen小于0,则是删除请求，即从RemoveDocs（）函数中传过来的
	 	if request.Remove {
	 		indexer.dbforwardIndex.Delete([]byte(request.DocID))
	 		//atomic.AddUint64(&indexer.numDocsStore,^uint64(1-1))
	 		continue
	 	}else {
			buf,_:=request.Field.Marshal(nil)
			indexer.dbforwardIndex.Set([]byte(request.DocID), buf)
			//atomic.AddUint64(&indexer.numDocsStore, 1)
		}
	}
}

func (indexer *Indexer) StoreUpdateReverseIndexWorker() {
	if indexer.dbRevertIndex==nil {
		log.Fatalf("indexer %v dbreverse is not open",indexer.shardNumber)
	}
	for {
		request := <-indexer.storeUpdateReverseIndexChan
		buf,_:=request.Indices.Marshal(nil)
		indexer.dbRevertIndex.Set([]byte(request.Token),buf)
	}
}


