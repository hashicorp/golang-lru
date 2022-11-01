package simplelru

type BetterMap struct {
	firstMap        map[interface{}]interface{}
	secondMap       map[interface{}]interface{}
	deleteNumThresh int32
	totalDeleteNum  int32
	useMapIndex     int32
}

func NewBetterMap(deleteNumThresh int32) *BetterMap {
	return &BetterMap{
		firstMap:        make(map[interface{}]interface{}, 0),
		secondMap:       make(map[interface{}]interface{}, 0),
		deleteNumThresh: deleteNumThresh,
		totalDeleteNum:  0,
		useMapIndex:     1,
	}
}

func (bmap *BetterMap) Set(key interface{}, value interface{}) {
	if bmap.useMapIndex == 1 {
		bmap.firstMap[key] = value
	} else {
		bmap.secondMap[key] = value
	}

}

func (bmap *BetterMap) GetValue(key interface{}) (interface{}, bool) {
	if value, ok := bmap.firstMap[key]; ok {
		return value, ok
	}

	value, ok := bmap.secondMap[key]

	return value, ok

}

func (bmap *BetterMap) GetValues() []interface{} {
	values := make([]interface{}, 0)
	if bmap.useMapIndex == 1 {
		for _, value := range bmap.firstMap {
			values = append(values, value)
		}
	} else {
		for _, value := range bmap.secondMap {
			values = append(values, value)
		}
	}

	return values
}

func (bmap *BetterMap) GetMap() map[interface{}]interface{} {
	if bmap.useMapIndex == 1 {
		return bmap.firstMap
	}

	return bmap.secondMap
}

func (bmap *BetterMap) GetLen() int {
	if bmap.useMapIndex == 1 {
		return len(bmap.firstMap)
	}

	return len(bmap.secondMap)
}

func (bmap *BetterMap) Purge() {
	bmap.firstMap = nil
	bmap.secondMap = nil
}

func (bmap *BetterMap) DelKey(key interface{}) {
	if bmap.useMapIndex == 1 {
		delete(bmap.firstMap, key)
		bmap.totalDeleteNum++
		if bmap.totalDeleteNum >= bmap.deleteNumThresh {
			bmap.secondMap = make(map[interface{}]interface{}, len(bmap.firstMap))
			for i, v := range bmap.firstMap {
				bmap.secondMap[i] = v
			}

			bmap.useMapIndex = 2
			bmap.totalDeleteNum = 0
			bmap.firstMap = nil
		}
	} else {
		delete(bmap.secondMap, key)
		bmap.totalDeleteNum++
		if bmap.totalDeleteNum >= bmap.deleteNumThresh {
			bmap.firstMap = make(map[interface{}]interface{}, len(bmap.secondMap))
			for i, v := range bmap.secondMap {
				bmap.firstMap[i] = v
			}

			bmap.useMapIndex = 1
			bmap.totalDeleteNum = 0
			bmap.secondMap = nil
		}
	}

}
