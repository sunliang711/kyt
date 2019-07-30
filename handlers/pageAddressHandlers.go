package handlers

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/sunliang711/kyt/utils"
	"github.com/sunliang711/kyt/models"
	"log"
	"math"
	"net/http"
	"sort"
)

func addressIdentity(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	queryValue := query.Get("address")
	if len(queryValue) == 0 {
		utils.JsonResponse(resp{1, "empty query parameter", nil}, w)
		return
	}

	rows, err := models.DB.Query("select idtag from addresses where address = ? ", queryValue)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		return
	}
	defer rows.Close()
	var idtag string
	if rows.Next() {
		rows.Scan(&idtag)
	}
	if len(idtag) == 0 {
		idtag = "Unknown"
	}
	utils.JsonResponse(resp{0, "OK", idtag}, w)

}

type IDWithValue struct {
	TxID  int64
	Value int64
}

type IDWithValueSlice []*IDWithValue

func (s IDWithValueSlice) Less(i, j int) bool {
	return s[i].TxID < s[j].TxID
}

func (s IDWithValueSlice) Len() int {
	return len(s)
}

func (s IDWithValueSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func addressRadar(w http.ResponseWriter, req *http.Request) {
	//select addrID from addresses where address = ADDRESS;

	//select txID from txin where addrID = ADDRID
	//select txID from txout where addrID = ADDRID

	//select blockID from txinfo where txID = TXID

	//select timestamp from blockinfo where blockID = BLOCKID

	//OR ---------------------------------------------------------------------------------------------
	//input txID  :select a.txID from txin a join addresses b on (a.addrID = b.addrID) where b.address = ADDRESS;
	//output txID :select a.txID from txout a join addresses b on (a.addrID = b.addrID) where b.address = ADDRESS;

	//select a.timestamp from blockinfo a join txinfo b on (a.blockID = b.blockID) where b.txID = TXID;
	//排序timestamp，最大值减最小值，就是active life,active life可以换算成总的周数

	//count(input txID) + count(output txID) 就是所有交易数，除以总的周数就是transaction frequency
	query := req.URL.Query()
	address := query.Get("address")
	if len(address) == 0 {
		utils.JsonResponse(resp{1, "empty query parameter", nil}, w)
		return
	}

	log.Printf("Got parameter address: %v\n", address)
	///////////////////////////////////////////////////////////////////////////////////////////////
	//Get balanceMax
	///////////////////////////////////////////////////////////////////////////////////////////////
	//根据address到txin里拿所有该address对应的txID和该地址消耗的value
	sql := "select a.txID,a.value from txin a join addresses b on (a.addrID = b.addrID) where b.address = ?"
	logrus.Debugf("Before query: %v", sql)
	rows, err := models.DB.Query(sql, address)
	logrus.Debugf("After query: %v", sql)
	if err != nil {
		utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select a.txID,a.value from txin a join addresses b on (a.addrID = b.addrID) where b.address = %v", address), nil}, w)
		return
	}
	var (
		txID         int64
		inputTxIDs   []int64
		outputTxIDs  []int64
		value        int64
		IDWithValues IDWithValueSlice
	)

	for rows.Next() {
		rows.Scan(&txID, &value)
		inputTxIDs = append(inputTxIDs, txID)
		IDWithValues = append(IDWithValues, &IDWithValue{txID, -value})
	}
	rows.Close()
	log.Printf("txin,id with values: %v\n", IDWithValues)
	for _, v := range IDWithValues {
		log.Printf("%v", *v)
	}
	//根据address到txout里拿所有该address对应的txID收到的value
	sql = "select a.txID,a.value from txout a join addresses b on (a.addrID = b.addrID) where b.address = ?"
	logrus.Debugf("Before query: %v", sql)
	rows, err = models.DB.Query(sql, address)
	logrus.Debugf("After query: %v", sql)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		return
	}
	for rows.Next() {
		rows.Scan(&txID, &value)
		outputTxIDs = append(outputTxIDs, txID)
		IDWithValues = append(IDWithValues, &IDWithValue{txID, value})
	}
	rows.Close()
	log.Printf("txout,id with values: %v\n", IDWithValues)
	for _, v := range IDWithValues {
		log.Printf("%v", *v)
	}
	sort.Sort(IDWithValues)
	log.Printf("after sort,id with values: %v\n", IDWithValues)
	for _, v := range IDWithValues {
		log.Printf("%v", *v)
	}

	resultSlice := make([]int64, len(IDWithValues))
	for i := range resultSlice {
		if i == 0 {
			resultSlice[i] = IDWithValues[i].Value
		} else {
			resultSlice[i] = resultSlice[i-1] + IDWithValues[i].Value
		}
	}
	log.Printf("resultSlice: %v\n", resultSlice)
	var balanceMax int64
	for _, v := range resultSlice {
		balanceMax += v
	}
	log.Printf("balanceMax: %v\n", balanceMax)

	///////////////////////////////////////////////////////////////////////////////////////////////
	//Get communtyeSize
	///////////////////////////////////////////////////////////////////////////////////////////////
	var addrID int64
	var userID int64
	sql = "select addrID from addresses where address = ?"
	logrus.Debugf("Before query: %v", sql)
	rows, err = models.DB.Query(sql, address)
	logrus.Debugf("After query: %v", sql)
	if err != nil {
		utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select addrID from addresses where address = %v", address), nil}, w)
		return
	}
	if rows.Next() {
		rows.Scan(&addrID)
		rows.Close()
	} else {
		utils.JsonResponse(resp{1, fmt.Sprintf("no record in table addresses with address = %v", address), nil}, w)
		return
	}
	log.Printf("addrID: %v\n", addrID)
	sql = "select userID from entities where addrID = ?"
	logrus.Debugf("Before query: %v", sql)
	rows, err = models.DB.Query(sql, addrID)
	logrus.Debugf("After query: %v", sql)
	if err != nil {
		utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select userID from entities where addrID = %v", addrID), nil}, w)
		return
	}
	var communtyeSize int64
	if rows.Next() {
		rows.Scan(&userID)
		rows.Close()
		sql = "select count(*) from entities where userID = ?"
		logrus.Debugf("Before query: %v", sql)
		rows, err = models.DB.Query(sql, userID)
		logrus.Debugf("After query: %v", sql)
		if err != nil {
			utils.JsonResponse(resp{1, "internal db error", nil}, w)
			return
		}
		if rows.Next() {
			rows.Scan(&communtyeSize)
			rows.Close()
		}
		if communtyeSize == 0 {
			communtyeSize = 1
		}
	} else {
		communtyeSize = 1
	}

	log.Printf("communtye size: %v\n", communtyeSize)

	///////////////////////////////////////////////////////////////////////////////////////////////
	//Get tx volume
	///////////////////////////////////////////////////////////////////////////////////////////////
	var txVolume int64
	for _, v := range IDWithValues {
		txVolume += int64(math.Abs(float64(v.Value)))
	}
	log.Printf("tx volume: %v\n", txVolume)

	///////////////////////////////////////////////////////////////////////////////////////////////
	//Get active life
	///////////////////////////////////////////////////////////////////////////////////////////////
	//select a.timestamp from blockinfo a join txinfo b on (a.blockID = b.blockID) where txID in ();
	allTxIDs := append(inputTxIDs, outputTxIDs...)
	log.Printf("allTxIDs: %v\n", allTxIDs)
	params := []interface{}{}
	for _, v := range allTxIDs {
		params = append(params, v)
	}
	sql = "select a.timestamp from blockinfo a join txinfo b on (a.blockID = b.blockID) where txID in "
	logrus.Debugf("Before query: %v", sql)
	rows, err = models.DB.Query(sql+utils.MakeQuestion(len(params)), params...)
	if err != nil {
		utils.JsonResponse(resp{1, "intddernal db error", nil}, w)
		return
	}
	var (
		timestamp  int
		timestamps []int
	)
	for rows.Next() {
		rows.Scan(&timestamp)
		timestamps = append(timestamps, timestamp)
	}
	rows.Close()
	sort.Ints(timestamps)
	log.Printf("sorted timestamps: %v\n", timestamps)
	period := timestamps[len(timestamps)-1] - timestamps[0]
	log.Printf("active life in seconds: %v\n", period)

	activeWeek := int(math.Ceil(float64(period) / 60 / 60 / 24 / 7))
	log.Printf("active life in week: %v\n", activeWeek)

	///////////////////////////////////////////////////////////////////////////////////////////////
	//Get tx frequency
	///////////////////////////////////////////////////////////////////////////////////////////////
	weeks := activeWeek
	if activeWeek == 0 {
		weeks = 1
	}
	txFreq := len(allTxIDs) / weeks
	log.Printf("tx freq: %v\n", txFreq)

	//response to client
	response := struct {
		BalanceMax           int64 `json:"balance_max"`
		CommuntyeSize        int64 `json:"communtye_size"`
		TransactionVolume    int64 `json:"transaction_volume"`
		ActiveLife           int   `json:"active_life"`
		TransactionFrequency int   `json:"transaction_frequency"`
	}{balanceMax, communtyeSize, txVolume, period, txFreq}
	utils.JsonResponse(resp{0, "OK", response}, w)
}

type txIDRisktagValue struct {
	TxID    int64
	Risktag string
	Value   int
}

func addressSourceType(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	address := query.Get("address")
	if len(address) == 0 {
		utils.JsonResponse(resp{1, "empty query parameter", nil}, w)
		return
	}
	log.Printf("parameter address: %v\n", address)

	//select distinct a.txID from txout a join addresses b on (a.addrID=b.addrID) where b.address='1HSvEYwCFkDYDM3kGZgLbzSKiBdZ6QtvDC';
	//select distinct a.address from addresses a join txin b on (a.addrID = b.addrID) where b.txID in (txIDs);

	var (
		addrID int64
		//txID              int64
		//txIDs             []interface{}
		//sumValue          int
		//allValue          int
		//txIDRisktagValues []*txIDRisktagValue
		risktag string
		value   int
	)
	rows, err := models.DB.Query("select addrID from addresses where address = ?", address)
	if err != nil {
		utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select addrID from addresses where address = %v", address), nil}, w)
		return
	}
	if rows.Next() {
		rows.Scan(&addrID)
		rows.Close()
	} else {
		utils.JsonResponse(resp{1, fmt.Sprintf("no record in table addresses with address = %v", address), nil}, w)
		return
	}
	log.Printf("get addrID: %v\n", addrID)

	////select txID,addrID,sum(value) from txout where addrID=72 group by txID;
	//rows, err = db.Query("select txID,sum(value) from txout where addrID = ? group by txID", addrID)
	//if err != nil {
	//	utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select txID,sum(value) from txout where addrID = %v group by txID", addrID), nil}, w)
	//	return
	//}
	////根据txID到txhash中取得risktag

	//直接用这句
	//select a.risktag,value from txhash a join (select txID,sum(value)as value from txout where addrID = 72 group by txID) b on (a.txID=b.txID);
	rows2, err := models.DB.Query("select b.value,a.risktag from txhash a join (select txID,sum(value)as value from txout where addrID = ? group by txID) b on (a.txID=b.txID)", addrID)
	if err != nil {
		utils.JsonResponse(resp{1, "SQL error: select a.risktag,value from txhash...", nil}, w)
		return
	}
	sourceMap := make(map[string]float64)
	var total int
	for rows2.Next() {
		rows2.Scan(&value, &risktag)
		log.Printf("risktag:%v,value:%v.", risktag, value)
		total += value
		sourceMap[risktag] += float64(value)
	}
	for k, v := range sourceMap {
		if total == 0 {
			sourceMap[k] = 0
		} else {
			sourceMap[k] = v / float64(total)
		}
	}
	log.Printf("sourceMap:%v\n", sourceMap)

	//{"hack":xxx,"laundry":xxx,"spam":xxx,"ransomware":xxx,"fraud":xxx,"Gamble":xxx,"Unknown":xxx}
	sourceObj := struct {
		Hack       float64 `json:"hack"`
		Laundry    float64 `json:"laundry"`
		Spam       float64 `json:"spam"`
		Ransomware float64 `json:"ransomware"`
		Fraud      float64 `json:"fraud"`
		Gamble     float64 `json:"gamble"`
		Unknown    float64 `json:"unknown"`
	}{}
	for k, v := range sourceMap {
		switch k {
		case "hack":
			sourceObj.Hack += v
		case "laundry":
			sourceObj.Laundry += v
		case "spam":
			sourceObj.Spam += v
		case "ransomware":
			sourceObj.Ransomware += v
		case "fraud":
			sourceObj.Fraud += v
		case "gamble":
			sourceObj.Gamble += v
		case "":
			sourceObj.Unknown += v
		}
	}
	utils.JsonResponse(resp{0, "OK", sourceObj}, w)

}

func addressRiskLevel(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	queryValue := query.Get("address")
	if len(queryValue) == 0 {
		utils.JsonResponse(resp{1, "empty query parameter", nil}, w)
		return
	}

	rows, err := models.DB.Query("select risktag from addresses where address = ?", queryValue)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		return
	}
	var risktag string
	if rows.Next() {
		rows.Scan(&risktag)
		rows.Close()
	}
	utils.JsonResponse(resp{0, "OK", risktag}, w)

}

type hashTag struct {
	Hash string `json:"hash"`
	Tag  string `json:"tag"`
}

func addressBadTxList(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	address := query.Get("address")
	if len(address) == 0 {
		utils.JsonResponse(resp{1, "empty query parameter", nil}, w)
		return
	}

	//select a.txhash,a.risktag from txhash a join txout b on (a.txID=b.txID) where b.addrID=(select addrID from addresses where address='1HSvEYwCFkDYDM3kGZgLbzSKiBdZ6QtvDC')
	//and a.risktag is not null;
	sql := "select a.txhash,a.risktag from txhash a join txout b on (a.txID=b.txID) where b.addrID=(select addrID from addresses where address=?) and a.risktag is not null;"
	rows, err := models.DB.Query(sql, address)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		log.Printf("SQL error: %v", sql)
		return
	}
	var (
		txhash   string
		risktag  string
		hashTags []*hashTag
	)

	for rows.Next() {
		rows.Scan(&txhash, &risktag)
		log.Printf("txhash: %v,risktag: %v", txhash, risktag)
		hashTags = append(hashTags, &hashTag{txhash, risktag})
	}
	rows.Close()
	utils.JsonResponse(resp{0, "OK", hashTags}, w)

}
