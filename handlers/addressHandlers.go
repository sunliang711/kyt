package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/sunliang711/kyt/models"
	"github.com/sunliang711/kyt/utils"
)

type radarSample struct {
	BalanceMax           []int     `json:"balance_max"`
	EntitySize           []int     `json:"entity_size"`
	TotalOutput          []int     `json:"total_output"`
	TransactionFrequency []int     `json:"transaction_frequency"`
	VarTxamount          []float64 `json:"var_txamount"`
	TotalInput           []int     `json:"total_input"`
	TransactionVolume    []int     `json:"transaction_volume"`
	ActiveLife           []int     `json:"active_life"`
}

const (
	radarFile = "sample_radar_sorted.txt"
)

var (
	radar radarSample
)

func init() {
	bs, err := ioutil.ReadFile(radarFile)
	if err != nil {
		logrus.Fatalf("read filer %s error: %s", radarFile, err)
	}
	err = json.Unmarshal(bs, &radar)
	if err != nil {
		logrus.Fatalf("Unmarshal sample radar error: %s", err)
	}
	logrus.Debugf("sample radar: %+v", radar)
}

// Identity TODO
// 2019/09/23 15:35:11
type Identity struct {
	Type string `json:"type"`
	Desc string `json:"desc"`
}

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
	var identities []Identity
	if len(idtag) == 0 {
		idtag = "Unknown"
	} else {
		var ids [][]string
		idtag = strings.ReplaceAll(idtag, "'", "\"")
		json.Unmarshal([]byte(idtag), &ids)
		for _, i := range ids {
			iden := Identity{i[0], i[1]}
			identities = append(identities, iden)
		}
	}
	utils.JsonResponse(resp{0, "OK", identities}, w)

}

// IDWithValue TODO
type IDWithValue struct {
	TxID  int64
	Value int64
}

// IDWithValueSlice TODO
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
		logrus.Debugf("id with value: %v %v", txID, -value)
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
		logrus.Debugf("id with value: %v %v", txID, value)
	}

	sort.Sort(IDWithValues)
	for _, v := range IDWithValues {
		logrus.Debugf("after sort,id with valud: %v", *v)
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
	balanceMax = resultSlice[len(resultSlice)-1]
	// for _, v := range resultSlice {
	// 	balanceMax += v
	// }
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
	// log.Printf("allTxIDs: %v\n", allTxIDs)
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
	// log.Printf("sorted timestamps: %v\n", timestamps)
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

	///////////////////////////////////////////////////////////////////////////////////////////////
	//Get percents
	///////////////////////////////////////////////////////////////////////////////////////////////
	var (
		percentBalanceMax    float64
		percentCommuntyeSize float64
		percentTxVolume      float64
		percentPeriod        float64
		percentTxFreq        float64
	)

	//percentBalanceMax
	index := sort.Search(len(radar.BalanceMax), func(i int) bool {
		return int64(radar.BalanceMax[i]) <= balanceMax
	})
	if index == len(radar.BalanceMax) {
		percentBalanceMax = 1.0
	} else {
		fmt.Printf("BalanceMax index: %v\n", index)
		percentBalanceMax = float64(index) / float64(len(radar.BalanceMax))
	}

	//percentCommuntyeSize
	index = sort.Search(len(radar.EntitySize), func(i int) bool {
		return int64(radar.EntitySize[i]) <= communtyeSize
	})
	if index == len(radar.EntitySize) {
		percentCommuntyeSize = 1.0
	} else {
		fmt.Printf("EntitySize index: %v\n", index)
		percentCommuntyeSize = float64(index) / float64(len(radar.EntitySize))
	}

	//percentTxVolume
	index = sort.Search(len(radar.VarTxamount), func(i int) bool {
		return int64(radar.VarTxamount[i]) <= txVolume
	})

	if index == len(radar.VarTxamount) {
		percentTxVolume = 1.0
	} else {
		fmt.Printf("VarTxamount index: %v\n", index)
		percentTxVolume = float64(index) / float64(len(radar.VarTxamount))
	}

	//percentPeriod
	index = sort.Search(len(radar.ActiveLife), func(i int) bool {
		return radar.ActiveLife[i] <= period
	})
	if index == len(radar.ActiveLife) {
		percentPeriod = 1.0
	} else {
		fmt.Printf("ActiveLife index: %v\n", index)
		percentPeriod = float64(index) / float64(len(radar.ActiveLife))
	}

	//percentTxFreq
	index = sort.Search(len(radar.TransactionFrequency), func(i int) bool {
		return radar.TransactionFrequency[i] <= txFreq
	})
	if index == len(radar.TransactionFrequency) {
		percentTxFreq = 1.0
	} else {
		fmt.Printf("TransactionFrequency index: %v\n", index)
		percentTxFreq = float64(index) / float64(len(radar.TransactionFrequency))
	}
	//percentBalanceMax    float64
	//percentCommuntyeSize float64
	//percentTxVolume      float64
	//percentPeriod        float64
	//percentTxFreq        float64

	//response to client
	response := struct {
		BalanceMax           int64   `json:"balance_max"`
		CommuntyeSize        int64   `json:"communtye_size"`
		TransactionVolume    int64   `json:"transaction_volume"`
		ActiveLife           int     `json:"active_life"`
		TransactionFrequency int     `json:"transaction_frequency"`
		PercentBalanceMax    float64 `json:"percent_balance_max"`
		PercentCommuntyeSize float64 `json:"percent_communtye_size"`
		PercentTxVolume      float64 `json:"percent_tx_volume"`
		PercentPeriod        float64 `json:"percent_period"`
		PercentTxFreq        float64 `json:"percent_tx_freq"`
	}{balanceMax,
		communtyeSize,
		txVolume,
		period,
		txFreq,
		percentBalanceMax,
		percentCommuntyeSize,
		percentTxVolume,
		percentPeriod,
		percentTxFreq,
	}
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
		value int
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
	rows2, err := models.DB.Query("select b.value,a.risktag,a.txID from txhash a join (select txID,sum(value)as value from txout where addrID = ? group by txID) b on (a.txID=b.txID)", addrID)
	if err != nil {
		utils.JsonResponse(resp{1, "SQL error: select a.risktag,value from txhash...", nil}, w)
		return
	}
	sourceMap := make(map[string]float64)
	var total int
	var txID int
	for rows2.Next() {
		var risktag string
		rows2.Scan(&value, &risktag, &txID)
		log.Printf("risktag:%v,value:%v,txID: %v", risktag, value, txID)
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
		case "Hack":
			sourceObj.Hack += v
		case "Laundry":
			sourceObj.Laundry += v
		case "Spam":
			sourceObj.Spam += v
		case "Ransomware":
			sourceObj.Ransomware += v
		case "Fraud":
			sourceObj.Fraud += v
		case "Gamble":
			sourceObj.Gamble += v
		case "":
			sourceObj.Unknown += v
		}
	}
	log.Printf("sourceObj: %v\n", sourceObj)
	utils.JsonResponse(resp{0, "OK", sourceObj}, w)

}

var riskLevelMap = map[string]int{
	"Low":        0,
	"Suspicious": 1,
	"High":       2,
	"Very High":  3,
	"Black":      4,
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
	riskNo, ok := riskLevelMap[risktag]
	if !ok {
		riskNo = 1
	}
	utils.JsonResponse(resp{0, "OK", riskNo}, w)

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
	sql2 := "select distinct a.txhash,a.risktag from txhash a join txin b on (a.txID=b.txID) where b.addrID=(select addrID from addresses where address=?) and a.risktag is not null;"
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

	rows2, err := models.DB.Query(sql2, address)
	for rows2.Next() {
		rows2.Scan(&txhash, &risktag)
		log.Printf("txhash: %v,risktag: %v", txhash, risktag)
		hashTags = append(hashTags, &hashTag{txhash, risktag})
	}

	for rows.Next() {
		rows.Scan(&txhash, &risktag)
		log.Printf("txhash: %v,risktag: %v", txhash, risktag)
		hashTags = append(hashTags, &hashTag{txhash, risktag})
	}
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		log.Printf("SQL error: %v", sql)
		return
	}
	rows.Close()
	utils.JsonResponse(resp{0, "OK", hashTags}, w)

}

//根据给定的地址，找到所有该地址参与的交易，然后把这些交易中标记为Black和xx的所有交易找出来，把所有交易的srcs和dests找出来
func addressBadTxGraph(w http.ResponseWriter, req *http.Request) {
}
